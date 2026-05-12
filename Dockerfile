ARG BASE_IMAGE=docker.io/library/ubuntu:25.10@sha256:5922638447b1e3ba114332c896a2c7288c876bb94adec923d70d58a17d2fec5e

####################################################################################################
# Athena Base - minimal runtime image
####################################################################################################
FROM $BASE_IMAGE AS athena-base

LABEL org.opencontainers.image.source="https://github.com/useryege/athena"

USER root

ENV ATHENA_USER_ID=999 \
    DEBIAN_FRONTEND=noninteractive

RUN groupadd -g $ATHENA_USER_ID athena && \
    useradd -r -u $ATHENA_USER_ID -g athena athena && \
    mkdir -p /home/athena && \
    chown athena:0 /home/athena && \
    chmod g=u /home/athena && \
    apt-get update && \
    apt-get install --no-install-recommends -y \
    ca-certificates \
    tini \
    tzdata && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /app/config
RUN mkdir -p tls

ENV USER=athena

USER $ATHENA_USER_ID
WORKDIR /home/athena

####################################################################################################
# Athena UI stage
####################################################################################################
FROM --platform=$BUILDPLATFORM docker.io/library/node:23.0.0@sha256:9d09fa506f5b8465c5221cbd6f980e29ae0ce9a3119e2b9bc0842e6a3f37bb59 AS athena-ui

WORKDIR /src
COPY ["ui/package.json", "ui/yarn.lock", "./"]

RUN yarn install --network-timeout 200000 && \
    yarn cache clean

COPY ["ui/", "."]

ARG ATHENA_VERSION=latest
ENV ATHENA_VERSION=$ATHENA_VERSION
ARG TARGETARCH
RUN HOST_ARCH=$TARGETARCH NODE_ENV='production' NODE_ONLINE_ENV='online' NODE_OPTIONS=--max_old_space_size=8192 yarn build

####################################################################################################
# Athena Build stage which performs the actual build of Athena binaries
####################################################################################################
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.25.5@sha256:31c1e53dfc1cc2d269deec9c83f58729fa3c53dc9a576f6426109d1e319e9e9a AS athena-build

WORKDIR /go/src/github.com/useryege/athena

COPY go.* ./
RUN go mod download

# Perform the build
COPY . .
COPY --from=athena-ui /src/dist/app /go/src/github.com/useryege/athena/ui/dist/app
ARG TARGETOS \
    TARGETARCH
# These build args are optional; if not specified the defaults will be taken from the Makefile
ARG GIT_TAG \
    BUILD_DATE \
    GIT_TREE_STATE \
    GIT_COMMIT
RUN GIT_COMMIT=$GIT_COMMIT \
    GIT_TREE_STATE=$GIT_TREE_STATE \
    GIT_TAG=$GIT_TAG \
    BUILD_DATE=$BUILD_DATE \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    make athena-all

####################################################################################################
# Final image
####################################################################################################
FROM athena-base
ENTRYPOINT ["/usr/bin/tini", "--"]
COPY --from=athena-build /go/src/github.com/useryege/athena/dist/athena* /usr/local/bin/

USER root
RUN ln -s /usr/local/bin/athena /usr/local/bin/athena-server
USER $ATHENA_USER_ID
