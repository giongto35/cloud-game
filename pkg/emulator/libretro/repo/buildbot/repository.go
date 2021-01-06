package buildbot

import (
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo/raw"
)

type RepoBuildbot struct {
	raw.Repo
}

func NewBuildbotRepo(address string, compression string) RepoBuildbot {
	return RepoBuildbot{
		Repo: raw.Repo{
			Address:     address,
			Compression: compression,
		},
	}
}

func (r RepoBuildbot) GetCoreUrl(file string, info core.ArchInfo) string {
	var sb strings.Builder
	sb.WriteString(r.Address + "/")
	if info.Vendor != "" {
		sb.WriteString(info.Vendor + "/")
	}
	sb.WriteString(info.Os + "/" + info.Arch + "/latest/" + file + info.LibExt)
	if r.Compression != "" {
		sb.WriteString("." + r.Compression)
	}
	return sb.String()
}
