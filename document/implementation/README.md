# Web-based Cloud Gaming Service Implementation Document

## Code structure
.
├── cmd
│   ├── main.go
│   └── main_test.go
├── emulator: emulator internal
│   ├── director.go: coordinator of views
│   └── gameview.go: in game logic
├── overlord: coordinator of workers
├── games: roms list, no code logic
├── static: static file for front end
│   ├── js
│   │   └── ws.js: client logic
│   ├── game.html: frontend with gameboy ui
│   └── index_ws.html: raw frontend without ui
├── cws
│   └── cws.go: socket multiplexer library, used for signalling
├── webrtc
└── worker: integration between emulator + webrtc (communication) 

