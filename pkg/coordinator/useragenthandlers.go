package coordinator

import (
	"errors"
	"fmt"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/session"
)

func (bc *BrowserClient) handleHeartbeat() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket { return resp }
}

func (bc *BrowserClient) handleInitWebrtc(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		// initWebrtc now only sends signal to worker, asks it to createOffer
		bc.Printf("Received init_webrtc request -> relay to worker: %s", bc.WorkerID)
		// relay request to target worker
		// worker creates a PeerConnection, and createOffer
		// send SDP back to browser
		resp.SessionID = bc.SessionID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		sdp := wc.SyncSend(resp)
		bc.Println("Received SDP from worker -> sending back to browser")
		return sdp
	}
}

func (bc *BrowserClient) handleAnswer(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		// contains SDP of browser createAnswer
		// forward to worker
		bc.Println("Received browser answered SDP -> relay to worker")
		resp.SessionID = bc.SessionID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		wc.Send(resp, nil)
		// no need to response
		return cws.EmptyPacket
	}
}

func (bc *BrowserClient) handleIceCandidate(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		// contains ICE candidate of browser
		// forward to worker
		bc.Println("Received IceCandidate from browser -> relay to worker")
		resp.SessionID = bc.SessionID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		wc.Send(resp, nil)
		return cws.EmptyPacket
	}
}

func (bc *BrowserClient) handleGameStart(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received start request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = bc.SessionID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}

		// +injects game data into the original game request
		gameStartCall, err := newGameStartCall(resp.RoomID, resp.Data, o.library, o.cfg.Recording.Enabled)
		if err != nil {
			return cws.EmptyPacket
		}
		if packet, err := gameStartCall.To(); err != nil {
			return cws.EmptyPacket
		} else {
			resp.Data = packet
		}
		workerResp := wc.SyncSend(resp)
		// Response from worker contains initialized roomID. Set roomID to the session
		bc.RoomID = workerResp.RoomID
		bc.Println("Received room response from browser: ", workerResp.RoomID)

		if o.cfg.Recording.Enabled && gameStartCall.Record {
			bc.Send(cws.WSPacket{
				ID:          api.GameRecording,
				Data:        "ok",
				RoomID:      workerResp.RoomID,
				PlayerIndex: workerResp.PlayerIndex,
				PacketID:    workerResp.PacketID,
				SessionID:   workerResp.SessionID,
			}, func(response cws.WSPacket) {

			})
		}

		return workerResp
	}
}

func (bc *BrowserClient) handleGameQuit(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received quit request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = bc.SessionID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		// Send but, waiting
		wc.SyncSend(resp)

		return cws.EmptyPacket
	}
}

func (bc *BrowserClient) handleGameSave(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received save request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = bc.SessionID
		resp.RoomID = bc.RoomID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(resp)

		return resp
	}
}

func (bc *BrowserClient) handleGameLoad(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received load request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = bc.SessionID
		resp.RoomID = bc.RoomID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(resp)

		return resp
	}
}

func (bc *BrowserClient) handleGamePlayerSelect(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received update player index request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = bc.SessionID
		resp.RoomID = bc.RoomID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(resp)

		return resp
	}
}

func (bc *BrowserClient) handleGameMultitap(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received multitap request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = bc.SessionID
		resp.RoomID = bc.RoomID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(resp)

		return resp
	}
}

func (bc *BrowserClient) handleGameRecording(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received recording request from a browser -> relay to worker")

		if !o.cfg.Recording.Enabled {
			bc.Printf("Recording should be disabled!")
			return cws.EmptyPacket
		}

		request := api.GameRecordingRequest{}
		if err := request.From(resp.Data); err != nil {
			return cws.EmptyPacket
		}

		bc.Printf("Session: %v, room: %v, rec: %v user: %v", bc.SessionID, bc.RoomID, request.Active, request.User)

		if bc.RoomID == "" {
			bc.Printf("Recording in the empty room is not allowed!")
			return cws.EmptyPacket
		}

		resp.SessionID = bc.SessionID
		resp.RoomID = bc.RoomID
		wc, ok := o.workerClients[bc.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(resp)

		return resp
	}
}

// newGameStartCall gathers data for a new game start call of the worker
func newGameStartCall(roomId string, data string, library games.GameLibrary, recording bool) (api.GameStartCall, error) {
	request := api.GameStartRequest{}
	if err := request.From(data); err != nil {
		return api.GameStartCall{}, errors.New("invalid request")
	}

	// the name of the game either in the `room id` field or
	// it's in the initial request
	game := request.GameName
	if roomId != "" {
		// ! should be moved into coordinator
		name := session.GetGameNameFromRoomID(roomId)
		if name == "" {
			return api.GameStartCall{}, errors.New("couldn't decode game name from the room id")
		}
		game = name
	}

	gameInfo := library.FindGameByName(game)
	if gameInfo.Path == "" {
		return api.GameStartCall{}, fmt.Errorf("couldn't find game info for the game %v", game)
	}

	call := api.GameStartCall{
		Name: gameInfo.Name,
		Base: gameInfo.Base,
		Path: gameInfo.Path,
		Type: gameInfo.Type,
	}
	if recording {
		call.Record = request.Record
		call.RecordUser = request.RecordUser
	}
	return call, nil
}
