# The base cloud-game image
ARG BUILD_PATH=/go/src/github.com/giongto35/cloud-game

# build image
FROM golang:1.14 AS build
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

RUN apt-get update && apt-get install -y \
    libvpx5 \
    libopus0 \
    libopusfile0 \
    libsdl2-2.0-0 \
    libgl1-mesa-glx \
    xvfb \
  && rm -rf /var/lib/apt/lists/*

COPY --from=build ${BUILD_PATH}/bin/ ./
# it assumed there are only binary files yet
RUN cp -s $(pwd)/* /usr/local/bin
COPY web ./web
COPY assets/emulator/libretro/cores/*.so \
     assets/emulator/libretro/cores/*.cfg \
     ./assets/emulator/libretro/cores/

EXPOSE 8000 9000 3478/tcp 3478/udp
