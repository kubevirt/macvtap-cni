#!/usr/bin/env bash

if [ -z "$PLATFORMS" ] || [ -z "$MACVTAP_IMAGE_TAGGED" ]; then
    echo "Error: PLATFORMS, and MACVTAP_IMAGE_TAGGED must be set."
    exit 1
fi

IFS=',' read -r -a PLATFORM_LIST <<< "$PLATFORMS"

podman manifest rm "${MACVTAP_IMAGE_TAGGED}" 2>/dev/null || true
podman manifest rm "${MARKER_IMAGE_GIT_TAGGED}" 2>/dev/null || true
podman rmi "${MACVTAP_IMAGE_TAGGED}" 2>/dev/null || true
podman rmi "${MARKER_IMAGE_GIT_TAGGED}" 2>/dev/null || true
podman rmi $(podman images --filter "dangling=true" -q) 2>/dev/null || true

podman manifest create "${MACVTAP_IMAGE_TAGGED}"

for platform in "${PLATFORM_LIST[@]}"; do
    podman build \
        --platform "$platform" \
        --manifest "${MACVTAP_IMAGE_TAGGED}" \
        -f cmd/Dockerfile .
done
