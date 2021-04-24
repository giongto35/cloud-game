package network

import (
	"encoding/base64"
	"strings"

	"github.com/gofrs/uuid"
)

type Uid string

const EmptyUid Uid = ""

func NewUid() Uid {
	uid, _ := uuid.NewV4()
	return Uid(base64.RawURLEncoding.EncodeToString(uid.Bytes()))
}

// Short returns Github-like short ID version (i.e. 81cef7d).
func (u Uid) Short() string { return strings.ToLower(string(u[0:6])) }
