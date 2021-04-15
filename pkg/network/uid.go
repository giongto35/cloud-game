package network

import (
	"encoding/hex"
	"github.com/gofrs/uuid"
)

type Uid uuid.UUID

func NewUid() Uid {
	uid, _ := uuid.NewV4()
	return Uid(uid)
}

func (u Uid) String() string { return uuid.UUID(u).String() }

func (u Uid) Short() string {
	buf := make([]byte, 8)
	hex.Encode(buf[0:8], u[0:4])
	return string(buf)
}
