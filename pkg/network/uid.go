package network

import "github.com/rs/xid"

type Uid string

const EmptyUid Uid = ""

func NewUid() Uid { return Uid(xid.New().String()) }

func ValidUid(u Uid) bool {
	_, err := xid.FromString(string(u))
	return err == nil
}

func (u Uid) String() string { return string(u) }

func (u Uid) Short() string {
	return string(u)[:3] + "." + string(u)[len(u)-3:]
}
