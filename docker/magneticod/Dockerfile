FROM golang:latest as builder

WORKDIR /go/src/github.com/izolight/magnetico

ADD ./ .

RUN go get -d -v -tags fts5 ./cmd/magneticod

RUN CGO_ENABLED=0 GOOS=linux go build -a -o magneticod ./cmd/magneticod


FROM alpine:latest

WORKDIR /root/
COPY --from=builder /go/src/github.com/izolight/magnetico/magneticod .

ARG VERSION
ARG BUILD_DATE
ARG VCS_URL
ARG VCS_REF
ARG NAME

LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.name=$NAME \
      org.label-schema.description="magnetico dht crawler backend" \
      org.label-schema.url="https://github.com/izolight/magnetico" \
      org.label-schema.vcs-url=https://github.com/izolight/$VCS_URL \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.version=$VERSION \
      org.label-schema.docker.schema-version="1.0" \
      org.label-schema.docker.cmd="docker run -d izolight/magneticod"

CMD ["./magneticod"]
