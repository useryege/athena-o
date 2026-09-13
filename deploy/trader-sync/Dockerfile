ARG BASE_IMAGE=docker.io/library/ubuntu:25.10@sha256:5922638447b1e3ba114332c896a2c7288c876bb94adec923d70d58a17d2fec5e
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27.1@sha256:f44f6e88636cfb311f9ebace870ded69d943f227bb3cb27d32ffd84ea18c43ea AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH
ARG VERSION=dev GIT_COMMIT=unknown GIT_TREE_STATE=dirty GIT_TAG= BUILD_DATE=unknown
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -buildvcs=false -ldflags "-s -w -X github.com/useryege/athena/common.version=$VERSION -X github.com/useryege/athena/common.gitCommit=$GIT_COMMIT -X github.com/useryege/athena/common.gitTreeState=$GIT_TREE_STATE -X github.com/useryege/athena/common.gitTag=$GIT_TAG -X github.com/useryege/athena/common.buildDate=$BUILD_DATE" -o /out/athena-trader-sync ./cmd/athena-trader-sync
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -buildvcs=false -ldflags "-s -w -X github.com/useryege/athena/common.version=$VERSION -X github.com/useryege/athena/common.gitCommit=$GIT_COMMIT -X github.com/useryege/athena/common.gitTreeState=$GIT_TREE_STATE -X github.com/useryege/athena/common.gitTag=$GIT_TAG -X github.com/useryege/athena/common.buildDate=$BUILD_DATE" -o /out/athena-account-state-migrate ./cmd/athena-account-state-migrate
FROM $BASE_IMAGE
LABEL org.opencontainers.image.source="https://github.com/useryege/athena"
RUN groupadd -g 999 athena && useradd -r -u 999 -g athena athena && \
    apt-get update && apt-get install --no-install-recommends -y ca-certificates tini tzdata && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
COPY --from=build /out/athena-trader-sync /out/athena-account-state-migrate /usr/local/bin/
USER 999:999
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["athena-trader-sync"]
