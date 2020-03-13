package deviceplugin

import (
	"fmt"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/kubevirt/macvtap-cni/pkg/util"
)

const (
	tapPath = "/dev/tap"
	// Interfaces will be named as <Name><suffix>[0-<Capacity>]
	suffix          = "Mvp"
	defaultCapacity = 100
)

type MacvtapDevicePlugin struct {
	Name        string
	Master      string
	Mode        string
	Capacity    int
	stopWatcher chan struct{}
}

func NewMacvtapDevicePlugin(name string, master string, mode string, capacity int) *MacvtapDevicePlugin {
	return &MacvtapDevicePlugin{
		Name:        name,
		Master:      master,
		Mode:        mode,
		Capacity:    capacity,
		stopWatcher: make(chan struct{}),
	}
}

func (mdp *MacvtapDevicePlugin) generateMacvtapDevices() []*pluginapi.Device {
	var macvtapDevs []*pluginapi.Device

	var capacity = mdp.Capacity
	if capacity <= 0 {
		capacity = defaultCapacity
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

func (mdp *MacvtapDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	// Initialize two arrays, one for devices offered when master exists,
	// and no devices if master does not exist.
	allocatableDevs := mdp.generateMacvtapDevices()
	emptyDevs := make([]*pluginapi.Device, 0)

	emitResponse := func(masterExists bool) {
		if masterExists {
			glog.V(3).Info("Master exists, sending ListAndWatch response with available devices")
			s.Send(&pluginapi.ListAndWatchResponse{Devices: allocatableDevs})
		} else {
			glog.V(3).Info("Master does not exist, sending ListAndWatch response with no devices")
			s.Send(&pluginapi.ListAndWatchResponse{Devices: emptyDevs})
		}
	}

	didMasterExist := false
	onMasterEvent := func() {
		doesMasterExist, err := util.LinkExists(mdp.Master)
		if err != nil {
			glog.Warningf("Error while checking on master %s: %v", mdp.Master, err)
			return
		}
		if didMasterExist != doesMasterExist {
			emitResponse(doesMasterExist)
			didMasterExist = doesMasterExist
		}
	}

	// Listen for events of master interface. On any, check if master a
	// interface exists. If it does, offer up to capacity macvtap devices. Do
	// not offer any otherwise.
	util.OnLinkEvent(
		mdp.Master,
		onMasterEvent,
		mdp.stopWatcher,
		func(err error) {
			glog.Error(err)
		})

	return nil
}

func (mdp *MacvtapDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
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
			index, err := util.RecreateMacvtap(name, mdp.Master, mdp.Mode)
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

func (mdp *MacvtapDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return nil, nil
}

func (mdp *MacvtapDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return nil, nil
}

func (mdp *MacvtapDevicePlugin) Stop() error {
	close(mdp.stopWatcher)
	return nil
}
