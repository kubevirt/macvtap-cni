package deviceplugin

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/kubevirt/macvtap-cni/pkg/util"
)

const (
	tapPath = "/dev/tap"
	// Interfaces will be named as <Name><suffix>[0-<Capacity>]
	suffix = "Mvp"
	// DefaultCapacity is the default when no capacity is provided
	DefaultCapacity = 100
	// DefaultMode is the default when no mode is provided
	DefaultMode = "bridge"
)

type macvtapDevicePlugin struct {
	Name        string
	LowerDevice string
	Mode        string
	Capacity    int
	// NetNsPath is the path to the network namespace the plugin operates in.
	NetNsPath   string
	stopWatcher chan struct{}
}

func NewMacvtapDevicePlugin(name string, lowerDevice string, mode string, capacity int, netNsPath string) *macvtapDevicePlugin {
	return &macvtapDevicePlugin{
		Name:        name,
		LowerDevice: lowerDevice,
		Mode:        mode,
		Capacity:    capacity,
		NetNsPath:   netNsPath,
		stopWatcher: make(chan struct{}),
	}
}

func (mdp *macvtapDevicePlugin) generateMacvtapDevices() []*pluginapi.Device {
	var macvtapDevs []*pluginapi.Device

	var capacity = mdp.Capacity
	if capacity <= 0 {
		capacity = DefaultCapacity
	}

	for i := 0; i < capacity; i++ {
		name := fmt.Sprint(mdp.Name, suffix, i)
		macvtapDevs = append(macvtapDevs, &pluginapi.Device{
			ID:     name,
			Health: pluginapi.Healthy,
		})
	}

	return macvtapDevs
}

func (mdp *macvtapDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	// Initialize two arrays, one for devices offered when lower device exists,
	// and no devices if lower device does not exist.
	allocatableDevs := mdp.generateMacvtapDevices()
	emptyDevs := make([]*pluginapi.Device, 0)

	emitResponse := func(lowerDeviceExists bool) {
		if lowerDeviceExists {
			glog.V(3).Info("LowerDevice exists, sending ListAndWatch response with available devices")
			s.Send(&pluginapi.ListAndWatchResponse{Devices: allocatableDevs})
		} else {
			glog.V(3).Info("LowerDevice does not exist, sending ListAndWatch response with no devices")
			s.Send(&pluginapi.ListAndWatchResponse{Devices: emptyDevs})
		}
	}

	didLowerDeviceExist := false
	onLowerDeviceEvent := func() {
		var doesLowerDeviceExist bool
		err := ns.WithNetNSPath(mdp.NetNsPath, func(_ ns.NetNS) error {
			var err error
			doesLowerDeviceExist, err = util.LinkExists(mdp.LowerDevice)
			return err
		})
		if err != nil {
			glog.Warningf("Error while checking on lower device %s: %v", mdp.LowerDevice, err)
			return
		}

		if didLowerDeviceExist != doesLowerDeviceExist {
			emitResponse(doesLowerDeviceExist)
			didLowerDeviceExist = doesLowerDeviceExist
		}
	}

	// Listen for events of lower device interface. On any, check if lower
	// device exists. If it does, offer up to capacity macvtap devices. Do
	// not offer any otherwise.
	util.OnLinkEvent(
		mdp.LowerDevice,
		mdp.NetNsPath,
		onLowerDeviceEvent,
		mdp.stopWatcher,
		func(err error) {
			glog.Error(err)
		})

	return nil
}

func (mdp *macvtapDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	var response pluginapi.AllocateResponse

	for _, req := range r.ContainerRequests {
		var devices []*pluginapi.DeviceSpec
		for _, name := range req.DevicesIDs {
			dev := new(pluginapi.DeviceSpec)

			// There is a possibility the interface already exists from a
			// previous allocation. In a typical scenario, macvtap interfaces
			// would be deleted by the CNI when healthy pod sandbox is
			// terminated. But on occasions, sandbox allocations may fail and
			// the interface is left lingering. The device plugin framework has
			// no de-allocate flow to clean up. So we attempt to delete a
			// possibly existing existing interface before creating it to reset
			// its state.
			var index int
			err := ns.WithNetNSPath(mdp.NetNsPath, func(_ ns.NetNS) error {
				var err error
				index, err = util.RecreateMacvtap(name, mdp.LowerDevice, mdp.Mode)
				return err
			})
			if err != nil {
				return nil, err
			}

			devPath := fmt.Sprint(tapPath, index)
			dev.HostPath = devPath
			dev.ContainerPath = devPath
			dev.Permissions = "rw"
			devices = append(devices, dev)
		}

		response.ContainerResponses = append(response.ContainerResponses, &pluginapi.ContainerAllocateResponse{
			Devices: devices,
		})
	}

	return &response, nil
}

func (mdp *macvtapDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return nil, nil
}

func (mdp *macvtapDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return nil, nil
}

func (mdp *macvtapDevicePlugin) GetPreferredAllocation(context.Context, *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return nil, nil
}

func (mdp *macvtapDevicePlugin) Stop() error {
	close(mdp.stopWatcher)
	return nil
}
