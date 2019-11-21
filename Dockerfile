FROM golang:1.12.7 as builder
COPY ./ /go/src/github.com/AliyunContainerService/image-syncer
WORKDIR /go/src/github.com/AliyunContainerService/image-syncer
RUN CGO_ENABLED=0 GOOS=linux make

FROM alpine:latest
COPY --from=builder /go/src/github.com/AliyunContainerService/image-syncer/image-syncer /bin/
RUN chmod +x /bin/image-syncer
CMD ["image-syncer", "--config", "/etc/image-syncer/image-syncer.json"]
