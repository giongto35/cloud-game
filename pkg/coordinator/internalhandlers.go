package coordinator

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
)

func (wc *WorkerClient) handleConfigRequest() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		// try to load worker config
		conf := worker.NewConfig()
		return api.ConfigRequestPacket(conf.Serialize())
	}
}

func (wc *WorkerClient) handleHeartbeat() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		return resp
	}
}

// handleRegisterRoom event from a worker, when worker created a new room.
// RoomID is global so it is managed by coordinator.
func (wc *WorkerClient) handleRegisterRoom(s *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Coordinator: Received registerRoom room %s from worker %s", resp.Data, wc.WorkerID)
		s.roomToWorker[resp.Data] = wc.WorkerID
		log.Printf("Coordinator: Current room list is: %+v", s.roomToWorker)
		return api.RegisterRoomPacket(api.NoData)
	}
}

// handleGetRoom returns the server ID based on requested roomID.
func (wc *WorkerClient) handleGetRoom(s *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Coordinator: Received a get room request")
		log.Println("Result: ", s.roomToWorker[resp.Data])
		return api.GetRoomPacket(s.roomToWorker[resp.Data].String())
	}
}

// handleCloseRoom event from a worker, when worker close a room.
func (wc *WorkerClient) handleCloseRoom(s *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Coordinator: Received closeRoom room %s from worker %s", resp.Data, wc.WorkerID)
		delete(s.roomToWorker, resp.Data)
		log.Printf("Coordinator: Current room list is: %+v", s.roomToWorker)
		return api.CloseRoomPacket(api.NoData)
	}
}

// handleIceCandidate passes an ICE candidate (WebRTC) to the browser.
func (wc *WorkerClient) handleIceCandidate(s *Server) cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		wc.Println("relay IceCandidate to useragent")
		bc, ok := s.browserClients[resp.SessionID]
		if ok {
			// Remove SessionID while sending back to browser
			resp.SessionID = [16]byte{}
			bc.Send(resp, nil)
		} else {
			wc.Println("error: unknown SessionID:", resp.SessionID)
		}
		return cws.EmptyPacket
	}
}
