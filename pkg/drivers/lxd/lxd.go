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
	"io/ioutil"
	"strings"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/minikube/pkg/minikube/constants"
	"k8s.io/minikube/pkg/util"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"github.com/pkg/errors"
	pkgdrivers "k8s.io/minikube/pkg/drivers"
)

const (
	dockerPGPKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
YT90qFF93M3v01BbxP+EIY2/9tiIPbrd
=0YYh
-----END PGP PUBLIC KEY BLOCK-----`
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
			SSHUser:     "ubuntu",
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
	if _, err := d.getConnection(); err != nil {
		return err
	}
	return nil
}

func (d *Driver) GetURL() (string, error) {
	ip, err := d.GetIP()
	if err != nil {
		return "", errors.Wrap(err, "getting URL, could not get IP")
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
	return "", errors.New("machine has no global IPv4 addresses")
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

	// XXX Wait for cloud-init to finish

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
	log.Info("Creating ssh key...")
	if err := ssh.GenerateSSHKey(d.GetSSHKeyPath()); err != nil {
		return err
	}

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

	userData, err := d.makeUserData()
	if err != nil {
		return err
	}

	op, err := conn.CreateContainerFromImage(cloudImages, *image, api.ContainersPost{
		Name: d.MachineName,
		ContainerPut: api.ContainerPut{
			Profiles: []string{"default", "minikube"}, // XXX make configurable
			Config:   map[string]string{"user.user-data": userData},
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

func (d *Driver) makeUserData() (string, error) {
	publicKeyData, err := ioutil.ReadFile(d.GetSSHKeyPath() + ".pub")
	if err != nil {
		return "", err
	}
	// TODO(axw) add user-data to install kubeadm, etc.
	userDataMap := map[string]interface{}{
		"users": []map[string]interface{}{{
			"name":                "ubuntu",
			"home":                "/home/ubuntu",
			"shell":               "/bin/bash",
			"ssh_authorized_keys": []string{strings.TrimSpace(string(publicKeyData))},
			"sudo":                "ALL=(ALL) NOPASSWD:ALL",
		}},
		/*
			"apt": map[string]interface{}{
				"sources": map[string]interface{}{
					"docker.list": map[string]interface{}{
						"source": "deb https://download.docker.com/linux/ubuntu bionic stable",
						"key":    dockerPGPKey,
					},
				},
			},
			"packages": []string{"docker-ce"},
		*/
		"runcmd": []string{
			"curl https://get.docker.com | bash",
		},
	}
	out, err := yaml.Marshal(userDataMap)
	if err != nil {
		return "", err
	}
	return "#cloud-config\n" + string(out), nil
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
