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
node selector, a white-list mechanism is triggered, where only the intersection
of the host interfaces w/ the nics specified in the nicSelector field would be
advertised, under the name specified in each `logical nic`.

It would consist of 3 stages:
  - tagging the hosts
  - provisioning the macvtap policy
  - provisioning the macvtap NAD

#### Host tagging
Host tagging could be based on:
  - macvtapAware: bool (indicates if it is selectable via node selector)
  - TODO ...

#### Macvtap policy
The macvtap policy would look like:
```yaml
apiVersion: ...
kind: MacvtapNetworkNodePolicy
metadata:
  name: policy-1
  namespace: macvtap-cni
spec:
  nicSelector:
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

That resource will be exposed with the name provided in resourceName.

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
