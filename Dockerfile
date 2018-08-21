# Reproducible builder image
FROM golang:1.10.0 as builder
WORKDIR /go/src/github.com/enxebre/cluster-api-provider-libvirt
# This expects that the context passed to the docker build command is
# the cluster-api-provider-libvirt directory.
COPY . .
RUN apt-get update
RUN apt-get -y install libvirt-dev
RUN CGO_ENABLED=1 go build -v -o ./libvirt-actuator ./

# Final container
FROM debian:stretch-slim
RUN apt-get update
RUN apt-get -y install libvirt-dev
COPY --from=builder /go/src/github.com/enxebre/cluster-api-provider-libvirt/libvirt-actuator .
