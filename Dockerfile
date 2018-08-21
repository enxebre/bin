# Reproducible builder image
FROM golang:1.10.0 as builder
WORKDIR /go/src/github.com/enxebre/cluster-api-libvirt-actuator
# This expects that the context passed to the docker build command is
# the machine-api-operator directory.
# e.g. docker build -t <tag> -f <this_Dockerfile> <path_to_machine-api-operator>
#COPY . .
RUN apt-get update
RUN apt-get -y install libvirt-dev
#RUN CGO_ENABLED=1 go install -a -ldflags '-extldflags "-static"' main.go
