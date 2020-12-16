package core

import "strings"

type BuildbotRepo struct {
	address     string
	compression string
}

func (r *BuildbotRepo) GetLink(file string, info ArchInfo) string {
	var sb strings.Builder
	sb.WriteString(r.address + "/")
	if info.vendor != "" {
		sb.WriteString(info.vendor + "/")
	}
	sb.WriteString(info.os + "/" + info.arch + "/latest/" + file + info.Lib)
	if r.compression != "" {
		sb.WriteString("." + r.compression)
	}
	return sb.String()
}
