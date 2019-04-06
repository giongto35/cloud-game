From golang:1.12

RUN mkdir -p /go/src/github.com/giongto35/cloud-game
COPY . /go/src/github.com/giongto35/cloud-game/
WORKDIR /go/src/github.com/giongto35/cloud-game

# Install server dependencies
RUN apt-get update
#RUN apt-get install portaudio19-dev -y
RUN apt-get install libvpx-dev -y
RUN go get github.com/pion/webrtc
#RUN go get github.com/gordonklaus/portaudio
RUN go get github.com/gorilla/mux
RUN go get github.com/gorilla/websocket
RUN go install github.com/giongto35/cloud-game

EXPOSE 8000
