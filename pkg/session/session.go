package session

import (
	"math/rand"
	"strconv"
	"strings"
)

const separator = "___"

// GetGameNameFromRoomID parse roomID to get roomID and gameName.
func GetGameNameFromRoomID(roomID string) string {
	parts := strings.Split(roomID, separator)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// GenerateRoomID generate a unique room ID containing 16 digits.
func GenerateRoomID(gameName string) string {
	// RoomID contains random number + gameName
	// Next time when we only get roomID, we can launch game based on gameName
	roomID := strconv.FormatInt(rand.Int63(), 16) + separator + gameName
	return roomID
}
