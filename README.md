# macvtap CNI

This plugin allows users to define Kubernetes networks on top of existing
host interfaces. By using the macvtap plugin, the user is able to directly
connect the pod to a host interface and consume it through a tap device.

The main use cases are virtualization workloads inside the pod driven by
Kubevirt but it can also be used directly with QEMU/libvirt and it might be
suitable combined with other virtualization backends.

macvtap CNI includes a device plugin to properly expose the macvtap interfaces
to the pods. A metaplugin such as [Multus](https://github.com/intel/multus-cni)
gets the name of the interface allocated by the device plugin and is responsible
to invoke the cni plugin with that name as deviceID.

## Deployment

The device plugin is configured through environment variable `DP_MACVTAP_CONF`.
The value is a json array and each element of the array is a separate resource
to be made available:

* `name` (string, required) the name of the resource
* `master` (string, required) the name of the macvtap lower link
* `mode` (string, optional, default=bridge) the macvtap operating mode
* `capacity` (uint, optional, default=100) the capacity of the resource

In the default deployment, this configuration shall be provided through a
config map, for [example](examples/macvtap-deviceplugin-config.yaml):

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: macvtap-deviceplugin-config
data:
  DP_MACVTAP_CONF: |
    [ {
        "name" : "dataplane",
        "master" : "eth0",
        "mode": "bridge",
        "capacity" : 50
    } ]
```

```bash
$ kubectl apply -f https://raw.githubusercontent.com/kubevirt/macvtap-cni/master/examples/macvtap-deviceplugin-config.yaml
configmap "macvtap-deviceplugin-config" created
```

This configuration will result in up to 50 macvtap interfaces being offered for
consumption, using eth0 as the lower device, in bridge mode, and under
resource name `macvtap.network.kubevirt.io/dataplane`.

The macvtap CNI can be deployed using the proposed
[daemon set](manifests/macvtap.yaml):

```
$ kubectl apply -f https://raw.githubusercontent.com/kubevirt/macvtap-cni/master/manifests/macvtap.yaml
daemonset "macvtap-cni" created

$ kubectl get pods
NAME                                 READY     STATUS    RESTARTS   AGE
macvtap-cni-745x4                      1/1    Running           0    5m
```

This will result in the CNI being installed and device plugin running on all
nodes.

There is also a [template](templates/macvtap.yaml.in) available to parameterize
the deployment with different configuration options.

## Usage

macvtap CNI is best used with Multus by defining a NetworkAttachmentDefinition:

```yaml
kind: NetworkAttachmentDefinition
apiVersion: k8s.cni.cncf.io/v1
metadata:
  name: dataplane
  annotations:
    k8s.v1.cni.cncf.io/resourceName: macvtap.network.kubevirt.io/dataplane
spec:
  config: '{
      "cniVersion": "0.3.1",
      "name": "dataplane",
      "type": "macvtap-cni"
      "mtu": 1500
    }'
```

The CNI config json allows the following parameters:
* `name`     (string, required): the name of the network. Optional when used within a
   NetworkAttachmentDefinition, as Multus provides the name in that case.
* `type`     (string, required): "macvtap".
* `mac`      (string, optional): mac address to assign to the macvtap interface.
* `mtu`      (integer, optional): mtu to set in the macvtap interface.
* `deviceID` (string, required): name of an existing macvtap host interface, which
  will be moved to the correct net namespace and configured. Optional when used within a
  NetworkAttachmentDefinition, as Multus provides the deviceID in that case.

A pod can be attached to that network which would result in the pod having the corresponding
macvtap interface:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod
  annotations:
    k8s.v1.cni.cncf.io/networks: dataplane
spec:
  containers:
  - name: busybox
    image: busybox
    command: ["/bin/sleep", "1800"]
    resources:
      limits:
        macvtap.network.kubevirt.io/dataplane: 1 
``` 

A mac can also be assigned to the interface through the network annotation:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-with-mac
  annotations:
    k8s.v1.cni.cncf.io/networks: |
      [
        {
          "name":"dataplane",
          "mac": "02:23:45:67:89:01"
        }
      ]
spec:
  containers:
  - name: busybox
    image: busybox
    command: ["/bin/sleep", "1800"]
    resources:
      limits:
        macvtap.network.kubevirt.io/dataplane: 1 
```

**Note:** The resource limit can be ommited from the pod definition if 
[network-resources-injector](https://github.com/intel/network-resources-injector)
is deployed in the cluster.

The device plugin can potentially be used by itself in case you only need the
tap device in the pod and not the interface:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: macvtap-consumer
spec:
  containers:
  - name: busybox
    image: busybox
    command: ["/bin/sleep", "123"]
    resources:
      limits:
        macvtap.network.kubevirt.io/dataplane: 1 
```
