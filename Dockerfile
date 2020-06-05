#
FROM golang:1.14 AS builder


WORKDIR /go/src/github.com/giongto35/cloud-game/

# system libs layer
RUN apt-get update && \
    apt-get install pkg-config libvpx-dev libopus-dev libopusfile-dev -y

# go deps layer
COPY go.mod go.sum ./
RUN go mod download

# app build layer
COPY ./ ./
RUN go install ./cmd/coordinator && \
    go install ./cmd/worker

EXPOSE 8000
EXPOSE 9000

# to add some light runtime image
