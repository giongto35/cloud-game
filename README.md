# Cloud Gaming Service Lite

Cloud Gaming Service is an open source Cloud Gaming Service building on ![WebRTC](https://github.com/pion/webrtc).

With Cloud gaming, you can play any of your favourite NES game directly on your browser without installing it on your machine. It also brings modern online multiplayer gaming experience to NES game, so two people can play the game together . Joystick gaming is the past :P.

## Feature
1. 

## Try the game

For the best gaming experience, please select the closest region to you.  
Southeast Asia:  
  nes.webgame2d.com  
  nes.playcloud.games  
US West:  
  usw.nes.webgame2d.com  
  usw.nes.playcloud.games  
US East:  
  use.nes.webgame2d.com  
  use.nes.playcloud.games  
Europe:  
  eu.nes.webgame2d.com  
  eu.nes.playcloud.games  

## Development environment

You can host the server yourself by running `./run_local.sh`. It will spawn a docker environment and run it on `localhost:8000`. 

## Architecture
![Techstack](document/images/techstack.jpg)

# Code architecture

## Document
[**Frontend**](cs2dclient)

[**Backend**](cs2dserver)

[**AIEnvironment**](gym)

## Codebase
```
.
├── cs2dclient
│   ├── index.html
│   ├── src
│   │   ├── config.js: javascript config
│   │   ├── index.html
│   │   ├── main.js
│   │   ├── sprites
│   │   │   ├── Leaderboard.js: Leaderboard object
│   │   │   ├── Map.js: Map object
│   │   │   ├── Player.js: Player object
│   │   │   └── Shoot.js: Shoot object
│   │   ├── states
│   │   │   ├── Boot.js Boot screen
│   │   │   ├── const.js
│   │   │   ├── Game.js: Game master
│   │   │   ├── message_pb.js: Protobuf Message
│   │   │   ├── Splash.js
│   │   │   └── utils.js
│   │   └── utils.js
├── cs2dserver
│   ├── buildwall.js
│   ├── cmd
│   │   └── server
│   │       └── server.go: Entrypoint running server
│   ├── game
│   │   ├── common
│   │   ├── config
│   │   │   └── 1.map: Map represented 0 and 1
│   │   ├── eventmanager
│   │   ├── gameconst
│   │   ├── game.go
│   │   ├── mappkg
│   │   ├── objmanager
│   │   ├── playerpkg
│   │   ├── shape
│   │   ├── shootpkg
│   │   ├── types.go
│   │   └── ws
│   │       ├── types.go
│   │       ├── wsclient.go
│   │       └── wshub.go
│   ├── generate.sh: Generate protobuf for server + client + AI environment
│   ├── message.proto
│   └── Message_proto
│       └── message.pb.go
├── gym: Training script for game (IN PROGRESS)
│   ├── cs2denv.py: Agent to communicate with server. Can connect to localhost or prod server
│   ├── lib
│   │   ├── common.py
│   ├── loadtest.py: Load test script to server
│   ├── message_pb2.py
│   ├── messenger.py
│   ├── test_env.py
│   ├── train2.py
│   └── train.py
├── Dockerfile
└── run_local.sh
```

# Credits

Pion Webrtc team for the incredible Golang Webrtc library and their supports https://github.com/pion/webrtc/
Fogleman for the awesome nes emulator https://github.com/fogleman/nes
poi5305 for the video encoding https://github.com/poi5305/go-yuv2webRTC
And last but not least, my longtime friend: https://github.com/trichimtrich for his

# Contributor

Nguyen Huu Thanh  
https://www.linkedin.com/in/huuthanhnguyen/

Tri Dang Minh
https://trich.im

