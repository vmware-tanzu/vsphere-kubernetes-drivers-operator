# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY ./ ./

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

LABEL name="vsphere-kubernetes-drivers-operators"
LABEL maintainer="vdo@vmware.com"
LABEL vendor="VMware"
LABEL version="0.1.0"
LABEL release="1"
LABEL summary="Kubernetes Operator to manage vSphere Kubernetes drivers."
LABEL description="vSphere Kubernetes Drivers Operator manages lifecycle of vSphere CSI/CPI drivers on Kubernetes."

WORKDIR /

# Copy the licenses
COPY LICENSE /licenses/

COPY --from=builder /workspace/manager .
ENTRYPOINT ["/manager"]
