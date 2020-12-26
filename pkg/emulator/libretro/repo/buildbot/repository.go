package buildbot

import (
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo"
)

type Repo struct {
	address     string
	compression repo.CompressionType
}

func NewBuildbotRepo(address string, compression string) Repo {
	return Repo{address: address, compression: (repo.CompressionType)(compression)}
}

func (r Repo) GetCoreData(file string, info core.ArchInfo) repo.Data {
	var sb strings.Builder
	sb.WriteString(r.address + "/")
	if info.Vendor != "" {
		sb.WriteString(info.Vendor + "/")
	}
	sb.WriteString(info.Os + "/" + info.Arch + "/latest/" + file + info.LibExt)
	if r.compression != "" {
		sb.WriteString("." + r.compression.GetExt())
	}
	return repo.Data{Url: sb.String(), Compression: r.compression}
}
