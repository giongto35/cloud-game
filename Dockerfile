From golang:1.12

RUN mkdir -p /go/src/github.com/giongto35/cloud-game
COPY . /go/src/github.com/giongto35/cloud-game/
WORKDIR /go/src/github.com/giongto35/cloud-game

# Install server dependencies
RUN apt-get update

RUN apt-get install pkg-config libvpx-dev libopus-dev libopusfile-dev -y

RUN go get gopkg.in/hraban/opus.v2
RUN go get github.com/pion/webrtc
RUN go get github.com/gorilla/websocket
RUN go get github.com/satori/go.uuid
RUN go get cloud.google.com/go/storage
RUN go install github.com/giongto35/cloud-game/cmd

EXPOSE 8000
