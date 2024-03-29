#!/usr/bin/env bash

set -ex

CNI_MOUNT_PATH=${CNI_MOUNT_PATH}                               # the default is stored in Makefile
NAMESPACE=${NAMESPACE}                                         # the default is stored in Makefile
IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY}                         # the default is stored in Makefile
DEVICE_PLUGIN_CONFIG_MAP_NAME=${DEVICE_PLUGIN_CONFIG_MAP_NAME} # the default is stored in Makefile

# compose the full img name - defaults in Makefile
MACVTAP_IMG=${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}

DESTINATION=${DESTINATION:-manifests}

for template in templates/*.in; do
    name=$(basename ${template%.in})
    sed \
        -e "s#'{{#{{#g" \
        -e "s#}}'#}}#g" \
        -e "s#{{ .MacvtapImage }}#${MACVTAP_IMG}#g" \
        -e "s#{{ .CniMountPath }}#${CNI_MOUNT_PATH}#g" \
        -e "s#{{ .Namespace }}#${NAMESPACE}#g" \
        -e "s#{{ .ImagePullPolicy }}#${IMAGE_PULL_POLICY}#g" \
        -e "s#{{ .DevicePluginConfigName }}#${DEVICE_PLUGIN_CONFIG_MAP_NAME}#g" \
        ${template} > ${DESTINATION}/${name}
done
