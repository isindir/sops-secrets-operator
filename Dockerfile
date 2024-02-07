############################################################
# https://wiki.ubuntu.com/Releases
# https://hub.docker.com/_/ubuntu/tags?page=1&name=noble
# UPDATE_HERE
FROM ubuntu:noble-20240114 as asdf-builder

# UPDATE_HERE
ARG ASDF_VERSION=v0.14.0

# Install build tools
RUN apt-get -y update \
      && apt-get -y install build-essential \
      && apt-get -y install autoconf automake gdb git libffi-dev zlib1g-dev libssl-dev curl \
      && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install asdf
WORKDIR /usr/local
RUN git config --global user.email "you@example.com" \
  && git config --global user.name "Your Name" \
  && git config --global init.defaultBranch ${ASDF_VERSION} \
  && git config --global pull.rebase true \
  && git init \
  && git add . \
  && git commit -m'Initial commit' && git remote add origin https://github.com/asdf-vm/asdf.git \
  && git pull origin ${ASDF_VERSION} --allow-unrelated-histories \
  && rm -fr .git ~/.gitconfig

# Install project build tools and linters using asdf
WORKDIR /root
COPY .tool-versions .

RUN awk '$0 !~ /^#/ {print $1}' ~/.tool-versions|xargs -i asdf plugin add  {} \
      && asdf install && asdf reshim
ENV PATH "/root/.asdf/shims:/root/.asdf/bin:$PATH"

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
FROM ubuntu:noble-20240114

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
