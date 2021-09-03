# syntax=docker/dockerfile:1.3
FROM golang:1 AS gobuild
WORKDIR /work
COPY . ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go build -o /gunk -ldflags "-w -s" -v -mod=readonly .

FROM node:lts AS uibuild
WORKDIR /work
COPY ui/package.json ui/yarn.lock ./
RUN --mount=type=cache,target=/root/.yarn yarn install
COPY ui/ ./
RUN --mount=type=cache,target=/root/.yarn yarn build

FROM debian:testing-slim
RUN rm -f /etc/apt/apt.conf.d/docker-clean; echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    apt update && apt install -y ca-certificates ffmpeg
COPY --from=uibuild /work/dist /usr/share/gunk/ui
COPY --from=gobuild /gunk /usr/bin/gunk
CMD ["/usr/bin/gunk"]
