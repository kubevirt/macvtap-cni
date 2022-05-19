#!/usr/bin/env bash

set -e

determine_cri_bin() {
    if [ "${KUBEVIRTCI_RUNTIME}" = "podman" ]; then
        echo "podman --remote --url=unix://${XDG_RUNTIME_DIR}/podman/podman.sock"
    elif [ "${KUBEVIRTCI_RUNTIME}" = "docker" ]; then
        echo docker
    else
        if curl --unix-socket "${XDG_RUNTIME_DIR}/podman/podman.sock" http://d/v3.0.0/libpod/info >/dev/null 2>&1; then
            echo "podman --remote --url=unix://${XDG_RUNTIME_DIR}/podman/podman.sock"
        elif docker ps >/dev/null 2>&1; then
            echo docker
        else
            echo ""
        fi
    fi
}

determine_cri_bin
