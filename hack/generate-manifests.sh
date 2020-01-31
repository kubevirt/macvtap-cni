#!/usr/bin/env bash

set -ex

IMAGE_REGISTRY=${IMAGE_REGISTRY} # the default is stored in Makefile
IMAGE_NAME=${IMAGE_NAME} # the default is stored in Makefile
IMAGE_TAG=${IMAGE_TAG} # the default is store in Makefile

DESTINATION=${DESTINATION:-manifests}

for template in templates/*.in; do
    name=$(basename ${template%.in})
    sed \
        -e "s#{{ .ImageRegistry }}#${IMAGE_REGISTRY}#g" \
        -e "s#{{ .ImageName }}#${IMAGE_NAME}#g" \
        -e "s#{{ .ImageTag }}#${IMAGE_TAG}#g" \
        ${template} > ${DESTINATION}/${name}
done
