# Copyright 2019-2020 Hewlett Packard Enterprise Development LP

# Dockerfile for building HMS Redfish Collector.

# v1.1.0 of Confluent Go Kafka requirest at least v1.1.0 of librdkafka.
ARG LIBRDKAFKA_VER_MIN=1.1.0

# Build base just has the packages installed we need.
FROM dtr.dev.cray.com/baseos/golang:1.14-alpine3.12 AS build-base

ARG LIBRDKAFKA_VER_MIN

RUN set -ex \
    && apk update \
    && apk add --no-cache \
        build-base \
        "librdkafka-dev>${LIBRDKAFKA_VER_MIN}" \
        pkgconf

# Base copies in the files we need to test/build.
FROM build-base AS base

# Copy all the necessary files to the image.
COPY cmd        $GOPATH/src/stash.us.cray.com/HMS/hms-hmcollector/cmd
COPY internal   $GOPATH/src/stash.us.cray.com/HMS/hms-hmcollector/internal
COPY vendor     $GOPATH/src/stash.us.cray.com/HMS/hms-hmcollector/vendor

### Build Stage ###
FROM base AS builder

RUN set -ex \
    && go build -v -o /usr/local/bin/hmcollector stash.us.cray.com/HMS/hms-hmcollector/cmd/hmcollector

## Final Stage ###

FROM dtr.dev.cray.com/baseos/alpine:3.12
LABEL maintainer="Cray, Inc."
EXPOSE 80

ARG LIBRDKAFKA_VER_MIN

COPY --from=builder /usr/local/bin/hmcollector /usr/local/bin

RUN set -ex \
    && apk update \
    && apk add --no-cache \
        "librdkafka-dev>${LIBRDKAFKA_VER_MIN}" \
        pkgconf \
        curl

ENV LOG_LEVEL=info
ENV POLLING_ENABLED=false
ENV RF_SUBSCRIBE_ENABLED=false
ENV REST_ENABLED=true
ENV KAFKA_HOST=localhost
ENV KAFKA_PORT=9092
ENV POLLING_INTERVAL=1
ENV HSM_REFRESH_INTERVAL=30
ENV SM_URL=https://cray-smd
ENV REST_URL=localhost
ENV REST_PORT=80

CMD ["sh", "-c", "hmcollector"]