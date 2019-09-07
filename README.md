# Web-based Cloud Gaming Service
- [http://cloudretro.io](http://cloudretro.io)

CloudRetro, Open-source Cloud Gaming Service For Retro Games
  
This project aims to bring the most modern and convenient gaming experience to user as well as experiement the performance of Cloud-gaming technology. You can play any retro games on your browser directly, which is fully compatible on multi-platform like Desktop, Android, IOS. This flexibility also enables online gaming experience to retro games.  

\*Because there are limited servers in US East, US West, Eu, Singapore, you may experience some latency issues in particular regions. You can try hosting your own service following the instruction the next section to have a better sense of smoothness.  

**Video demo**: https://www.youtube.com/watch?v=koqWB1VKflo
Screenshot | Screenshot
:-------------------------:|:-------------------------:
![screenshot](document/img/landing-page-ps-hm.png)|![screenshot](document/img/landing-page-ps-x4.png)
![screenshot](document/img/landing-page-gb.png)|![screenshot](document/img/landing-page-front.png)

## Feature
1. Cloud gaming: Game logic and storage is hosted on cloud service. It reduces the cumbersome of game initialization. Images and audio are streamed to user in the most optimal way.
2. Cross-platform compatibility: The game is run on web browser, the most universal built-in app. No console, plugin, external app or devices are needed. Chrome with the latest version and fully WebRTC support is recommended for the game. 
3. Emulator agnostic: The game can be played directly without any extra effort to set up the gaming emulator or platform.
4. Vertically scaled: The infrastructure is designed to be able to scale under high traffic by adding more instances.
5. Cloud storage: Game state is storing on online storage, so you can come back to continue playing in a game.
6. Online multiplayer: Bring online multiplayer gaming to retro games. (In Road map)
7. Collaborate gameplay: Follow the idea of "Twitch Plays Pokemon", multiple players can play the same game together (In Road map)

## Run on local by Docker

You try running the server yourself by running `make run-docker`. It will spawn a docker environment and you can access the service on `localhost:8000`.  

## Development environment

Install Golang https://golang.org/doc/install . Because the project uses GoModule, so it requires Go1.11 version.

Install dependencies  

  * Install [libvpx](https://www.webmproject.org/code/) and [pkg-config](https://www.freedesktop.org/wiki/Software/pkg-config/)
```
# Ubuntu
apt-get install -y pkg-config libvpx-dev libopus-dev libopusfile-dev

# MacOS
brew install libvpx pkg-config opus opusfile

# Windows
... not tested yet ...
```

Because coordinator and workers needs to run simulateneously. Workers connects to coordinator.
1. Script
  * `make run`
  * The scripts includes build the binary using Go module
2. Manual
  * `go run cmd/main.go -overlordhost overlord` - spawn coordinator
  * `go run cmd/main.go -overlordhost ws://localhost:8000/wso` - spawn workers connecting to coordinator

## Wiki
- [Wiki](https://github.com/giongto35/cloud-game/wiki)

## FAQ
- [FAQ](https://github.com/giongto35/cloud-game/wiki/3.-FAQ)  

## Credits

* *Pion* Webrtc team for the incredible Golang Webrtc library and their supports https://github.com/pion/webrtc/.  
* *Nanoarch* Golang RetroArch https://github.com/libretro/go-nanoarch and https://retroarch.com.  
* *gen2brain* for the h264 go encoder https://github.com/gen2brain/x264-go
* *poi5305* for the video encoding https://github.com/poi5305/go-yuv2webRTC.  
* *fogleman* for the NES emulator https://github.com/fogleman/nes.  
* And last but not least, my longtime friend Tri as the co-author. 

## Contributor

Nguyen Huu Thanh  
https://www.linkedin.com/in/huuthanhnguyen/  

Tri Dang Minh  
https://trich.im  

