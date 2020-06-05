#
FROM golang:1.14 AS build

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

# base image
FROM debian:10-slim
RUN apt-get update && \
    apt-get install libvpx-dev libopus-dev libopusfile-dev -y && \
    rm -rf /var/lib/apt/lists/*
COPY --from=build /go/bin/ /
COPY ./web /web

EXPOSE 8000
EXPOSE 9000
