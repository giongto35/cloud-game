<h1 align="center">
  <a href="https://pion.ly"><img src="./.github/gopher-pion.png" alt="Pion TURN" height="250px"></a>
  <br>
  Pion TURN
  <br>
</h1>
<h4 align="center">An extendable TURN server written in Go</h4>
<p align="center">
  <a href="https://pion.ly"><img src="https://img.shields.io/badge/pion-turn-gray.svg?longCache=true&colorB=brightgreen" alt="Pion TURN"></a>
  <a href="http://gophers.slack.com/messages/pion"><img src="https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen" alt="Slack Widget"></a>
  <a href="https://waffle.io/pion/webrtc"><img src="https://img.shields.io/badge/pm-waffle-gray.svg?longCache=true&colorB=brightgreen" alt="Waffle board"></a>
  <br>
  <a href="https://travis-ci.org/pion/turn"><img src="https://travis-ci.org/pion/turn.svg?branch=master" alt="Build Status"></a>
  <a href="https://godoc.org/github.com/pion/turn"><img src="https://godoc.org/github.com/pion/turn?status.svg" alt="GoDoc"></a>
  <a href="https://codecov.io/gh/pion/turn"><img src="https://codecov.io/gh/pion/turn/branch/master/graph/badge.svg" alt="Coverage Status"></a>
  <a href="https://goreportcard.com/report/github.com/pion/turn"><img src="https://goreportcard.com/badge/github.com/pion/turn" alt="Go Report Card"></a>
  <a href="https://www.codacy.com/app/Sean-Der/turn"><img src="https://api.codacy.com/project/badge/Grade/d53ec6c70576476cb16c140c2964afde" alt="Codacy Badge"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
</p>
<br>

A TURN server written in Go that is designed to be scalable, extendable and embeddable out of the box.
For simple use cases it only requires downloading 1 static binary, and setting 3 options.

See [DESIGN.md](DESIGN.md) for the the features it offers, and future goals.

### Quick Start
If you want just a simple TURN server with a few static usernames `simple-turn` will perfectly suit your purposes. If you have
custom requirements such as a database proceed to extending.

`simple-turn` is a single static binary, and all config is driven by environment variables. On a fresh Linux AWS instance these are all the steps you would need.
```
$ wget -q https://github.com/pion/turn/releases/download/1.0.3/simple-turn-linux-amd64
$ chmod +x simple-turn-linux-amd64
$ export USERS='user=password foo=bar'
$ export REALM=my-server.com
$ export UDP_PORT=3478
$ ./simple-turn-linux-amd64
````

To explain what every step does
* Download simple-turn for Linux x64, see [release](https://github.com/pion/turn/releases) for other platforms
* Make it executable
* Configure auth, in the form of `USERNAME=PASSWORD USERNAME=PASSWORD` with no limits
* Set your realm, this is the public URL or name of your server
* Set the port you listen on, 3478 is the default port for TURN

That is it! Then to use your new TURN server your WebRTC config would look like
```
{ iceServers: [{
  urls: "turn:YOUR_SERVER"
  username: "user",
  credential: "password"
}]
```
---

If you are using Windows you would set these values in Powershell by doing. Also make sure your firewall is configured properly.
```
> $env:USERS = "user=password foo=bar"
> $env:REALM = "my-server.com"
> $env:UDP_PORT = 3478
```
### Extending
See [simple-turn](https://github.com/pion/turn/blob/master/cmd/simple-turn/main.go)

pion-turn can be configured by implementing [these callbacks](https://github.com/pion/turn/blob/master/turn.go#L11) and by passing [these arguments](https://github.com/pion/turn/blob/master/turn.go#L11)

All that `simple-turn` does is take environment variables, and then uses the same API.


### Developing
For developing a Dockerfile is available with features like hot-reloads, and is meant to be volume mounted.
Make sure you also have github.com/pion/pkg in your path, or you can exclude the second volume mount.

This is only meant for development, see [demo-conference](https://github.com/pion/demo-conference)
to see TURN usage as a user.
```
docker build -t turn .
docker run -v $(pwd):/usr/local/src/github.com/pion/turn -v $(pwd)/../pkg:/usr/local/src/github.com/pion/pkg turn
```

Currently only Linux is supported until Docker supports full (host <-> container) networking on Windows/OSX

### RFCs
#### Implemented
* [RFC 5389: Session Traversal Utilities for NAT (STUN)](https://tools.ietf.org/html/rfc5389)
* [RFC 5766: Traversal Using Relays around NAT (TURN)](https://tools.ietf.org/html/rfc5766)

#### Planned
* [RFC 6062: Traversal Using Relays around NAT (TURN) Extensions for TCP Allocations](https://tools.ietf.org/html/rfc6062)
* [RFC 6156: Traversal Using Relays around NAT (TURN) Extension for IPv6](https://tools.ietf.org/html/rfc6156)

### Community
Pion has an active community on the [Golang Slack](https://invite.slack.golangbridge.org/). Sign up and join the **#pion** channel for discussions and support. You can also use [Pion mailing list](https://groups.google.com/forum/#!forum/pion).

We are always looking to support **your projects**. Please reach out if you have something to build!

### Contributing
Check out the [CONTRIBUTING.md](CONTRIBUTING.md) to join the group of amazing people making this project possible:

* [Michiel De Backker](https://github.com/backkem) - *Documentation*
* [Ingmar Wittkau](https://github.com/iwittkau) - *STUN client*
* [John Bradley](https://github.com/kc5nra) - *Original Author*
* [jose nazario](https://github.com/paralax) - *Documentation*
* [Mészáros Mihály](https://github.com/misi) - *Documentation*
* [Mike Santry](https://github.com/santrym) - *Mascot*
* [Sean DuBois](https://github.com/Sean-Der) - *Original Author*
* [winds2016](https://github.com/winds2016) - *Windows platform testing*
* [songjiayang](https://github.com/songjiayang) - *SongJiaYang*
* [Yutaka Takeda](https://github.com/enobufs) - *vnet*
* [namreg](https://github.com/namreg) - *Igor German*
* [Aleksandr Razumov](https://github.com/ernado) - *protocol*

### License
MIT License - see [LICENSE.md](LICENSE.md) for full text
