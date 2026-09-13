ARG BASE_IMAGE=docker.io/library/ubuntu:25.10@sha256:5922638447b1e3ba114332c896a2c7288c876bb94adec923d70d58a17d2fec5e

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27.1@sha256:f44f6e88636cfb311f9ebace870ded69d943f227bb3cb27d32ffd84ea18c43ea AS build

WORKDIR /go/src/github.com/useryege/athena

COPY go.* ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG GIT_TAG
ARG BUILD_DATE
ARG GIT_TREE_STATE
ARG GIT_COMMIT
RUN GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    STATIC_BUILD=true \
    make \
    GIT_COMMIT=$GIT_COMMIT \
    GIT_TREE_STATE=$GIT_TREE_STATE \
    GIT_TAG=$GIT_TAG \
    BUILD_DATE=$BUILD_DATE \
    athena-bsc-swap-indexer

FROM $BASE_IMAGE

LABEL org.opencontainers.image.source="https://github.com/useryege/athena"

ENV ATHENA_USER_ID=999 \
    DEBIAN_FRONTEND=noninteractive \
    USER=athena

RUN groupadd -g $ATHENA_USER_ID athena && \
    useradd -r -u $ATHENA_USER_ID -g athena athena && \
    mkdir -p /home/athena && \
    chown athena:athena /home/athena && \
    apt-get update && \
    apt-get install --no-install-recommends -y ca-certificates curl tini tzdata && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY --from=build /go/src/github.com/useryege/athena/dist/athena-bsc-swap-indexer /usr/local/bin/athena-bsc-swap-indexer

USER $ATHENA_USER_ID
WORKDIR /home/athena

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/athena-bsc-swap-indexer"]
