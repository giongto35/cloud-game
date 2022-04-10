package coordinator

import "github.com/giongto35/cloud-game/v2/pkg/api"

type ServerInfo interface {
	getServerList() []api.Server
}
