# Multi-stage dockerfile building a container image with both binaries included

FROM golang:1.16 as builder
ENV GOPATH=/go
WORKDIR /go/src/github.com/kubevirt/macvtap-cni
COPY . .
RUN GOOS=linux CGO_ENABLED=0 go build -o /macvtap-deviceplugin github.com/kubevirt/macvtap-cni/cmd/deviceplugin
RUN GOOS=linux CGO_ENABLED=0 go build -o /macvtap-cni github.com/kubevirt/macvtap-cni/cmd/cni

FROM registry.access.redhat.com/ubi8/ubi-minimal
COPY --from=builder /macvtap-deviceplugin /macvtap-deviceplugin
COPY --from=builder /macvtap-cni /macvtap-cni
