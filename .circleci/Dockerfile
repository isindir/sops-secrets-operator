FROM golang:1.13.4-alpine3.10

ENV GO111MODULE on
ENV GO15VENDOREXPERIMENT 1

RUN apk add --no-cache \
        bash \
        ca-certificates \
        curl \
        docker \
        git \
        gnupg \
        jq \
        make

ARG HELM_VERSION=v2.16.1
ARG CHART_RELEASER_VERSION=0.1.4
ARG KUBEVAL_VERSION=0.12.0
ARG OPERATOR_SDK_VERSION=0.13.0

RUN wget -q https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz -O - | tar -xzO linux-amd64/helm > /usr/local/bin/helm \
    && chmod +x /usr/local/bin/helm

RUN wget -q https://github.com/instrumenta/kubeval/releases/download/${KUBEVAL_VERSION}/kubeval-linux-amd64.tar.gz -O - \
    | tar -xzO kubeval > /usr/local/bin/kubeval \
    && chmod +x /usr/local/bin/kubeval

RUN wget -q https://github.com/helm/chart-releaser/releases/download/v${CHART_RELEASER_VERSION}/chart-releaser_${CHART_RELEASER_VERSION}_Linux_x86_64.tar.gz -O  - \
    | tar -xzO chart-releaser > /usr/local/bin/chart-releaser \
    && chmod +x /usr/local/bin/chart-releaser

RUN wget -q https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk-v${OPERATOR_SDK_VERSION}-x86_64-linux-gnu -O /usr/local/bin/operator-sdk \
    && chmod +x /usr/local/bin/operator-sdk

RUN go get -u golang.org/x/lint/golint

RUN mkdir -p /go/src/github.com/isindir
WORKDIR /go/src/github.com/isindir

CMD [ "bash" ]
