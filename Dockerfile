FROM golang:1.12.7 as builder
WORKDIR /go/src/github.com/cndoit18/image-syncer
COPY ./ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make

FROM alpine:latest
WORKDIR /bin/
COPY --from=builder /go/src/github.com/cndoit18/image-syncer/image-syncer ./
RUN chmod +x ./image-syncer
RUN apk add -U --no-cache ca-certificates && rm -rf /var/cache/apk/* && mkdir -p /etc/ssl/certs \
  && update-ca-certificates --fresh
ENTRYPOINT ["image-syncer"]
CMD ["--config", "/etc/image-syncer/image-syncer.json"]
