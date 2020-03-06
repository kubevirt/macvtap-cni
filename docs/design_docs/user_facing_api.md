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
  - if configMap doesn't exist
    - read all devices & bonds
    - expose 1 resource per deviceOrBond with hard-coded values for:
      - resource name: it would use the 'device' name
      - mode: <AP: which should be the default mode ? macvtap default is VEPA; I think bridge makes more sense>
      - capacity: hard-coded to <AP: which should be the default capacity?>
