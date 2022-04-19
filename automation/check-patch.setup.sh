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

echo 'Install Go 1.16'
export GIMME_GO_VERSION=1.16
GIMME=/tmp/macvtap-cni/go/gimme
mkdir -p $GIMME
curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=${GIMME} bash >> ${GIMME}/gimme.sh
source ${GIMME}/gimme.sh

echo 'Install operator repository under the temporary Go path'
TMP_PROJECT_PATH=${GOPATH}/src/github.com/kubevirt/macvtap-cni
rm -rf ${TMP_PROJECT_PATH}
mkdir -p ${TMP_PROJECT_PATH}
cp -rf $(pwd)/. ${TMP_PROJECT_PATH}

echo 'Exporting temporary project path'
export TMP_PROJECT_PATH
