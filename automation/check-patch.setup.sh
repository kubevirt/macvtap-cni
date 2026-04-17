# Prepare environment for end to end testing. This includes temporary Go paths and binaries.
#
# source automation/check-patch.e2e.setup.sh
# cd ${TMP_PROJECT_PATH}

echo 'Setup Go paths'
export GOROOT=/tmp/macvtap-cni/go/root
mkdir -p $GOROOT
export GOPATH=/tmp/macvtap-cni/go/path
mkdir -p $GOPATH
export PATH=${GOPATH}/bin:${GOROOT}/bin:${PATH}
export GOBIN=${GOROOT}/bin/
mkdir -p $GOBIN

GO_VERSION=$(grep "^go " go.mod | awk '{print $2}')
echo "Install Go ${GO_VERSION}"
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -sL https://dl.google.com/go/go${GO_VERSION}.linux-${ARCH}.tar.gz | tar -xz -C /tmp/macvtap-cni/go
export GOROOT=/tmp/macvtap-cni/go/go
export GOBIN=${GOROOT}/bin/
export PATH=${GOBIN}:${GOPATH}/bin:${PATH}

echo 'Install operator repository under the temporary Go path'
TMP_PROJECT_PATH=${GOPATH}/src/github.com/kubevirt/macvtap-cni
rm -rf ${TMP_PROJECT_PATH}
mkdir -p ${TMP_PROJECT_PATH}
cp -rf $(pwd)/. ${TMP_PROJECT_PATH}

echo 'Exporting temporary project path'
export TMP_PROJECT_PATH

echo 'Ensuring the manifests are in sync'
if [[ -n $(git status --porcelain 2>/dev/null) ]]; then
    echo "ERROR: git tree state is not clean!"
    echo "Run `make manifests` and commit those changes"
    git status
    git diff
    exit 1
fi
