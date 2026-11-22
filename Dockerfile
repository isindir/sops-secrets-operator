############################################################
# https://wiki.ubuntu.com/Releases
# https://gallery.ecr.aws/ubuntu/ubuntu
#   crane ls public.ecr.aws/ubuntu/ubuntu
# https://gallery.ecr.aws/lts/ubuntu
#   crane ls public.ecr.aws/lts/ubuntu
# UPDATE_HERE
FROM public.ecr.aws/ubuntu/ubuntu:26.04 AS mise-builder

# UPDATE_HERE
# https://github.com/jdx/mise/releases
ARG MISE_VERSION=v2026.11.12

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# hadolint ignore=DL3008
RUN apt-get -y update \
  && apt-get -y install --no-install-recommends git curl ca-certificates \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://mise.run | MISE_VERSION=${MISE_VERSION} MISE_INSTALL_PATH=/usr/local/bin/mise sh

# Install go pinned in mise.toml, other tools from mise.toml are not needed to build the binary
WORKDIR /workspace
COPY mise.toml mise.lock ./
RUN mise trust mise.toml && mise install --locked go
ENV PATH="/root/.local/share/mise/shims:$PATH"

# Compile source code
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

############################################################
# UPDATE_HERE
FROM public.ecr.aws/ubuntu/ubuntu:26.04

# Install build tools
RUN apt-get -y update \
  && apt-get -y upgrade \
  && apt-get -y install --no-install-recommends gnupg2 ca-certificates \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/local/bin
COPY --from=mise-builder /workspace/manager .

RUN useradd --create-home --user-group nonroot
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/manager"]
