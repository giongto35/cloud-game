# Klog, Web-based Cloud Gaming Service
- [http://cloud.webgame2d.com](http://cloud.webgame2d.com)

- [**Game Instruction**](document/instruction/)
For the best gaming experience, please select the closest region to you.   

**Video demo**: https://www.youtube.com/watch?v=koqWB1VKflo

---

Klog is an open source Cloud Gaming Service building on [WebRTC](https://github.com/pion) using browser as the main platform.  
  
Klog aims to bring the most convenient gaming experience to gamer. You can play any games on your browser directly, which is fully compatible on multi-platform like Desktop, Android, IOS. This flexibility enables modern online gaming experience to retro games starting with NES in this current release.  

Note: The current state of Klog are not optimized for production. The service will still experience lag under heavy traffic. You can try hosting your own service following the instruction in the next session.  

![screenshot](document/img/landing-page.gif)

## Feature
1. Cloud gaming: Game logic is hosted on a remote server. User doesn't have to install or setup anything. Images and audio are streamed to user in the most optimal way.
2. Cross-platform compatibility: The game is run on webbrowser, the most universal built-in app. No console, plugin, external app or devices are needed. The device must support webRTC to perform streaming. Joystick is also supported.
3. Vertically scaled: We can add more machines to handle more traffic. The closest server with highest free resource will be assigned to user (In development).
4. Cloud storage: Game state is storing on online storage, so you can come back to continue playing in a game.

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

### How does the project start?

- The project is inspired by Google Stadia. The most important question comes to everyone mind is how good is the latency? Will gaming experience is affected by the network? I did some researches on that topic and WebRTC seems to be the most suitable protocol for that purpose. I limited the project scope and made a POC of Cloud-gaming. The result indeed looks very promising.  

### How good is the result

- My estimation is that it requires 10Mbps per NES game session with resolution 256 * 240.

### Why is the game lag for some people?

- Cloud-gaming is very network-sensitive. It requires the server is close to the user, so please pick the nearest server to you. If there is not, you can try hosting the platform on your own machine followed above instruction and test.  
- Cloud-gaming is based on WebRTC peer to peer, so there are some cases direct communication is not possible because of the firewall. In that case, relay communication happens and the game is not smooth. You can find a public network and retry.  
- The current state of project is hosted on a limited resource, so during high traffic, the game might got lag due to CPU is overused, not because of the network. Besides, my memory management is not working properly sometimes and game sessions are not fully separated, so the game session can lag over time. In that case, please reload or continue your game by clicking share and reopen the old game.  

### Why NES but not some more modern games?

- For the purpose of latency demonstration and fast iteration, I picked NES but integrating with other emulators like GBA, NDS and even Playstation is also possible. For High-end games, there will be problems with hardware and infrastructure. Google has a lot of resource and its distributed GPU will enhance this cloud-gaming use case. My resource is not as abundant, so I consider NES emulator for my first step.

### Why Web browser as the main platform?

- Web browser is most universal built-in app and it will bring the most convenient and modern gaming experience together with cloud-gaming. You can try the platform on Android. Unfortunately, IOS doesn't support WebRTC protocol yet, [http://iswebrtcreadyyet.com/](http://iswebrtcreadyyet.com/)

## Credits

* *Pion* Webrtc team for the incredible Golang Webrtc library and their supports https://github.com/pion/webrtc/.  
* *fogleman* for the awesome NES emulator https://github.com/fogleman/nes.  
* *poi5305* for the video encoding https://github.com/poi5305/go-yuv2webRTC.  
* And last but not least, my longtime friend Tri as the co-author. 

## Contributor

Nguyen Huu Thanh  
https://www.linkedin.com/in/huuthanhnguyen/  

Tri Dang Minh  
https://trich.im  

