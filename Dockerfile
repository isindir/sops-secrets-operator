############################################################
# https://wiki.ubuntu.com/Releases
# https://gallery.ecr.aws/ubuntu/ubuntu
#   crane ls public.ecr.aws/ubuntu/ubuntu
# https://gallery.ecr.aws/lts/ubuntu
#   crane ls public.ecr.aws/lts/ubuntu
# UPDATE_HERE
FROM public.ecr.aws/ubuntu/ubuntu:26.04 AS install-asdf

# UPDATE_HERE
# https://github.com/asdf-vm/asdf/releases
ARG ASDF_VERSION=v0.18.0

# hadolint ignore=DL3008
RUN apt-get -y update \
  && apt-get -y install --no-install-recommends git bash golang ca-certificates \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN go install github.com/asdf-vm/asdf/cmd/asdf@${ASDF_VERSION}

############################################################
# UPDATE_HERE
FROM public.ecr.aws/ubuntu/ubuntu:26.04 AS asdf-builder

COPY --from=install-asdf /root/go/bin/asdf /usr/local/bin/

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install build tools
RUN apt-get -y update \
  && apt-get -y install --no-install-recommends build-essential \
  && apt-get -y install --no-install-recommends autoconf automake gdb git libffi-dev zlib1g-dev libssl-dev curl wget ca-certificates \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install project build tools and linters using asdf
WORKDIR /root
COPY .tool-versions .

RUN awk '$0 !~ /^#/ {print $1}' .tool-versions|xargs -I{} asdf plugin add  {} \
  && asdf install && asdf reshim
ENV PATH="/root/.asdf/shims:/root/.asdf/bin:$PATH"

# Compile source code
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

############################################################
# UPDATE_HERE
FROM public.ecr.aws/ubuntu/ubuntu:26.04

# Install build tools
RUN apt-get -y update \
  && apt-get -y upgrade \
  && apt-get -y install --no-install-recommends gnupg2 ca-certificates \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/local/bin
COPY --from=asdf-builder /workspace/manager .

RUN useradd --create-home --user-group nonroot
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/manager"]
