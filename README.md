# Web-based Cloud Gaming Service

Web-based Cloud Gaming Service is an open source Cloud Gaming Service building on [WebRTC](https://github.com/pion) with browser target.  
  
.. aims to bring the most convenient gaming experience to gamer. You can play any games on your browser directly, which is fully compatible on multi-platform like Desktop, Android, IOS. This flexibility enables modern online gaming experience to retro games starting with NES in this current release.

![screenshot](static/img/landing-page.png)

## Demo
https://www.youtube.com/watch?v=qkjV2VIwVIo

## Try the game

For the best gaming experience, please select the closest region to you. 

Southeast Asia:  
* [http://nes.webgame2d.com](http://nes.webgame2d.com)
* [http://nes.playcloud.games](http://nes.playcloud.games)

US West (Los Angeles):  
* [http://us.nes.webgame2d.com](http://us.nes.webgame2d.com)
* [http://us.nes.playcloud.games](http://us.nes.playcloud.games)

US East: (Haven't hosted)  
* [http://use.nes.webgame2d.com](http://use.nes.webgame2d.com)
* [http://use.nes.playcloud.games](http://use.nes.playcloud.games)

Europe: (Haven't hosted)  
* [http://eu.nes.webgame2d.com](http://eu.nes.webgame2d.com)
* [http://eu.nes.playcloud.games](http://eu.nes.playcloud.games)  

Note: The current state of cloud gaming service lite are not optimized for production. The service will still experience lag in the case of heavy traffic. You can try hosting your own service following the instruction in the next session.

## Feature
1. Cloud gaming: Game logic is handled in server and streamed to user.
2. Cross-platform compatibility: The game is run on webbrowser, the most universal builtin app. No console, external app or devices are need.
3. Verially scaled: Services are distributed. The closest server with highest free resource will be assigned to user. (In progress)
4. Collaborative hosting: this is our invented term refering to the whole community can contribute to host the platform. Whenever the server is hosted and able to connect to the coordinator, it can join the cloud-gaming network and Collaboratively serve prod traffic. 
5. Local streaming: If you host a server on your local machine and play game on other device close to that, the game will automatically detect running local server and serve the traffic.
6. Cloud storage: Game state is storing on an online storage, so you can come back to continue playing in a game.

## Run on local by Docker

You try hosting the server yourself by running `./run_local_docker.sh`. It will spawn a docker environment and you can access the emulator on `localhost:8000`.  

## Development environment

Install Golang https://golang.org/doc/install  

Install dependencies  

  * Install [libvpx](https://www.webmproject.org/code/) and [pkg-config](https://www.freedesktop.org/wiki/Software/pkg-config/)
```
# Ubuntu
apt-get install -y pkg-config libvpx-dev

# MacOS
brew install libvpx pkg-config

# Windows
...
```
And golang dependencies
  * `go get github.com/pion/webrtc/`  
  * `go get github.com/gorilla/websocket`  
  * `go get gopkg.in/hraban/opus.v2`
  * `go get github.com/satori/go.uuid`
  * `go get cloud.google.com/go/storage`
  
And run 
  * `./run_local.sh`

## Collaborative hosting
  * `go run cmd/main.go -overlordhost ...` - start game workers (in charge of peerconnection) connecting to cloud-game network

## Code structure

```
.
├── cmd
│   ├── main.go
│   └── main_test.go
├── emulator: emulator internal
│   ├── director.go: coordinator of views
│   └── gameview.go: in game logic
├── overlord: coordinator of workers
├── games: roms list, no code logic
├── static: static file for front end
│   ├── js
│   │   └── ws.js: client logic
│   ├── gameboy.html: frontend with gameboy ui
│   └── index_ws.html: raw frontend without ui
├── cws
│   └── cws.go: socket multiplexer library, used for signalling
├── webrtc
└── worker: integration between emulator + webrtc (communication) 
```

## Follow up

This project demos the simplest cloud game with NES. Integrating with other emulator like GBA, NDS will also be possible. I'm welcome for the contribution.

## Credits

* *Pion* Webrtc team for the incredible Golang Webrtc library and their supports https://github.com/pion/webrtc/.  
* *fogleman* for the awesome nes emulator https://github.com/fogleman/nes.  
* *poi5305* for the video encoding https://github.com/poi5305/go-yuv2webRTC.  
* *bchanx* for the gameboy https://github.com/bchanx/animated-gameboy-in-css. 
* And last but not least, my longtime friend Tri as the co-author. 

## Contributor

Nguyen Huu Thanh  
https://www.linkedin.com/in/huuthanhnguyen/  

Tri Dang Minh  
https://trich.im  

