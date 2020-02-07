# Multi-stage dockerfile building a container image with both binaries included

FROM golang:1.13 as builder
ENV GOPATH=/go
WORKDIR /go/src/github.com/kubevirt/macvtap-cni
COPY . .
RUN GOOS=linux CGO_ENABLED=0 go build -o /macvtap github.com/kubevirt/macvtap-cni/cmd/deviceplugin

FROM registry.access.redhat.com/ubi8/ubi-minimal
COPY --from=builder /macvtap /macvtap
ENTRYPOINT [ "./macvtap", "-v", "3", "-logtostderr"]
