# UPDATE_HERE
# Build the manager binary
# https://www.debian.org/releases/
# https://hub.docker.com/_/golang/tags?page=1&name=bullseye
FROM golang:1.21.3-bullseye as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Build (GOARCH=amd64)
RUN CGO_ENABLED=0 GO111MODULE=on go build -a -o manager cmd/main.go

# https://wiki.ubuntu.com/Releases
# https://hub.docker.com/_/ubuntu/tags?page=1&name=lunar
FROM ubuntu:lunar-20231004

RUN apt-get -y update \
      && apt-get -y upgrade \
      && apt-get -y install --no-install-recommends gnupg2 ca-certificates \
      && apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/local/bin
COPY --from=builder /workspace/manager .

RUN useradd --create-home --user-group nonroot
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/manager"]
