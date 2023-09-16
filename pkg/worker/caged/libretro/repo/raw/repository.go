package raw

import "github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/repo/arch"

type Repo struct {
	Address     string
	Compression string
}

// NewRawRepo defines a simple zip file containing
// all the cores that will be extracted as is.
func NewRawRepo(address string) Repo { return Repo{Address: address, Compression: "zip"} }

func (r Repo) GetCoreUrl(_ string, _ arch.Info) string { return r.Address }
