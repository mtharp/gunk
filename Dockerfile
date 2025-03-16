# syntax=docker/dockerfile:1.3
FROM --platform=$BUILDPLATFORM golang:1 AS gobuild
ARG TARGETOS
ARG TARGETARCH
WORKDIR /work
COPY . ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go test -v ./... && \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -o /gunk -ldflags "-w -s" -v -mod=readonly .

FROM --platform=$BUILDPLATFORM node AS uibuild
WORKDIR /work
COPY ui/ ./
RUN --mount=type=cache,target=/root/.yarn yarn install
RUN --mount=type=cache,target=/root/.yarn yarn build

FROM debian:trixie-slim
RUN rm -f /etc/apt/apt.conf.d/docker-clean; echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    apt update && apt install -y ca-certificates brotli
COPY --from=uibuild /work/dist /usr/share/gunk/ui
RUN brotli -9kv /usr/share/gunk/ui/*.html /usr/share/gunk/ui/assets/*.css /usr/share/gunk/ui/assets/*.js
RUN gzip -9kv /usr/share/gunk/ui/*.html /usr/share/gunk/ui/assets/*.css /usr/share/gunk/ui/assets/*.js
COPY --from=gobuild /gunk /usr/bin/gunk
CMD ["/usr/bin/gunk"]
