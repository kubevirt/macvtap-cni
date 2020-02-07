package main

import (
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	macvtap_cni "github.com/kubevirt/macvtap-cni/pkg/cni"
)

func main() {
	skel.PluginMain(macvtap_cni.CmdAdd, macvtap_cni.CmdCheck, macvtap_cni.CmdDel, version.All, bv.BuildString("macvtap"))
}
