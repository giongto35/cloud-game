# Cloud Gaming Service Design Document

Cloud Gaming Service contains multiple workers for gaming streams and a coordinator for distributing traffic and pairing
up connections.

## Coordinator

Coordinator is a web-frontend, load balancer and signalling server for WebRTC.

```
                                                                  WORKERS
                                                                 ┌──────────────────────────────────┐
                                                                 │                                  │
                                                                 │  REGION 1   REGION 2   REGION N  │
                                                                 │    (US)       (DE)       (XX)    │
                                                                 │  ┌──────┐   ┌──────┐   ┌──────┐  |
 COORDINATOR                                                     │  │WORKER│   │WORKER│   │WORKER│  |
┌───────────┐                                                    │  └──────┘   └──────┘   └──────┘  |
│           │ ───────────────────────HEALTH────────────────────► │     •          •          •      |
│  HTTP/WS  │ ◄─────────────────────REG/DEREG─────────────────── │     •          •          •      |
│┌─────────┐│                                                    │     •          •          •      |
│|         |│                          USER                      │  ┌──────┐*  ┌──────┐   ┌──────┐  |
│└─────────┘│                        ┌──────┐                    │  │WORKER│   │WORKER│   │WORKER│  |
│           │ ◄──(1)CONNECT───────── │      │ ────(3)SELECT────► │  └──────┘   └──────┘   └──────┘  |
│           │ ───(2)LIST WORKERS───► │      │ ◄───(4)STREAM───── │                                  │
└───────────┘                        └──────┘                    │     * MULTIPLAYER                │
                                                                 │         ┌──────┐────► ONE GAME   │
                                                                 │    ┌───►│WORKER│◄──┐             │
                                                                 │    │    └──────┘   │             │
                                                                 │    │     ▲    ▲    │             │
                                                                 │   ┌┴─┐   │    │   ┌┴─┐           |
                                                                 │   │U1│ ┌─┴┐  ┌┴─┐ │U4│           |
                                                                 │   └──┘ │U2│  │U3│ └──┘           |
                                                                 │        └──┘  └──┘                |
                                                                 │                                  |
                                                                 └──────────────────────────────────┘
```

- (1) A user opens the main page of the app in the browser, i.e. connects to the coordinator.
- (2) The coordinator searches and serves a list of most suitable workers to the user.
- (3) The user proceeds with latency check of each worker from the list, then coordinator collects user-to-worker
  latency data and picks the best candidate.
- (4) The coordinator sets up peer-to-peer connection between a worker and the user based on the WebRTC protocol and a
  game hosted on the worker is streamed to the user.

## Worker

Worker is responsible for running and streaming games to users.

```
 WORKER                                                                                   
┌─────────────────────────────────────────────────────────────────┐                                          
│        EMULATOR                                       WEBRTC    │             BROWSER                      
│  ┌─────────────────┐            ENCODER             ┌────────┐  │           ┌──────────┐                   
│  │                 │          ┌─────────┐           |  DMUX  |  | ───RTP──► |  WEBRTC  |                   
│  │  AUDIO SAMPLES  │ ──PCM──► │         │ ──OPUS──► │  ┌──►  │  │ ◄──SCTP── |          |                   
│  │  VIDEO FRAMES   │ ──RGB──► │         │ ──H264──► │  └──►  |  |           └──────────┘      COORDINATOR  
│  │                 │          └─────────┘           │        │  │                •          ┌─────────────┐
│  │                 │                                |  MUX   |  | ───TCP──────── • ───────► |  WEBSOCKET  |
│  │                 │                                │  ┌──   │  │                •          └─────────────┘
|  |                 |            BINARY              |  ▼     |  |             BROWSER                      
│  │  INPUT STATE    │ ◄───────────────────────────── │  •     │  │           ┌──────────┐                   
│  │                 │                                │  ▲     │  │ ───RTP──► |  WEBRTC  |                   
│  └─────────────────┘            HTTP/WS             |  └──   |  │ ◄──SCTP── │          │                   
│                               ┌─────────┐           └────────┘  │           └──────────┘                   
|                               |         |                       |                                          
|                               └─────────┘                       |                                          
└─────────────────────────────────────────────────────────────────┘                                         
```

- After coordinator matches the most appropriate server (peer 1) to the user (peer 2), a WebRTC peer-to-peer handshake
  will be conducted. The coordinator will help initiate the session between the two peers over a WebSocket connection.
- The worker either spawns new rooms running game emulators or connects users to existing rooms.
- Raw image and audio streams from the emulator are captured and encoded to a WebRTC-supported streaming format. Next,
  these stream are piped out (dmux) to all users in the room.
- On the other hand, input from players is sent to the worker over WebRTC DataChannel. The game logic on the emulator
  will be updated based on the input stream of all players, for that each stream is multiplexed (mux) into one.
- Game states (saves) are stored in cloud storage, so all distributed workers can keep game states in sync and players
  can continue their games where they left off.
