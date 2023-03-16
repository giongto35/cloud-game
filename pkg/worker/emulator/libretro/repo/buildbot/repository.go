package buildbot

import (
	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/libretro"
	"strings"

	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/libretro/repo/raw"
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

func (r RepoBuildbot) GetCoreUrl(file string, info libretro.ArchInfo) string {
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
