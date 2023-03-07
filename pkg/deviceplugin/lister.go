package deviceplugin

import (
	"encoding/json"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/golang/glog"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"github.com/kubevirt/macvtap-cni/pkg/util"
)

const (
	resourceNamespace         = "macvtap.network.kubevirt.io"
	ConfigEnvironmentVariable = "DP_MACVTAP_CONF"
)

type macvtapConfig struct {
	Name        string `json:"name"`
	LowerDevice string `json:"lowerDevice"`
	Mode        string `json:"mode"`
	Capacity    int    `json:"capacity"`
}

type macvtapLister struct {
	Config map[string]macvtapConfig
	// NetNsPath is the path to the network namespace the lister operates in.
	NetNsPath string
}

func NewMacvtapLister(netNsPath string) *macvtapLister {
	return &macvtapLister{
		NetNsPath: netNsPath,
	}
}

func (ml macvtapLister) GetResourceNamespace() string {
	return resourceNamespace
}

func readConfig() (map[string]macvtapConfig, error) {
	var config []macvtapConfig
	configMap := make(map[string]macvtapConfig)

	configEnv, isEnvVarDefined := os.LookupEnv(ConfigEnvironmentVariable)
	if !isEnvVarDefined {
		return map[string]macvtapConfig{}, nil
	}

	err := json.Unmarshal([]byte(configEnv), &config)
	if err != nil {
		return configMap, err
	}

	for _, macvtapConfig := range config {
		configMap[macvtapConfig.Name] = macvtapConfig
	}

	return configMap, nil
}

func discoverByConfig(pluginListCh chan dpm.PluginNameList) (map[string]macvtapConfig, error) {
	var plugins = make(dpm.PluginNameList, 0)

	config, err := readConfig()
	if err != nil {
		glog.Errorf("Error reading config: %v", err)
		return nil, err
	}

	glog.V(3).Infof("Read configuration %+v", config)

	for _, macvtapConfig := range config {
		plugins = append(plugins, macvtapConfig.Name)
	}

	if len(plugins) > 0 {
		pluginListCh <- plugins
	}
	return config, nil
}

func discoverByLinks(pluginListCh chan dpm.PluginNameList, netNsPath string) error {
	// To know when the manager is stoping, we need to read from pluginListCh.
	// We avoid reading our own updates by using a middle channel.
	// We buffer up to one msg because of the initial call to sendSuitableParents.
	parentListCh := make(chan []string, 1)
	defer close(parentListCh)

	sendSuitableParents := func() error {
		var linkNames []string
		err := ns.WithNetNSPath(netNsPath, func(_ ns.NetNS) error {
			var err error
			linkNames, err = util.FindSuitableMacvtapParents()
			return err
		})

		if err != nil {
			glog.Errorf("Error while finding links: %v", err)
			return err
		}

		parentListCh <- linkNames
		return nil
	}

	// Do an initial search to catch early permanent runtime problems
	err := sendSuitableParents()
	if err != nil {
		return err
	}

	// Keep updating on changes for suitable parents.
	stop := make(chan struct{})
	defer close(stop)
	go util.OnSuitableMacvtapParentEvent(
		netNsPath,
		// Wrapper to ignore error
		func() {
			sendSuitableParents()
		},
		stop,
		func(err error) {
			glog.Error(err)
		})

	// Keep forwarding updates to the manager until it closes down
	for {
		select {
		case parentNames := <-parentListCh:
			pluginListCh <- parentNames
		case _, open := <-pluginListCh:
			if !open {
				return nil
			}
		}
	}
}

func (ml *macvtapLister) Discover(pluginListCh chan dpm.PluginNameList) {
	config, err := discoverByConfig(pluginListCh)
	if err != nil {
		os.Exit(1)
	}

	// Configuration is static and we don't need to do anything else
	ml.Config = config
	if len(config) > 0 {
		return
	}

	// If there was no configuration, we setup resources based on the existing
	// links of the host.
	err = discoverByLinks(pluginListCh, ml.NetNsPath)
	if err != nil {
		os.Exit(1)
	}
}

func (ml *macvtapLister) NewPlugin(name string) dpm.PluginInterface {
	c, ok := ml.Config[name]
	if !ok {
		c = macvtapConfig{
			Name:        name,
			LowerDevice: name,
			Mode:        DefaultMode,
			Capacity:    DefaultCapacity,
		}
	}

	glog.V(3).Infof("Creating device plugin with config %+v", c)
	return NewMacvtapDevicePlugin(c.Name, c.LowerDevice, c.Mode, c.Capacity, ml.NetNsPath)
}
