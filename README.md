# Web-based Cloud Gaming Service
- [http://cloudretro.io](http://cloudretro.io)

- [**Game Instruction**](document/instruction/)

**Video demo**: https://www.youtube.com/watch?v=koqWB1VKflo

---

CloudRetro, Open source Web-based Cloud Gaming Service building on [WebRTC](https://github.com/pion) and [LibRetro](https://retroarch.com/).  
  
This project aims to bring the most modern and convenient gaming experience to user. You can play any retro games on your browser directly, which is fully compatible on multi-platform like Desktop, Android, IOS. This flexibility also enables online gaming experience to retro games.  

Note: **Due to the high cost of hosting, I will Hibernate the servers for a while. I'm working on a big change and will turn on hosting again. Sorry for that :(**  
You can try hosting your own service following the instruction in the next session.  

Screenshot | Screenshot
:-------------------------:|:-------------------------:
![screenshot](document/img/landing-page-ps-hm.png)|![screenshot](document/img/landing-page-ps-x4.png)
![screenshot](document/img/landing-page-gb.png)|![screenshot](document/img/landing-page.gif)

## Feature
1. Cloud gaming: Game logic is hosted on a remote server. User doesn't have to install or setup anything. Images and audio are streamed to user in the most optimal way.
2. Cross-platform compatibility: The game is run on webbrowser, the most universal built-in app. No console, plugin, external app or devices are needed. The device must support webRTC to perform streaming. Joystick is also supported.
4. Emulator agnostic: The game can be play directly without emulator selection and initialization as long as the its cores are supported by RetroArch.
3. Vertically scaled + Load balancing: We can add more machines to handle more traffic. The closest server with highest free resource will be assigned to user.
5. Cloud storage: Game state is storing on online storage, so you can come back to continue playing in a game.

## Run on local by Docker

You try hosting the server yourself by running `./run_local_docker.sh`. It will spawn a docker environment and you can access the emulator on `localhost:8000`.  

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

And run 
  * `./run_local.sh`
  * The scripts includes build the binary using Go module

## Documentation
- ✏ [Design Doc](document/designdoc/)  
- 💿 [Implementation Doc](document/implementation/)  

## FAQ
- [FAQ](https://github.com/giongto35/cloud-game/wiki/3.-FAQ)  

## Credits

* *Pion* Webrtc team for the incredible Golang Webrtc library and their supports https://github.com/pion/webrtc/.  
* *Nanoarch* Golang RetroArch https://github.com/libretro/go-nanoarch and https://retroarch.com.  
* *fogleman* for the awesome NES emulator https://github.com/fogleman/nes.  
* *poi5305* for the video encoding https://github.com/poi5305/go-yuv2webRTC.  
* And last but not least, my longtime friend Tri as the co-author. 

## Contributor

Nguyen Huu Thanh  
https://www.linkedin.com/in/huuthanhnguyen/  

Tri Dang Minh  
https://trich.im  

