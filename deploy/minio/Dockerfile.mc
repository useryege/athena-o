ARG GO_IMAGE=docker.io/library/golang:1.25.5@sha256:31c1e53dfc1cc2d269deec9c83f58729fa3c53dc9a576f6426109d1e319e9e9a
ARG RUNTIME_IMAGE=docker.io/library/ubuntu:25.10@sha256:5922638447b1e3ba114332c896a2c7288c876bb94adec923d70d58a17d2fec5e

FROM ${GO_IMAGE} AS build

ARG MC_COMMIT=7394ce0dd2a80935aded936b09fa12cbb3cb8096

RUN apt-get update && \
    apt-get install --no-install-recommends -y ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /src
RUN git init && \
    git remote add origin https://github.com/minio/mc.git && \
    git fetch --depth 1 origin "${MC_COMMIT}" && \
    git checkout --detach FETCH_HEAD && \
    test "$(git rev-parse HEAD)" = "${MC_COMMIT}"

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "$(MC_RELEASE=RELEASE go run buildscripts/gen-ldflags.go)" \
    -o /out/mc .

FROM ${RUNTIME_IMAGE}

ARG MC_COMMIT=7394ce0dd2a80935aded936b09fa12cbb3cb8096

LABEL org.opencontainers.image.source="https://github.com/minio/mc" \
      org.opencontainers.image.revision="${MC_COMMIT}"

RUN apt-get update && \
    apt-get install --no-install-recommends -y ca-certificates tini && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /out/mc /usr/local/bin/mc
COPY --from=build /src/LICENSE /licenses/minio-mc/LICENSE
COPY --from=build /src/CREDITS /licenses/minio-mc/CREDITS
COPY --chmod=0555 init-avatar-bucket.sh /usr/local/bin/init-avatar-bucket

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/init-avatar-bucket"]
