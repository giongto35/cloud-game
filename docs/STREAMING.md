## Streaming process description

This document describes the step-by-step process of media streaming in all parts of the application.

```
┌──────────────┐      ┌───────────────┐    ┌──────────────┐
│  USER AGENT  │      │  COORDINATOR  │    │  WORKER...n  │
├──────────────┤      ├───────────────┤    ├──────────────┤
│       TCP/WS ├──1──►│ WS ──────► WS │◄───┤ TCP/WS       │
│              │      │ ▲     2     │ │    │              │
│              │      │ └───────────┘ │    │              │
│              │      └───────────────┘    │              │
│      UDP/RTP │◄─────────────3────────────┤ UDP/RTP      │
│      AUDIO < │  OPUS                     │ AUDIO        │
│      VIDEO < │  VP8/H264                 │ VIDEO        │
│      DATA  > │  010101                   │ DATA         │
└──────────────┘                           └──────────────┘
```

The app is based on WebRTC technology which allows the server to stream media and exchange data with ultra-low latencies. An essential part of these types of P2P connections is the signaling process. It's implemented as a custom text-based messaging protocol on top of WebSocket (quite similarly to [WAMP](https://wamp-proto.org)). The app supports both STUN and TURN protocols for NAT traversal or ICE. In terms of supported codecs, it can stream h264, VP8, and OPUS media.

The streaming process begins when a user opens the main application page (index.html) served by the coordinator.
- The user's browser tries to open a new WebSocket connection to the coordinator — socket.init(roomId, zone) [web/js/network/socket.js:32](https://github.com/giongto35/cloud-game/blob/ae5260fb4726fd34cc0b0b05100dcc8457f52883/web/js/network/socket.js#L32)
> In the initial WebSocket Upgrade request query it may send two params: roomId — an identification number for existing game rooms stored in the URL query of the application page (i.e. app.com/?id=xxxxxx), zone — or, more precisely, region — serves the purpose of CDN and geographical segmentation of the streaming.
- On the coordinator side this request goes into a dedicated handler (/ws) — func (o *Server) WS(w http.ResponseWriter, r *http.Request) [pkg/coordinator/handlers.go:150](https://github.com/giongto35/cloud-game/blob/ae5260fb4726fd34cc0b0b05100dcc8457f52883/pkg/coordinator/handlers.go#L150)
- There, it unconditionally accepts the WebSocket connection and tags it with some ID, so it will be listening to messages from the user's side. Here a new client connection should be considered as established.
- Next, given provided query params, the coordinator tries to find a suitable worker whose job — directly stream games to a user.
> This process of choosing the right worker is following: if there is no roomId param, then the coordinator gathers the full list of available workers, filters them by a zone value (if provided), returns the user a list of public URLs, which he can ping and send results back to the coordinator. After that, the coordinator links the fastest one with the user. Alternatively, if the user did provide some roomId, then the coordinator directly assigns a worker with that room (workers have 1:1 mapping to rooms or games).
> All the information exchange initiated from the worker side is handled in a separate endpoint (/wso) [pkg/coordinator/handlers.go#L81](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/coordinator/handlers.go#L81).
- Coordinator sends to the user ICE servers and the list of games available for playing. That's handled in [web/js/network/socket.js:57](https://github.com/giongto35/cloud-game/blob/ae5260fb4726fd34cc0b0b05100dcc8457f52883/web/js/network/socket.js#L57).
- From this point, the user's browser begins to initialize WebRTC connection to the worker — web/js/controller.js:413 → [web/js/network/rtcp.js:16](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/network/rtcp.js#L16).
- First, it sends init request through the WebSocket connection to the coordinator handler in [pkg/coordinator/useragenthandlers.go:17](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/coordinator/useragenthandlers.go#L17).
> Following a standard WebRTC call [negotiation procedure](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Signaling_and_video_calling), the coordinator acts as a mediator between users and workers. The signaling protocol here is a text messaging through WebSocket transport.
- Coordinator notifies the user's worker that it wants to establish a new PeerConnection (call). That part is being handled in [pkg/worker/internalhandlers.go:42](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/worker/internalhandlers.go#L42). It is worth noting that it is a worker who makes SDP offer and waits for an SDP answer.
- Worker initializes new WebRTC connection handler in func (w *WebRTC) StartClient(isMobile bool, iceCB OnIceCallback) (string, error) [pkg/webrtc/webrtc.go:103](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/webrtc/webrtc.go#L103).
- Then through the coordinator it makes simultaneously an SDP offer as well as sends ICE candidates that are handled on the coordinator side (from the user) in [pkg/coordinator/useragenthandlers.go](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/coordinator/useragenthandlers.go), 
(from the worker) in [pkg/coordinator/internalhandlers.go](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/coordinator/internalhandlers.go), and on the user side both in [web/js/network/socket.js:56](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/network/socket.js#L56) and inside [web/js/network/rtcp.js](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/network/rtcp.js).
 - Browser on the user's side after SDP offer links remote streams to the HTML Video element in [web/js/controller.js:417](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/controller.js#L417), makes SDP answer and gathers remote ICE candidates until it's done (if receive an empty ICE candidate).
 - For the user's side a successful WebRTC connection should be considered established when WebRTC datachannel is opened here [web/js/network/rtcp.js:31](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/network/rtcp.js#L31).
 *And that should be it for the streaming part.*
 > At this point all the connections should be successfully established and the user's ready for a game to start. The coordinator should notify the worker about that fact and the worker starts pushing media frames, listen to the input through the direct to the user WebRTC data channel.
 - Then the user may send the game start request to the coordinator in [web/js/controller.js:153](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/controller.js#L153).

#### Streaming requirements
- Workers should not have any closed UDP ports to be able to provide suitable ICE candidates.
- Coordinator should have at least one non-blocked TCP port (default: 8000) for HTTP/WebSocket signaling connections from users and workers.
- Browser should not block WebRTC and support it (check [here](https://test.webrtc.org/)).
