FROM golang:1.13rc1 AS gobuild
WORKDIR /work
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /gunk -ldflags "-w -s" -v -mod=readonly .

FROM node AS uibuild
WORKDIR /work
COPY ui/package.json ui/yarn.lock ./
RUN yarn install
COPY ui/ ./
RUN yarn build

FROM debian:testing-slim
RUN apt update && apt install -y ca-certificates ffmpeg && rm -rf /var/lib/apt/lists/*
COPY --from=gobuild /gunk /usr/bin/gunk
COPY --from=uibuild /work/dist /usr/share/gunk/ui
CMD ["/usr/bin/gunk"]
