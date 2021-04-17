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
	return func(resp cws.WSPacket) cws.WSPacket {
		// initWebrtc now only sends signal to worker, asks it to createOffer
		bc.Printf("Received init_webrtc request -> relay to worker: %s", bc.Worker)
		// relay request to target worker
		// worker creates a PeerConnection, and createOffer
		// send SDP back to browser
		return o.RelayPacket(bc, resp, func(w *WorkerClient, p cws.WSPacket) cws.WSPacket {
			defer bc.Println("Received SDP from worker -> sending back to browser")
			return w.SyncSend(p)
		})
	}
}

func (bc *BrowserClient) handleAnswer(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		// contains SDP of browser createAnswer
		bc.Println("Received browser answered SDP -> relay to worker")
		return o.RelayPacket(bc, resp, func(w *WorkerClient, p cws.WSPacket) cws.WSPacket {
			w.SendPacket(p)
			return cws.EmptyPacket
		})
	}
}

func (bc *BrowserClient) handleIceCandidate(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		// contains ICE candidate of browser
		bc.Println("Received IceCandidate from browser -> relay to worker")
		return o.RelayPacket(bc, resp, func(w *WorkerClient, p cws.WSPacket) cws.WSPacket {
			w.SendPacket(p)
			return cws.EmptyPacket
		})
	}
}

func (bc *BrowserClient) handleGameStart(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received start request from a browser -> relay to worker")
		// TODO: Async
		return o.RelayPacket(bc, resp, func(w *WorkerClient, p cws.WSPacket) cws.WSPacket {
			// +injects game data into the original game request
			gameStartCall, err := newGameStartCall(p.RoomID, p.Data, o.library)
			if err != nil {
				return cws.EmptyPacket
			}
			if packet, err := gameStartCall.To(); err != nil {
				return cws.EmptyPacket
			} else {
				p.Data = packet
			}
			workerResp := w.SyncSend(p)

			// Response from worker contains initialized roomID. Set roomID to the session
			bc.RoomID = workerResp.RoomID
			bc.Println("Received room response from browser: ", workerResp.RoomID)
			return workerResp
		})
	}
}

func (bc *BrowserClient) handleGameQuit(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received quit request from a browser -> relay to worker")
		// TODO: Async
		return o.RelayPacket(bc, resp, func(w *WorkerClient, p cws.WSPacket) cws.WSPacket {
			w.SyncSend(p)
			return cws.EmptyPacket
		})
	}
}

func (bc *BrowserClient) handleGameSave(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received save request from a browser -> relay to worker")
		// TODO: Async
		return o.RelayPacket(bc, resp, func(w *WorkerClient, p cws.WSPacket) cws.WSPacket {
			p.RoomID = bc.RoomID
			return w.SyncSend(p)
		})
	}
}

func (bc *BrowserClient) handleGameLoad(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received load request from a browser -> relay to worker")
		// TODO: Async
		return o.RelayPacket(bc, resp, func(w *WorkerClient, p cws.WSPacket) cws.WSPacket {
			p.RoomID = bc.RoomID
			return w.SyncSend(p)
		})
	}
}

func (bc *BrowserClient) handleGamePlayerSelect(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received update player index request from a browser -> relay to worker")
		// TODO: Async
		return o.RelayPacket(bc, resp, func(w *WorkerClient, p cws.WSPacket) cws.WSPacket {
			p.RoomID = bc.RoomID
			return w.SyncSend(p)
		})
	}
}

func (bc *BrowserClient) handleGameMultitap(o *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		bc.Println("Received multitap request from a browser -> relay to worker")
		// TODO: Async
		return o.RelayPacket(bc, resp, func(w *WorkerClient, p cws.WSPacket) cws.WSPacket {
			p.RoomID = bc.RoomID
			return w.SyncSend(p)
		})
	}
}

// newGameStartCall gathers data for a new game start call of the worker
func newGameStartCall(roomId string, data string, library games.GameLibrary) (api.GameStartCall, error) {
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

	return api.GameStartCall{
		Name: gameInfo.Name,
		Path: gameInfo.Path,
		Type: gameInfo.Type,
	}, nil
}
