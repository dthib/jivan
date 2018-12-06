FROM golang:1.10-alpine AS builder
RUN apk update && apk upgrade && \
    apk add --no-cache alpine-sdk bash openssh openssh-client zip
ADD . /go/src/github.com/go-spatial/jivan/
WORKDIR /go/src/github.com/go-spatial/jivan/
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-s -w" -a -installsuffix cgo -o /go/bin/jivan

FROM  alpine:latest
COPY go-wfs-config.toml /config/config.toml
COPY data/coordinateRefSystem.gpkg /data/coordinateRefSystem.gpkg
COPY --from=builder /go/bin/jivan /go/bin/jivan
ENTRYPOINT ["/go/bin/jivan", "-c", "/config/config.toml"]

