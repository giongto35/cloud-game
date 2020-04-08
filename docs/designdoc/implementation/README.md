# Web-based Cloud Gaming Service Implementation Document

## Code structure
```
.
├── cmd: service entrypoint
│   ├── main.go: Spawn coordinator or worker based on flag
│   └── main_test.go
├── static: static file for front end
│   ├── js
│   │   └── ws.js: client logic
│   ├── game.html: frontend with gameboy ui
│   └── index_ws.html: raw frontend without ui
├── coordinator: coordinator
│   ├── handlers.go: coordinator entrypoint
│   ├── browser.go: router listening to browser
│   └── worker.go: router listening to worker
├── games: roms list, no code logic
├── worker: integration between emulator + webrtc (communication) 
│   ├── room:
│   │   ├── room.go: room logic
│   │   └── media.go: video + audio encoding
│   ├── handlers.go: worker entrypoint
│   └── coordinator.go: router listening to coordinator
├── emulator: emulator internal
│   ├── nes: NES device internal
│   ├── director.go: coordinator of views
│   └── gameview.go: in game logic
├── cws
│   └── cws.go: socket multiplexer library, used for signaling
└── webrtc: webrtc streaming logic
```

## Room
Room is a fundamental part of the system. Each user session will spawn a room with a game running inside. There is a pipeline to encode images and audio and stream them out from emulator to user. The pipeline also listens to all input and streams to the emulator.

## Worker
Worker is an instance that can be provisioned to scale up the traffic. There are multiple rooms inside a worker. Worker will listen to coordinator events in `coordinator.go`.

## Coordinator
Coordinator is the coordinator, which handles all communication with workers and frontend. 
Coordinator will pair up a worker and a user for peer streaming. In WebRTC handshaking, two peers need to exchange their signature (Session Description Protocol) to initiate a peerconnection.
Events come from frontend will be handled in `coordinator/browser.go`. Events come from worker will be handled in `coordinator/worker.go`. Coordinator stays in the middle and relays handshake packages between workers and user.
