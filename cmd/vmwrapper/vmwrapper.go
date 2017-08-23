/*
Copyright 2016-2017 Mirantis

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

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/golang/glog"

	"github.com/Mirantis/virtlet/pkg/tapmanager"
)

const (
	fdSocketPath    = "/var/lib/virtlet/tapfdserver.sock"
	defaultEmulator = "/usr/bin/qemu-system-x86_64" // FIXME
	emulatorVar     = "VIRTLET_EMULATOR"
	netKeyEnvVar    = "VIRTLET_NET_KEY"
)

func main() {
	// configure glog (apparently no better way to do it ...)
	flag.CommandLine.Parse([]string{"-v=3", "-alsologtostderr=true"})

	emulator := os.Getenv(emulatorVar)
	emulatorArgs := os.Args[1:]
	var netArgs []string
	if emulator == "" {
		// this happens during 'qemu -help' invocation by libvirt
		// (capability check)
		emulator = defaultEmulator
	} else {
		netFdKey := os.Getenv(netKeyEnvVar)

		c := tapmanager.NewFDClient(fdSocketPath)
		if err := c.Connect(); err != nil {
			glog.Errorf("Can't connect to fd server: %v", err)
			os.Exit(1)
		}
		tapFd, hwAddr, err := c.GetFD(netFdKey)
		if err != nil {
			glog.Errorf("Failed to obtain tap fd for key %q: %v", netFdKey, err)
			os.Exit(1)
		}

		netArgs = []string{
			"-netdev",
			fmt.Sprintf("tap,id=tap0,fd=%d", tapFd),
			"-device",
			"virtio-net-pci,netdev=tap0,id=net0,mac=" + net.HardwareAddr(hwAddr).String(),
		}
	}

	glog.V(0).Infof("executing emulator %q: args %#v", emulator, emulatorArgs)
	args := append([]string{emulator}, emulatorArgs...)
	if err := syscall.Exec(emulator, append(args, netArgs...), os.Environ()); err != nil {
		glog.Errorf("can't exec emulator %q: %v", emulator, err)
		os.Exit(1)
	}
}
