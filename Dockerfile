# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine3.23 AS build
RUN apk add --no-cache build-base
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /out/searchproxy .

FROM alpine:3.23
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 65532 nonroot \
    && adduser  -S -u 65532 -G nonroot nonroot
COPY --from=build /out/searchproxy /usr/local/bin/searchproxy
USER nonroot
ENV GODEBUG=netdns=cgo
ENTRYPOINT ["/usr/local/bin/searchproxy"]
CMD ["-config", "/etc/searchproxy/config.yaml"]
