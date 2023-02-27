package com

import "github.com/rs/xid"

type Uid struct {
	xid.ID
}

var NilUid = Uid{xid.NilID()}

func NewUid() Uid { return Uid{xid.New()} }

func (u Uid) Generate() Uid { return NewUid() }
func (u Uid) GetId() Uid    { return u }
func (u Uid) IsEmpty() bool { return u.IsNil() }
func (u Uid) Short() string { return u.String()[:3] + "." + u.String()[len(u.String())-3:] }
