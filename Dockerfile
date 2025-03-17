# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1 AS gobuild
ARG TARGETOS
ARG TARGETARCH
WORKDIR /work
COPY . ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go test -v ./... && \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -o /gunk -v -mod=readonly .

FROM --platform=$BUILDPLATFORM node AS uibuild
RUN rm -f /etc/apt/apt.conf.d/docker-clean; echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN --mount=type=cache,id=node-apt-cache,target=/var/cache/apt \
    --mount=type=cache,id=node-apt-lib,target=/var/lib/apt \
    apt update && apt install -y brotli
WORKDIR /work
COPY ui/ ./
RUN --mount=type=cache,target=/root/.yarn \
    --mount=type=cache,target=/root/.cache \
    corepack yarn install --immutable && \
    corepack yarn build && find dist -ls
RUN brotli -9kv dist/*.html dist/assets/*.css dist/assets/*.js
RUN gzip -9kv dist/*.html dist/assets/*.css dist/assets/*.js

FROM debian:trixie-slim
RUN rm -f /etc/apt/apt.conf.d/docker-clean; echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    apt update && apt install -y ca-certificates
COPY --link --from=uibuild /work/dist /usr/share/gunk/ui
COPY --link --from=gobuild /gunk /usr/bin/gunk
CMD ["/usr/bin/gunk"]
