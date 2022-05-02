// Copyright 2019 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cni

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"runtime"

	"github.com/kubevirt/macvtap-cni/pkg/util"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ipam"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
)

// A NetConf structure represents a Multus network attachment definition configuration
type NetConf struct {
	types.NetConf
	DeviceID string `json:"deviceID"`
	MTU      int    `json:"mtu,omitempty"`
}

// EnvArgs structure represents inputs sent from each VMI via environment variables
type EnvArgs struct {
	types.CommonArgs
	MAC types.UnmarshallableString `json:"mac,omitempty"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadConf(bytes []byte) (NetConf, string, error) {
	n := NetConf{}
	if err := json.Unmarshal(bytes, &n); err != nil {
		return n, "", fmt.Errorf("failed to load netconf: %v", err)
	}

	return n, n.CNIVersion, nil
}

func getEnvArgs(envArgsString string) (EnvArgs, error) {
	e := EnvArgs{}
	err := types.LoadArgs(envArgsString, &e)
	if err != nil {
		return e, err
	}
	return e, nil
}

// CmdAdd - CNI interface
func CmdAdd(args *skel.CmdArgs) error {
	var err error
	netConf, cniVersion, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}

	envArgs, err := getEnvArgs(args.Args)
	if err != nil {
		return err
	}

	var mac *net.HardwareAddr = nil
	if envArgs.MAC != "" {
		aMac, err := net.ParseMAC(string(envArgs.MAC))
		mac = &aMac
		if err != nil {
			return err
		}
	}

	isLayer3 := netConf.IPAM.Type != ""

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", netns, err)
	}

	// Delete link if err to avoid link leak in this ns
	defer func() {
		netns.Close()
		if err != nil {
			util.LinkDelete(netConf.DeviceID)
		}
	}()

	macvtapInterface, err := util.ConfigureInterface(netConf.DeviceID, args.IfName, mac, netConf.MTU, netns)
	if err != nil {
		return err
	}

	// Assume L2 interface only
	result := &current.Result{
		CNIVersion: cniVersion,
		Interfaces: []*current.Interface{macvtapInterface},
	}

	if isLayer3 {
		// run the IPAM plugin and get back the config to apply
		r, err := ipam.ExecAdd(netConf.IPAM.Type, args.StdinData)
		if err != nil {
			return err
		}

		// Invoke ipam del if err to avoid ip leak
		defer func() {
			if err != nil {
				ipam.ExecDel(netConf.IPAM.Type, args.StdinData)
			}
		}()

		// Convert whatever the IPAM result was into the current Result type
		ipamResult, err := current.NewResultFromResult(r)
		if err != nil {
			return err
		}

		if len(ipamResult.IPs) == 0 {
			return errors.New("IPAM plugin returned missing IP config")
		}

		result.IPs = ipamResult.IPs
		result.Routes = ipamResult.Routes
		result.DNS = ipamResult.DNS

		err = netns.Do(func(_ ns.NetNS) error {
			_, _ = sysctl.Sysctl(fmt.Sprintf("net/ipv4/conf/%s/arp_notify", args.IfName), "1")

			if err := ipam.ConfigureIface(args.IfName, ipamResult); err != nil {
				return err
			}
			return nil
		})
	}

	return types.PrintResult(result, cniVersion)
}

// CmdDel - CNI plugin Interface
func CmdDel(args *skel.CmdArgs) error {
	netConf, _, err := loadConf(args.StdinData)
	isLayer3 := netConf.IPAM.Type != ""

	if isLayer3 {
		err = ipam.ExecDel(netConf.IPAM.Type, args.StdinData)
		if err != nil {
			return err
		}
	}

	if args.Netns == "" {
		return nil
	}

	// There is a netns so try to clean up. Delete can be called multiple times
	// so don't return an error if the device is already removed.
	err = ns.WithNetNSPath(args.Netns, func(_ ns.NetNS) error {

		if err := ip.DelLinkByName(args.IfName); err != nil {
			if err != ip.ErrLinkNotFound {
				return err
			}
		}
		return nil
	})

	return err
}

// CmdCheck - CNI plugin Interface
func CmdCheck(args *skel.CmdArgs) error {
	return nil
}
