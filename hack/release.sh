#!/bin/bash -e
git tag $TAG
git push https://github.com/kubevirt/macvtap-cni $TAG

$GITHUB_RELEASE release -u kubevirt -r macvtap-cni \
    --tag $TAG \
	--name $TAG \
    --description "$(cat $DESCRIPTION)"

for resource in "$@" ;do
    $GITHUB_RELEASE upload -u kubevirt -r macvtap-cni \
        --name $(basename $resource) \
	    --tag $TAG \
		--file $resource
done
