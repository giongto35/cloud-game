package util

import (
	"net"
	"strings"

	"github.com/gorilla/websocket"
)

// GetRemoteAddress returns public address of websocket connection
func GetRemoteAddress(conn *websocket.Conn) string {
	var remoteAddr string
	// log.Println("Address :", conn.RemoteAddr().String())
	if parts := strings.Split(conn.RemoteAddr().String(), ":"); len(parts) == 2 {
		remoteAddr = parts[0]
	}
	if remoteAddr == "" {
		return "localhost"
	}

	return remoteAddr
}

// IsPublicIP checks if address is public address
func IsPublicIP(address string) bool {
	ip := net.ParseIP(address)
	if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}

// GetHostPublicIP to get the public ip address. Only work if not behind NAT
func GetHostPublicIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")
	return localAddr[0:idx]
}
