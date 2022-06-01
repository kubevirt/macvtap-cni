module github.com/kubevirt/macvtap-cni

go 1.17

require (
	github.com/aktau/github-release v0.8.1
	github.com/containernetworking/cni v0.8.1
	github.com/containernetworking/plugins v0.9.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/kubevirt/device-plugin-manager v0.19.2
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.3
	github.com/pkg/errors v0.9.1
	github.com/vishvananda/netlink v1.1.1-0.20201029203352-d40f9887b852
	github.com/vishvananda/netns v0.0.0-20200728191858-db3c7e526aae
	golang.org/x/net v0.0.0-20211209124913-491a49abca63
	golang.org/x/tools v0.1.6-0.20210820212750-d4cc65f0b2ff
	google.golang.org/grpc v1.40.0
	k8s.io/api v0.23.6
	k8s.io/apimachinery v0.23.6
	k8s.io/client-go v0.23.6
	k8s.io/kubelet v0.23.6
)

require (
	github.com/coreos/go-iptables v0.5.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/github-release/github-release v0.8.1 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/inconshreveable/log15 v0.0.0-20200109203555-b30bc20e4fd1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/rest v0.0.0-20200429221318-0d2892b400f8 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nxadm/tail v1.4.4 // indirect
	github.com/safchain/ethtool v0.0.0-20190326074333-42ed695e3de8 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80 // indirect
	github.com/voxelbrain/goptions v0.0.0-20180630082107-58cddc247ea2 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f // indirect
	golang.org/x/sys v0.0.0-20210831042530-f4d43177bf5e // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210831024726-fe130286e0e2 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65 // indirect
	k8s.io/utils v0.0.0-20211116205334-6203023598ed // indirect
	sigs.k8s.io/json v0.0.0-20211020170558-c049b76a60c6 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

// Pinned to kubernetes-1.21.11
replace (
	golang.org/x/text => golang.org/x/text v0.3.3
	k8s.io/api => k8s.io/api v0.23.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.6
	k8s.io/apiserver => k8s.io/apiserver v0.23.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.6
	k8s.io/client-go => k8s.io/client-go v0.23.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.6
	k8s.io/code-generator => k8s.io/code-generator v0.23.6
	k8s.io/component-base => k8s.io/component-base v0.23.6
	k8s.io/cri-api => k8s.io/cri-api v0.23.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.6
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.6
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.1
	k8s.io/kubelet => k8s.io/kubelet v0.23.6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.6
	k8s.io/metrics => k8s.io/metrics v0.23.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.1
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
)
