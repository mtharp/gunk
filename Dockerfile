FROM golang AS gobuild
WORKDIR /work
COPY go.mod go.sum ./
COPY _srtp _srtp
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gunk -ldflags "-w -s" -v .

FROM node AS uibuild
WORKDIR /work
COPY ui/package.json ui/package-lock.json ./
RUN npm install
COPY ui/ ./
RUN npm run build

FROM alpine
RUN apk add --no-cache ca-certificates tzdata
COPY --from=gobuild /gunk /usr/bin/gunk
COPY --from=uibuild /work/dist /usr/share/gunk/ui
CMD ["/usr/bin/gunk"]
