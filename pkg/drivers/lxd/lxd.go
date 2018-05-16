/*
Copyright 2018 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lxd

import (
	"fmt"

	"k8s.io/minikube/pkg/minikube/constants"
	"k8s.io/minikube/pkg/util"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/state"
	"github.com/kr/pretty"
	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"github.com/pkg/errors"
	pkgdrivers "k8s.io/minikube/pkg/drivers"
)

type Driver struct {
	*drivers.BaseDriver
	*pkgdrivers.CommonDriver

	// How much memory, in MB, to allocate to the VM
	Memory int

	// How many cpus to allocate to the VM
	CPU int

	// The name of the default network
	Network string

	// The name of the private network
	PrivateNetwork string

	// The size of the disk to be created for the VM, in MB
	DiskSize int

	// The randomly generated MAC Address
	// If empty, a random MAC will be generated.
	MAC string
}

const (
	defaultPrivateNetworkName = "minikube-net"
	defaultNetworkName        = "default"
)

func NewDriver(hostName, storePath string) *Driver {
	return &Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
			SSHUser:     "docker",
		},
		CommonDriver:   &pkgdrivers.CommonDriver{},
		CPU:            constants.DefaultCPUS,
		DiskSize:       util.CalculateDiskSizeInMB(constants.DefaultDiskSize),
		Memory:         constants.DefaultMemory,
		PrivateNetwork: defaultPrivateNetworkName,
		Network:        defaultNetworkName,
	}
}

func (d *Driver) getConnection() (lxd.ContainerServer, error) {
	conn, err := lxd.ConnectLXDUnix("", nil)
	if err != nil {
		return nil, errors.Wrap(err, "Error connecting to LXD")
	}
	return conn, nil
}

func (d *Driver) PreCommandCheck() error {
	conn, err := d.getConnection()
	if err != nil {
		return err
	}
	server, _, err := conn.GetServer()
	if err != nil {
		return err
	}
	pretty.Println(server) // XXX
	return nil
}

func (d *Driver) GetURL() (string, error) {
	ip, err := d.GetIP()
	if err != nil {
		return "", errors.Wrap(err, "getting URL, could not get IP")
	}
	if ip == "" {
		return "", nil
	}
	return fmt.Sprintf("tcp://%s:2376", ip), nil
}

func (d *Driver) GetState() (state.State, error) {
	conn, err := d.getConnection()
	if err != nil {
		return state.None, errors.Wrap(err, "getting connection")
	}

	s, _, err := conn.GetContainerState(d.MachineName)
	if err != nil {
		if isNotFound(err) {
			return state.None, nil
		}
		return state.None, errors.Wrap(err, "getting container")
	}

	switch s.StatusCode {
	case api.Starting, api.Started:
		return state.Starting, nil
	case api.Stopping:
		return state.Stopping, nil
	case api.Stopped:
		return state.Stopped, nil
	case api.Running:
		return state.Running, nil
	}
	return state.None, nil
}

func (d *Driver) GetIP() (string, error) {
	conn, err := d.getConnection()
	if err != nil {
		return "", err
	}
	s, _, err := conn.GetContainerState(d.MachineName)
	if err != nil {
		return "", errors.Wrap(err, "getting container")
	}
	net := s.Network["eth0"]
	for _, addr := range net.Addresses {
		if addr.Family != "inet" && addr.Scope != "global" {
			continue
		}
		return addr.Address, nil
	}
	return "", nil
}

func (d *Driver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

func (d *Driver) DriverName() string {
	return "lxd"
}

func (d *Driver) Kill() error {
	conn, err := d.getConnection()
	if err != nil {
		return err
	}
	op, err := conn.DeleteContainer(d.MachineName)
	if err != nil {
		return err
	}
	return op.Wait()
}

func (d *Driver) Restart() error {
	return pkgdrivers.Restart(d)
}

func (d *Driver) Start() error {
	// Start container
	log.Info("Starting container...")
	conn, err := d.getConnection()
	if err != nil {
		return err
	}

startContainer:
	for {
		s, etag, err := conn.GetContainerState(d.MachineName)
		if err != nil {
			return err
		}
		switch s.StatusCode {
		case api.Starting, api.Started, api.Running:
			// already started/starting/running
			break startContainer
		case api.Stopping, api.Stopped:
		}
		// TODO(axw) check for etag error below, and continue to top of loop.
		op, err := conn.UpdateContainerState(d.MachineName, api.ContainerStatePut{Action: "start"}, etag)
		if err != nil {
			return err
		}
		if err := op.Wait(); err != nil {
			return err
		}
		break
	}

	/*
		log.Info("Waiting to get IP...")
		for i := 0; i <= 40; i++ {
			ip, err := d.GetIP()
			if err != nil {
				return errors.Wrap(err, "getting ip during machine start")
			}
			if ip == "" {
				log.Debugf("Waiting for machine to come up %d/%d", i, 40)
				time.Sleep(3 * time.Second)
				continue
			}

			if ip != "" {
				log.Infof("Found IP for machine: %s", ip)
				d.IPAddress = ip
				break
			}
		}

		if d.IPAddress == "" {
			return errors.New("Machine didn't return an IP after 120 seconds")
		}
	*/

	log.Info("Waiting for SSH to be available...")
	if err := drivers.WaitForSSH(d); err != nil {
		d.IPAddress = ""
		return errors.Wrap(err, "SSH not available after waiting")
	}

	return nil
}

func (d *Driver) Create() error {
	log.Debug("Creating container...")
	conn, err := d.getConnection()
	if err != nil {
		return err
	}

	cloudImages, err := lxd.ConnectSimpleStreams("https://cloud-images.ubuntu.com/minimal/daily", nil)
	if err != nil {
		return errors.Wrap(err, "Error connecting to cloud-images.ubuntu.com")
	}
	alias, _, err := cloudImages.GetImageAlias("bionic")
	if err != nil {
		return err
	}
	image, _, err := cloudImages.GetImage(alias.Target)
	if err != nil {
		return err
	}

	// TODO(axw) add user-data to install kubeadm, etc.

	op, err := conn.CreateContainerFromImage(cloudImages, *image, api.ContainersPost{
		Name: d.MachineName,
		ContainerPut: api.ContainerPut{
			Profiles: []string{"default", "docker"}, // XXX make configurable
		},
	})
	if err != nil {
		return err
	}
	if err := op.Wait(); err != nil {
		return err
	}

	log.Debug("Finished creating machine, now starting machine...")
	return d.Start()
}

func (d *Driver) Stop() error {
	conn, err := d.getConnection()
	if err != nil {
		return err
	}
	for {
		s, etag, err := conn.GetContainerState(d.MachineName)
		if err != nil {
			return err
		}
		switch s.StatusCode {
		case api.Stopped:
			return nil
		case api.Stopping:
			// Wait until it's actually stopped.
			panic("stopping...") // XXX
		case api.Starting, api.Started, api.Running:
		}
		// TODO(axw) check for etag error below, and continue to top of loop.
		op, err := conn.UpdateContainerState(d.MachineName, api.ContainerStatePut{Action: "stop"}, etag)
		if err != nil {
			return err
		}
		return op.Wait()
	}
}

func (d *Driver) Remove() error {
	log.Debug("Removing machine...")
	conn, err := d.getConnection()
	if err != nil {
		return err
	}
	if err := d.Stop(); err != nil {
		return err
	}
	op, err := conn.DeleteContainer(d.MachineName)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}
	return op.Wait()
}

func isNotFound(err error) bool {
	return errors.Cause(err).Error() == "not found"
}
