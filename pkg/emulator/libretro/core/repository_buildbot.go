package core

import "strings"

type BuildbotRepo struct {
	address     string
	compression CompressionType
}

func New(address string) *BuildbotRepo {
	return &BuildbotRepo{
		address: address,
	}
}

func (r *BuildbotRepo) WithCompression(compression string) *BuildbotRepo {
	r.compression = (CompressionType)(compression)
	return r
}

func (r *BuildbotRepo) GetCoreData(file string, info ArchInfo) Data {
	var sb strings.Builder
	sb.WriteString(r.address + "/")
	if info.vendor != "" {
		sb.WriteString(info.vendor + "/")
	}
	sb.WriteString(info.os + "/" + info.arch + "/latest/" + file + info.Lib)
	if r.compression != "" {
		sb.WriteString("." + r.compression.GetExt())
	}
	return Data{Url: sb.String(), Compression: r.compression}
}
