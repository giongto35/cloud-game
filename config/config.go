package config

import (
	"flag"
	"time"
)

const defaultoverlord = "ws://localhost:9000/wso"

var IsDebug = flag.Bool("debug", false, "Is game running in debug mode?")
var OverlordHost = flag.String("overlordhost", defaultoverlord, "Specify the path for overlord. If the flag is `overlord`, the server will be run as overlord")
var Port = flag.String("port", "8000", "Port of the game")
var IsMonitor = flag.Bool("monitor", false, "Turn on monitor")
var FrontendSTUNTURN = flag.String("turn", `[{"urls":"stun:stun-turn.webgame2d.com:3478"},{"urls":"turn:stun-turn.webgame2d.com:3478","username":"root","credential":"root"}]`, "Frontend STUN TURN servers")

var Width = 256
var Height = 240
var WSWait = 20 * time.Second
