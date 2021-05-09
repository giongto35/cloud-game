## Streaming process description

This document describes the step-by-step process of media streaming through the whole application.

All begins when a player (p) opens the application page (index.html) served by the coordinator (c).
- The user's browser tries to open a new WebSocket connection to the coordinator — socket.init(roomId, zone) [web/js/network/socket.js:32](https://github.com/giongto35/cloud-game/blob/ae5260fb4726fd34cc0b0b05100dcc8457f52883/web/js/network/socket.js#L32)
> In the initial HTTP-to-WS Upgrade request query it may send two params: roomId — an identification number for already created game rooms if a user knows this ID 
> and opens the application page with the ?id=xxxxxx query param, zone — or, more precisely, region — serves the purpose of CDN and geographical segmentation of the streaming.
- On the coordinator side this request goes into a dedicated handler (/ws) — func (o *Server) WS(w http.ResponseWriter, r *http.Request) [pkg/coordinator/handlers.go:150](https://github.com/giongto35/cloud-game/blob/ae5260fb4726fd34cc0b0b05100dcc8457f52883/pkg/coordinator/handlers.go#L150)
- There, it unconditionally accepts a new WebSocket connection, registers user connection with some ID in storage, 
starts to listen to WebSocket messages from the user's side, and a new connection should be considered as established.
- Next, given provided query params, the coordinator tries to find a suitable worker (w) whose job is to directly stream emulated games to a user.
> This process of choosing the right worker is as follows: if there is no roomId param then coordinator gathers the full list of available workers, 
> filters them by a zone value if it's provided, sends through WebSocket back to the user a list of public worker endpoints (URLs) that the user can ping and send results back to 
> the coordinator after that coordinator chooses the fastest one and internally assigns that worker to that user. Alternatively, if the user did provide some roomId then the coordinator directly assigns that worker which runs that room (workers have 1:1 mapping to rooms).
> All the information exchange initiated from the worker side is handled in a separate endpoint (/wso) [pkg/coordinator/handlers.go#L81](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/coordinator/handlers.go#L81).
- Coordinator sends to the user some WebRTC info and the list of games available for playing. That's handled in [web/js/network/socket.js:57](https://github.com/giongto35/cloud-game/blob/ae5260fb4726fd34cc0b0b05100dcc8457f52883/web/js/network/socket.js#L57).
- From this point, the user's browser begins to initialize WebRTC connection to the worker — web/js/controller.js:413 → [web/js/network/rtcp.js:16](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/network/rtcp.js#L16).
- First, it sends init request through the WebSocket connection to the coordinator handler in [pkg/coordinator/useragenthandlers.go:17](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/coordinator/useragenthandlers.go#L17).
> Following a standard WebRTC call [negotiation procedure](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Signaling_and_video_calling), the coordinator acts as a mediator between users and workers.
> Signalling protocol there are just text messages through WebSocket transport.
- Coordinator notifies assigned to the user earlier worker that it wants to establish a new PeerConnection or (RTC call). That part is being handled in [pkg/worker/internalhandlers.go:42](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/worker/internalhandlers.go#L42).
> It is worth noting that the worker makes SDP offer and waits for an SDP answer.
- Worker initializes new WebRTC connection handler in func (w *WebRTC) StartClient(isMobile bool, iceCB OnIceCallback) (string, error) [pkg/webrtc/webrtc.go:103](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/webrtc/webrtc.go#L103).
- Worker through coordinator makes in parallel an SDP offer as well as sends ICE candidates that are handled on the coordinator side (what goes directly from the user) in [pkg/coordinator/useragenthandlers.go](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/coordinator/useragenthandlers.go), 
(what goes diriectly from the worker) in [pkg/coordinator/internalhandlers.go](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/pkg/coordinator/internalhandlers.go), and on the user side both in [web/js/network/socket.js:56](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/network/socket.js#L56) 
 and inside [web/js/network/rtcp.js](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/network/rtcp.js).
 - Browser on the user's side on SDP offer links remote streams to the HTML Video element in [web/js/controller.js:417](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/controller.js#L417), makes SDP answer and gathers remote ICE candidates until it's done (if receive an empty ICE candidate).
 - For the user's side successful WebRTC connection should be established when WebRTC datachannel will be opened here [web/js/network/rtcp.js:31](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/network/rtcp.js#L31).
 *And that should be it for the streaming part.*
 > At this point all the connections should be successfully established and the user should just send a request for game start. The coordinator should notify the worker about that and the worker should start pushing A/V frames, listen to the input through the already established direct to the user WebRTC data cahnnel.
 - After that user should send the game start request to the coordinator in [web/js/controller.js:153](https://github.com/giongto35/cloud-game/blob/a7d8e53dac2bbcf8306e0dafe3878644c760d368/web/js/controller.js#L153).

#### Possible streaming initialization failures
- Worker server should not have any closed UDP ports in order to be able to provide suitable ICE candidates.
- Coordinator should have at least one non-blocked TCP port (default: 8000) for HTTP/WebSocket signaling connections from users and workers.
- Browser should not block WebRTC and support it (check [here](https://test.webrtc.org/)).
