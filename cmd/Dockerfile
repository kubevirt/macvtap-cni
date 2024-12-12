# Multi-stage dockerfile building a container image with both binaries included

FROM --platform=$BUILDPLATFORM quay.io/projectquay/golang:1.20 AS builder
ENV GOPATH=/go
WORKDIR /go/src/github.com/kubevirt/macvtap-cni
ARG TARGETOS
ARG TARGETARCH
ENV TARGETOS=${TARGETOS:-linux}
ENV TARGETARCH=${TARGETARCH:-amd64}

ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

COPY . .
RUN GOOS=linux CGO_ENABLED=0 go build -o /macvtap-deviceplugin github.com/kubevirt/macvtap-cni/cmd/deviceplugin
RUN GOOS=linux CGO_ENABLED=0 go build -o /macvtap-cni github.com/kubevirt/macvtap-cni/cmd/cni

FROM --platform=linux/${TARGETARCH} registry.access.redhat.com/ubi8/ubi-minimal
COPY --from=builder /macvtap-deviceplugin /macvtap-deviceplugin
COPY --from=builder /macvtap-cni /macvtap-cni
