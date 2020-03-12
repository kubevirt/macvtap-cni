package deviceplugin

import (
	"encoding/json"
	"os"

	"github.com/golang/glog"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
)

const (
	resourceNamespace         = "macvtap.network.kubevirt.io"
	ConfigEnvironmentVariable = "DP_MACVTAP_CONF"
)

type macvtapConfig struct {
	Name     string `json:"name"`
	Master   string `json:"master"`
	Mode     string `json:"mode"`
	Capacity int    `json:"capacity"`
}

type macvtapLister struct {
}

func NewMacvtapLister() *macvtapLister {
	return &macvtapLister{}
}

func (ml macvtapLister) GetResourceNamespace() string {
	return resourceNamespace
}

func readConfig() (map[string]macvtapConfig, error) {
	var config []macvtapConfig
	configMap := make(map[string]macvtapConfig)

	configEnv := os.Getenv(ConfigEnvironmentVariable)
	err := json.Unmarshal([]byte(configEnv), &config)
	if err != nil {
		return configMap, err
	}

	for _, macvtapConfig := range config {
		configMap[macvtapConfig.Name] = macvtapConfig
	}

	return configMap, nil
}

func (ml macvtapLister) Discover(pluginListCh chan dpm.PluginNameList) {
	var plugins = make(dpm.PluginNameList, 0)

	config, err := readConfig()
	if err != nil {
		glog.Errorf("Error reading config: %v", err)
		return
	}

	glog.V(3).Infof("Read configuration %+v", config)

	for _, macvtapConfig := range config {
		plugins = append(plugins, macvtapConfig.Name)
	}

	pluginListCh <- plugins
}

func (ml macvtapLister) NewPlugin(name string) dpm.PluginInterface {
	config, _ := readConfig()
	glog.V(3).Infof("Creating device plugin with config %+v", config[name])
	return NewMacvtapDevicePlugin(
		config[name].Name,
		config[name].Master,
		config[name].Mode,
		config[name].Capacity,
	)
}
