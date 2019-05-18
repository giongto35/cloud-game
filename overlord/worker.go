package overlord

import (
	"log"

	"github.com/giongto35/cloud-game/cws"
	"github.com/gorilla/websocket"
)

type WorkerClient struct {
	*cws.Client
}

// RouteWorker are all routes server received from worker
func (s *Session) RouteWorker() {
	iceCandidates := [][]byte{}

	workerClient := s.WorkerClient

	workerClient.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})

	workerClient.Receive("icecandidate", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received candidates ", resp.Data)
		iceCandidates = append(iceCandidates, []byte(resp.Data))
		return cws.EmptyPacket
	})

	workerClient.Receive("initwebrtc", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received sdp request from a worker")
		log.Println("Overlord: Sending back sdp to browser")
		s.BrowserClient.Send(resp, nil)

		return cws.EmptyPacket
	})

	//workerClient.Receive("quit", func(resp cws.WSPacket) (req cws.WSPacket) {
	//log.Println("Overlord: Received quit request from a worker")
	//log.Println("Overlord: Sending back sdp to browser")
	//s.GameName = resp.Data
	//s.RoomID = resp.RoomID
	//s.PlayerIndex = resp.PlayerIndex

	//// TODO:
	////room := s.handler.getRoom(s.RoomID)
	////if room.IsPCInRoom(s.peerconnection) {
	////s.handler.detachPeerConn(s.peerconnection)
	////}
	//log.Println("Sending to target host", resp.TargetHostID, " ", resp)
	//resp = s.handler.servers[resp.TargetHostID].SyncSend(
	//resp,
	//)

	//return cws.EmptyPacket
	//})

	// TODO: Add save and load
	//browserClient.Receive("save", func(resp cws.WSPacket) (req cws.WSPacket) {
	//log.Println("Saving game state")
	//req.ID = "save"
	//req.Data = "ok"
	//if s.RoomID != "" {
	//room := s.handler.getRoom(s.RoomID)
	//if room == nil {
	//return
	//}
	//err := room.SaveGame()
	//if err != nil {
	//log.Println("[!] Cannot save game state: ", err)
	//req.Data = "error"
	//}
	//} else {
	//req.Data = "error"
	//}

	//return req
	//})

	//browserClient.Receive("load", func(resp cws.WSPacket) (req cws.WSPacket) {
	//log.Println("Loading game state")
	//req.ID = "load"
	//req.Data = "ok"
	//if s.RoomID != "" {
	//room := s.handler.getRoom(s.RoomID)
	//err := room.LoadGame()
	//if err != nil {
	//log.Println("[!] Cannot load game state: ", err)
	//req.Data = "error"
	//}
	//} else {
	//req.Data = "error"
	//}

	//return req
	//})

	//browserClient.Receive("start", func(resp cws.WSPacket) (req cws.WSPacket) {
	//s.GameName = resp.Data
	//s.RoomID = resp.RoomID
	//s.PlayerIndex = resp.PlayerIndex

	//log.Println("Starting game")
	//// If we are connecting to overlord, request corresponding serverID based on roomID
	//if s.OverlordClient != nil {
	//roomServerID := getServerIDOfRoom(s.OverlordClient, s.RoomID)
	//log.Println("Server of RoomID ", s.RoomID, " is ", roomServerID, " while current server is ", s.ServerID)
	//// If the target serverID is different from current serverID
	//if roomServerID != "" && s.ServerID != roomServerID {
	//// TODO: Re -register
	//// Bridge Connection to the target serverID
	//go s.bridgeConnection(roomServerID, s.GameName, s.RoomID, s.PlayerIndex)
	//return
	//}
	//}

	//// Get Room in local server
	//// TODO: check if roomID is in the current server
	//room := s.handler.getRoom(s.RoomID)
	//log.Println("Got Room from local ", room, " ID: ", s.RoomID)
	//// If room is not running
	//if room == nil {
	//// Create new room
	//room = s.handler.createNewRoom(s.GameName, s.RoomID, s.PlayerIndex)
	//// Wait for done signal from room
	//go func() {
	//<-room.Done
	//s.handler.detachRoom(room.ID)
	//}()
	//}

	//// Attach peerconnection to room. If PC is already in room, don't detach
	//log.Println("Is PC in room", room.IsPCInRoom(s.peerconnection))
	//if !room.IsPCInRoom(s.peerconnection) {
	//s.handler.detachPeerConn(s.peerconnection)
	//room.AddConnectionToRoom(s.peerconnection, s.PlayerIndex)
	//}
	//s.RoomID = room.ID

	//// Register room to overlord if we are connecting to overlord
	//if room != nil && s.OverlordClient != nil {
	//s.OverlordClient.Send(cws.WSPacket{
	//ID:   "registerRoom",
	//Data: s.RoomID,
	//}, nil)
	//}
	//req.ID = "start"
	//req.RoomID = s.RoomID
	//req.SessionID = s.ID

	//return req
	//})
}

// NewWorkerClient returns a client connecting to worker. This connection exchanges information between workers and server
func NewWorkerClient(c *websocket.Conn) *WorkerClient {
	return &WorkerClient{
		Client: cws.NewClient(c),
	}
}
