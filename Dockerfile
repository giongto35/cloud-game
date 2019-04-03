From golang:1.11

RUN mkdir -p /go/src/github.com/giongto35/game-online
COPY . /go/src/github.com/giongto35/game-online/

# Install server dependencies
RUN go get github.com/pions/webrtc
RUN go get github.com/go-gl/glfw/v3.2/glfw
