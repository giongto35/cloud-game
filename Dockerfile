# The base cloud-game image
ARG BUILD_PATH=/go/src/github.com/giongto35/cloud-game

# build image
FROM golang:1.15 AS build
ARG BUILD_PATH
WORKDIR ${BUILD_PATH}

# system libs layer
RUN apt-get update && apt-get install -y \
    make \
    pkg-config \
    libvpx-dev \
    libopus-dev \
    libopusfile-dev \
    libsdl2-dev \
 && rm -rf /var/lib/apt/lists/*

# go deps layer
COPY go.mod go.sum ./
RUN go mod download

# app build layer
COPY ./ ./
RUN make build

# base image
FROM debian:10-slim
ARG BUILD_PATH
WORKDIR /usr/local/share/cloud-game

RUN apt-get update && apt-get install --no-install-recommends -y \
    ca-certificates \
    libvpx5 \
    libopus0 \
    libopusfile0 \
    libsdl2-2.0-0 \
    libgl1-mesa-glx \
    xvfb \
  && rm -rf /var/lib/apt/lists/*

COPY --from=build ${BUILD_PATH}/bin/ ./
RUN cp -s $(pwd)/* /usr/local/bin
COPY web ./web
COPY assets/cores/*.so \
     assets/cores/*.cfg \
     ./assets/cores/
COPY configs ./configs

EXPOSE 8000 9000 3478/tcp 3478/udp
