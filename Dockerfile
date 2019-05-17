FROM golang AS gobuild
WORKDIR /work
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /gunk -ldflags "-w -s" -v .

FROM node AS uibuild
WORKDIR /work
COPY ui/package.json ui/package-lock.json ./
RUN npm install
COPY ui/ ./
RUN npm run build

FROM debian:testing-slim
RUN apt update && apt install -y ca-certificates ffmpeg && rm -rf /var/lib/apt/lists/*
COPY --from=gobuild /gunk /usr/bin/gunk
COPY --from=uibuild /work/dist /usr/share/gunk/ui
CMD ["/usr/bin/gunk"]
