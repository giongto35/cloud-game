# Cloud Gaming Service Lite

Cloud Gaming Service is an open source Cloud Gaming Service building on [WebRTC](https://github.com/pion).  
  
With cloud gaming, you can play any of your favourite NES game directly on your browser without installing it. It also brings modern online multiplayer gaming experience to classic NES game, so two people can play the game together.

![screenshot](static/img/landing-page.png)

## Feature
1. Can play NES games directly from browser.  
2. Immediately startup, no need to install.
2. Can multiplayer over internet. A person host a game and the other person can join the same game as 1st or 2nd player.  
3. Save (S) and Load (L) at any point in time.  
4. If you save the roomID, next time you can come back to continue play in that room.  

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

## Run on local

You can host the server yourself by running `./run_local.sh`. It will spawn a docker environment and you can access the emulator on `localhost:8000`.  

You can open port, so other person can access your local machine and play the game together.  

## Development environment

Install Golang https://golang.org/doc/install  

Install dependencies  

  * `apt-get install libvpx-dev -y` (or alternatives of libvpx on other OS)  
  * `go get github.com/pion/webrtc/`  
  * `go get github.com/gorilla/mux`  
  * `go get github.com/gorilla/websocket`  

Then we can run the main directly
  * `go run main.go`

## Code structure

```
├── games: roms list, no code logic
├── nes: emulator internal
├── static: static file for front end
│   ├── js
│   │   └── ws.js: client logic
│   ├── gameboy.html: frontend with gameboy ui
│   └── index_ws.html: raw frontend without ui
├── ui
│   ├── director.go: coordinator of views
│   └── gameview.go: in game logic
├── vpx-encoder: vp8 encoding images -> video track
├── webrtc: peer to peer communication
├── main.go: integration between emulator + webrtc (communication) + websocket (signalling)
└── run_local.sh
```


## Follow up

This project demos the simplest cloud game with NES. Integrating with other emulator like GBA, NDS will also be possible. I will welcome for the contribution.

## Credits

* *Pion* Webrtc team for the incredible Golang Webrtc library and their supports https://github.com/pion/webrtc/  
* *fogleman* for the awesome nes emulator https://github.com/fogleman/nes  
* *poi5305* for the video encoding https://github.com/poi5305/go-yuv2webRTC  
* *bchanx* for the gameboy https://github.com/bchanx/animated-gameboy-in-css  
* And last but not least, my longtime friend Tri as the co-author.  

## Contributor

Nguyen Huu Thanh  
https://www.linkedin.com/in/huuthanhnguyen/  

Tri Dang Minh  
https://trich.im  

