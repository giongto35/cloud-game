package mcp

import (
	"encoding/binary"
	"encoding/json"
)

type Action struct {
	Key   string `json:"key"`
	Press bool   `json:"press"`
}

type Message struct {
	Actions []Action `json:"actions"`
}

func Parse(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	return m, err
}

func actionBytes(a Action) []byte {
	buf := make([]byte, 7)
	binary.BigEndian.PutUint32(buf, keyCode(a.Key))
	if a.Press {
		buf[4] = 1
	}
	// last two bytes are modifier flags, not used
	return buf
}

func ToBytes(m Message) [][]byte {
	b := make([][]byte, len(m.Actions))
	for i, a := range m.Actions {
		b[i] = actionBytes(a)
	}
	return b
}
