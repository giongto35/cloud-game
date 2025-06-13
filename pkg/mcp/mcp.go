package mcp

import (
	"encoding/binary"
	mcpg "github.com/mark3labs/mcp-go"
)

type Action = mcpg.Action

type Message = mcpg.Message

func Parse(data []byte) (Message, error) {
	return mcpg.Parse(data)
}

func actionBytes(a Action) []byte {
	buf := make([]byte, 7)
	binary.BigEndian.PutUint32(buf, keyCode(a.Key))
	if a.Press {
		buf[4] = 1
	}
	return buf
}

func ToBytes(m Message) [][]byte {
	b := make([][]byte, len(m.Actions))
	for i, a := range m.Actions {
		b[i] = actionBytes(a)
	}
	return b
}
