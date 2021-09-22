package client

import "github.com/giongto35/cloud-game/v2/pkg/network"

type NetClient interface {
	Close()
	Id() network.Uid
	Printf(format string, args ...interface{})
}

type RegionalClient interface {
	In(region string) bool
}
