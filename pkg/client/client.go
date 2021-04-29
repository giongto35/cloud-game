package client

import "github.com/giongto35/cloud-game/v2/pkg/network"

type NetClient interface {
	Id() network.Uid
	InRegion(region string) bool
	Printf(format string, args ...interface{})
}
