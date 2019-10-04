# CloudRetro
**Open-source Cloud Gaming Service For Retro Games**  
**Video demo**: https://www.youtube.com/watch?v=GUBrJGAxZZg

## Introduction
This project aims to experiment Cloud-gaming performance with [WebRTC](https://github.com/pion/webrtc/) and [libretro](https://www.libretro.com/), as well as trying to deliver the most modern and convenient gaming experience through the technology. Theoretically, games are run on remote servers and media are streamed to the player optimally to ensure the most comfortable user interaction. It opens the ability to play any retro games on web-browser directly, which are fully compatible with multi-platform like Desktop, Android, ~~IOS~~. This flexibility also enables online gaming experience to retro games.  

## Try the service at
**[http://cloudretro.io](http://cloudretro.io)**  
*Chrome and Chrome on Android is recommended. It's not working on iPhone and some other explorers. Click help button in the game UI to see keyboard mapping.*  

\*In ideal network condition and less resource contention on servers, the game will run smoothly as in the video demo. Because I only hosted the platform on limited servers in US East, US West, Eu, Singapore, you may experience some latency issues + connection problem. You can try hosting the service following the instruction the next section to have a better sense of performance.  

|                   Screenshot                   |                   Screenshot                   |
| :--------------------------------------------: | :--------------------------------------------: |
| ![screenshot](docs/img/landing-page-ps-hm.png) | ![screenshot](docs/img/landing-page-ps-x4.png) |
|  ![screenshot](docs/img/landing-page-gb.png)   | ![screenshot](docs/img/landing-page-front.png) |

## Feature
1. Cloud gaming: Game logic and storage is hosted on cloud service. It reduces the cumbersome of game initialization. Images and audio are streamed to user in the most optimal way using advanced encoding technology.
2. Cross-platform compatibility: The game is run on web browser, the most universal built-in app. No console, plugin, external app or devices are needed. Chrome with the latest version and fully WebRTC support is recommended for the game. 
3. Emulator agnostic: The game can be played directly without any extra effort to set up the gaming emulator or platform.
4. Vertically scaled: The infrastructure is designed to be able to scale under high traffic by adding more instances.
5. Cloud storage: Game state is storing on online storage, so you can come back and continue playing your incomplete game later.
6. Online multiplayer: Bring online multiplayer gaming to retro games. (In Road map)
7. Collaborate gameplay: Follow the idea of "Twitch Plays Pokemon", multiple players can play the same game together (In Road map)

## Run on local by Docker

You try running the server directly by `make dev.run-docker`. It will spawn a docker environment and you can access the service on `localhost:8000`.  

## Development environment

Install Golang https://golang.org/doc/install . Because the project uses GoModule, so it requires Go1.11 version.

(Optional) Setup MSYS2 (MinGW) environment if you are using Windows:
  * Please refer to the Libretro [doc](https://docs.libretro.com/development/retroarch/compilation/windows/#environment-configuration) for initial environment setup
  * Add Golang installation path into your .bashrc
    ```
    $ echo 'export PATH=/c/Go/bin:$PATH' >> ~/.bashrc
    ```
  * Install dependencies as described down bellow
  * Copy required [Libretro Core DLLs](http://buildbot.libretro.com/nightly/windows/x86_64/) into the `cloud-game\assets\emulator\libretro\cores` folder and replace existing Linux SOs in the `cloud-game\pkg\config\config.go` EmulatorConfig object.
  * Use `C:\msys64\mingw64.exe` for building

Install dependencies  

  * Install [libvpx](https://www.webmproject.org/code/), [libopus](http://opus-codec.org/), [pkg-config](https://www.freedesktop.org/wiki/Software/pkg-config/)
```
# Ubuntu
apt-get install -y pkg-config libvpx-dev libopus-dev libopusfile-dev

# MacOS
brew install libvpx pkg-config opus opusfile

# Windows (MSYS2)
pacman -S --noconfirm --needed git make mingw-w64-x86_64-toolchain mingw-w64-x86_64-pkg-config mingw-w64-x86_64-dlfcn mingw-w64-x86_64-libvpx mingw-w64-x86_64-opusfile
```

Because the coordinator and workers need to run simultaneously. Workers connect to the coordinator.
1. Script
  * `make dev.run`
  * The scripts spawns 2 processes one in the background and one in foreground
2. Manual
  * Need to run coordinator and worker separately in two session
  * `go run cmd/overlord/main.go` - spawn coordinator
  * `go run cmd/overworker/main.go --overlordhost ws://localhost:8000/wso` - spawn workers connecting to coordinator

## Wiki
- [Wiki](https://github.com/giongto35/cloud-game/wiki)

## FAQ
- [FAQ](https://github.com/giongto35/cloud-game/wiki/FAQ)  

## Credits

* *Pion* Webrtc team for the incredible Golang Webrtc library and their supports https://github.com/pion/webrtc/.  
* *libretro/kivutar* Golang libretro https://github.com/libretro/go-nanoarch and https://www.libretro.com/.  
* *gen2brain* for the h264 go encoder https://github.com/gen2brain/x264-go
* *poi5305* for the video encoding https://github.com/poi5305/go-yuv2webRTC.  
* *fogleman* for the NES emulator https://github.com/fogleman/nes.  
* And last but not least, my longtime friend Tri as the co-author. 

## Contributor

Nguyen Huu Thanh  
https://www.linkedin.com/in/huuthanhnguyen/  

Tri Dang Minh  
https://trich.im  

