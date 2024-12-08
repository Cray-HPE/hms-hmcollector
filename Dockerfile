# MIT License
#
# (C) Copyright [2019-2022,2024] Hewlett Packard Enterprise Development LP
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and associated documentation files (the "Software"),
# to deal in the Software without restriction, including without limitation
# the rights to use, copy, modify, merge, publish, distribute, sublicense,
# and/or sell copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
# THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
# OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.

# Dockerfile for building HMS Redfish Collector.

# v1.1.0 of Confluent Go Kafka requires at least v1.1.0 of librdkafka.
ARG LIBRDKAFKA_VER_MIN=1.1.0

# Build base just has the packages installed we need.
FROM artifactory.algol60.net/docker.io/library/golang:1.23-alpine AS build-base

ARG LIBRDKAFKA_VER_MIN

RUN set -ex \
    && apk -U upgrade \
    && apk add --no-cache \
        build-base \
        "librdkafka-dev>${LIBRDKAFKA_VER_MIN}" \
        pkgconf

# Base copies in the files we need to test/build.
FROM build-base AS base

RUN go env -w GO111MODULE=auto

# Copy all the necessary files to the image.
COPY cmd        $GOPATH/src/github.com/Cray-HPE/hms-hmcollector/cmd
COPY internal   $GOPATH/src/github.com/Cray-HPE/hms-hmcollector/internal
COPY vendor     $GOPATH/src/github.com/Cray-HPE/hms-hmcollector/vendor

### Build Stage ###
FROM base AS builder

RUN set -ex \
    && go build -v -o /usr/local/bin/hmcollector github.com/Cray-HPE/hms-hmcollector/cmd/hmcollector

## Final Stage ###

FROM artifactory.algol60.net/docker.io/alpine:3.15
LABEL maintainer="Hewlett Packard Enterprise"
EXPOSE 80

ARG LIBRDKAFKA_VER_MIN

COPY --from=builder /usr/local/bin/hmcollector /usr/local/bin

RUN set -ex \
    && apk -U upgrade \
    && apk add --no-cache \
        "librdkafka-dev>${LIBRDKAFKA_VER_MIN}" \
        pkgconf \
        curl \
        libcap \
        iputils \
    && setcap CAP_NET_BIND_SERVICE=+eip /usr/local/bin/hmcollector


ENV LOG_LEVEL=info
ENV POLLING_ENABLED=false
ENV RF_SUBSCRIBE_ENABLED=false
ENV REST_ENABLED=true
ENV KAFKA_HOST=localhost
ENV KAFKA_PORT=9092
ENV POLLING_INTERVAL=1
ENV PDU_POLLING_INTERVAL=30
ENV PRUNE_OLD_SUBSCRIPTIONS=true
ENV HSM_REFRESH_INTERVAL=30
ENV SM_URL=https://cray-smd
ENV REST_URL=localhost
ENV REST_PORT=80

# nobody 65534:65534
USER 65534:65534

CMD ["sh", "-c", "hmcollector"]
