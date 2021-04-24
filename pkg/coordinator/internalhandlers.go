package coordinator

import (
	"log"

	api2 "github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
)

func (wc *WorkerClient) handleHeartbeat() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		return resp
	}
}

func (wc *WorkerClient) handleRegisterRoom2(rt *Hub) cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Coordinator: Received registerRoom room %s from worker %s", resp.Data, wc.Id)
		rt.rooms[resp.Data] = wc
		log.Printf("Coordinator: Current room list is: %+v", rt.rooms)
		return api.RegisterRoomPacket(api.NoData)
	}
}

// handleGetRoom returns the server ID based on requested roomID.
func (wc *WorkerClient) handleGetRoom2(rt *Hub) cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Coordinator: Received a get room request")
		log.Println("Result: ", rt.rooms[resp.Data])
		return api.GetRoomPacket(string(rt.rooms[resp.Data].Id))
	}
}

func (wc *WorkerClient) handleCloseRoom2(rt *Hub) cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Coordinator: Received closeRoom room %s from worker %s", resp.Data, wc.Id)
		delete(rt.rooms, resp.Data)
		log.Printf("Coordinator: Current room list is: %+v", rt.rooms)
		return api.CloseRoomPacket(api.NoData)
	}
}

// handleIceCandidate passes an ICE candidate (WebRTC) to the browser.
func (wc *WorkerClient) handleIceCandidate2(rt *Hub) cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		wc.Println("relay IceCandidate to useragent")
		bc, ok := rt.users[resp.SessionID]
		if ok {
			// Remove SessionID while sending back to browser
			resp.SessionID = ""
			bc.SendAndForget(api2.P_webrtc_ice_candidate, resp.Data)
		} else {
			wc.Println("error: unknown SessionID:", resp.SessionID)
		}
		return cws.EmptyPacket
	}
}
