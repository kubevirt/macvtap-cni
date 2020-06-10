# Proposal for user facing API

## Objectives
- deploy the plugins without day 1 configuration
- dynamic day 2 host resource exposure
- expose lower device as a logical resource
- be able to use node selectors / node affinity / NIC selectors /
resource name

## Requirements
- keep the DP as independent from multus and CNI as possible
- keep the explicit configuration (configMap)
- have an implicit configuration (would be overriden by the explicit)

## Alternatives

### Expose all host interfaces
This approach would be valid for MVP.

It would:
  - read data from configMap
  - if configMap provides empty configuration
    - read all devices & bonds
    - expose 1 resource per deviceOrBond with hard-coded values for:
      - resource name: it would use the 'device' name
      - mode: bridge
      - capacity: 100

This approach would fulfil 3/5 objectives, missing:
- expose lower device as a logical resource
- be able to use node selectors / node affinity / NIC selectors /
resource name

This approach is implemented in
[PR #10](https://github.com/kubevirt/macvtap-cni/pull/10), and is considered
good enough for the MVP.

### Host device exposure using selectors
This proposal complements the proposal where
[all interfaces are exposed](#expose-all-host-interfaces). It would follow the
same behavior of
[k8s network policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/#isolated-and-non-isolated-pods);
by default all host's interfaces are exposed, but once an host is selected via
node selector, a white-list mechanism is triggered.

It would consist of 3 stages:
  - tagging the hosts
  - provisioning the macvtap policy
  - provisioning the macvtap NAD

#### Host tagging
Host tagging could be based on:
  - macvtapAware: bool (indicates if it is selectable via node selector)
  - !node-role.kubernetes.io/master
  - tag the nodes with the interface names you want it to expose (???)
    - e.g. parent_iface.macvtap.network.kubevirt.io/eth0

#### Macvtap policy
The macvtap policy would look like this, should we choose a node-centric
approach:
```yaml
apiVersion: ...
kind: MacvtapNetworkNodePolicy
metadata:
  name: policy-1
  namespace: macvtap-cni
spec:
  parentSelector:
    - parentIface: bond1
      mode: private
      quantity: 25
      resourceName: bondedIface
    - parentIface: eth1
      mode: vepa
      resourceName: quickDataplane
    - parentIface: eth2
      resourceName: dataplane
  nodeSelector:
    matchLabels:
      macvtapAware: "true"
```
For each node matching the `nodeSelector` knob, it would expose all interfaces
on that host that are featured in the `parentSelector` list. Each resource would
be advertised under the name specified in the `resourceName` attribute.

Let's see an example, to better follow the proposal; there are 3 hosts,
h1, h2, h3. The interfaces on each of them can be seen on the following yaml:

```yaml
h1:
  - bond1
  - eth0
  - eth1
h2:
  - bond1
  - ens3
  - ens4
  - ens5
  - ens6
h3:
  - eth0
  - eth1
```
Considering that nodes h1 and h2 **are tagged** with macvtapAware, and the
cluster has been provisioned with the following policy:

```yaml
apiVersion: ...
kind: MacvtapNetworkNodePolicy
metadata:
  name: policy-1
  namespace: macvtap-cni
spec:
  parentSelector:
    - parentIface: bond1
      mode: private
      quantity: 25
      resourceName: bondedIface
    - parentIface: eth1
      mode: vepa
      resourceName: quickDataplane
    - parentIface: eth2
      resourceName: dataplane
  nodeSelector:
    matchLabels:
      macvtapAware: true
```

It would expose, in each of the nodes:
```yaml
h1 (node has 2 interfaces from the policy iface list):
  - bond1; exposed as `macvtap.network.kubevirt.io/bondedIface`
  - eth1;  exposed as `macvtap.network.kubevirt.io/quickDataplane`

h2 (node has 1 interface from the policy iface list):
  - bond1; exposed as `macvtap.network.kubevirt.io/bondedIface`

h3 (not matched, exposing everything):
  - eth0;  exposed as `macvtap.network.kubevirt.io/eth0`  # iface name exposed (default)
  - eth1;  exposed as `macvtap.network.kubevirt.io/eth1`  # iface name exposed (default)
```

An alternative model, more resource-centric, where a 1 to 1 resource / policy
association can also be achieved by shaping the policy like:
```yaml
apiVersion: ...
kind: MacvtapNetworkNodePolicy
metadata:
  name: policy-1
  namespace: macvtap-cni
spec:
  parentSelector:
    - parentIface: bond1
      mode: private
      quantity: 25
    - parentIface: eth1
      mode: vepa
    - parentIface: eth2
  nodeSelector:
    matchLabels:
      macvtapAware: true
  resourceName: dataplane
```

It would expose a single resource - `resourceName` - and on each of the nodes,
which are selected via the `nodeSelector` attribute, it would expose the first
interface matched on the `parentSelector` list.

Taking into account the previous example, it would expose the following
interfaces in each of the nodes:

```yaml
h1 (exposing first match from the iface list):
  - bond1; exposed as `macvtap.network.kubevirt.io/bondedIface`

h2 (exposing first match from the iface list):
  - bond1; exposed as `macvtap.network.kubevirt.io/bondedIface`

h3 (not matched, exposing everything):
  - eth0;  exposed as `macvtap.network.kubevirt.io/eth0`  # iface name exposed (default)
  - eth1;  exposed as `macvtap.network.kubevirt.io/eth1`  # iface name exposed (default)
```

#### Macvtap NetworkAttachmentDefinition
The macvtap NetworkAttachmentDefinition would look like:
```yaml
  apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: macvtap0
  annotations:
    k8s.v1.cni.cncf.io/resourceName: macvtap.network.kubevirt.io/bondedIface
spec:
  config: '{
      "cniVersion": "0.3.1",
      "type": "macvtap",
      "mtu": 9000
    }'
```

This approach would fulfil **all** objectives.
The config map that holds the configuration would not be needed.
