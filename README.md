# Cloud Gaming Service Lite

Cloud Gaming Service is an open source Cloud Gaming Service building on [WebRTC](https://github.com/pion).  
  
With Cloud gaming, you can play any of your favourite NES game directly on your browser without installing it on your machine. It also brings modern online multiplayer gaming experience to NES game, so two people can play the game together . Joystick gaming is the past :P.

## Feature
1. Can play NES games directly from browser.  
2. Can multiplayer over internet. A person host a game and the other person can join the same game as 1st or 2nd player.  
3. Save (S) and Load (L) at any point in time.  
4. If you save the roomID, next time you can come back to continue play in that room.  

## Demo
https://www.youtube.com/watch?v=qkjV2VIwVIo

## Try the game

For the best gaming experience, please select the closest region to you. 

Southeast Asia:  
* nes.webgame2d.com  
* nes.playcloud.games  

US West:  
* usw.nes.webgame2d.com  
* usw.nes.playcloud.games  

US East:  
* use.nes.webgame2d.com  
* use.nes.playcloud.games  

Europe:  
* eu.nes.webgame2d.com  
* eu.nes.playcloud.games  

## Run on local

You can host the server yourself by running `./run_local.sh`. It will spawn a docker environment and you can access the emulator on `localhost:8000`.  

You can open port, so other person can access your local machine and play the game together.  

# Credits

Pion Webrtc team for the incredible Golang Webrtc library and their supports https://github.com/pion/webrtc/  
fogleman for the awesome nes emulator https://github.com/fogleman/nes  
poi5305 for the video encoding https://github.com/poi5305/go-yuv2webRTC  
bchanx for the gameboy https://github.com/bchanx/animated-gameboy-in-css  
And last but not least, my longtime friend Tri as the co-author.  

# Contributor

Nguyen Huu Thanh  
https://www.linkedin.com/in/huuthanhnguyen/  

Tri Dang Minh  
https://trich.im  

