package manager

import "strings"

type ArchInfo struct {
	Arch   string
	Ext    string
	Os     string
	Vendor string
}

type Data struct {
	Url         string
	Compression string
}

type Repository interface {
	CoreUrl(file string, info ArchInfo) (url string)
}

// Repo defines a simple zip file containing all the cores that will be extracted as is.
type Repo struct {
	Address     string
	Compression string
}

func (r Repo) CoreUrl(_ string, _ ArchInfo) string { return r.Address }

type Buildbot struct{ Repo }

func (r Buildbot) CoreUrl(file string, info ArchInfo) string {
	var sb strings.Builder
	sb.WriteString(r.Address + "/")
	if info.Vendor != "" {
		sb.WriteString(info.Vendor + "/")
	}
	sb.WriteString(info.Os + "/" + info.Arch + "/latest/" + file + info.Ext)
	if r.Compression != "" {
		sb.WriteString("." + r.Compression)
	}
	return sb.String()
}

type Github struct{ Buildbot }

func (r Github) CoreUrl(file string, info ArchInfo) string {
	return r.Buildbot.CoreUrl(file, info) + "?raw=true"
}

func NewRepo(kind string, url string, compression string, defaultRepo string) Repository {
	var repository Repository
	switch kind {
	case "buildbot":
		repository = Buildbot{Repo{Address: url, Compression: compression}}
	case "github":
		repository = Github{Buildbot{Repo{Address: url, Compression: compression}}}
	case "raw":
		repository = Repo{Address: url, Compression: "zip"}
	default:
		if defaultRepo != "" {
			repository = NewRepo(defaultRepo, url, compression, "")
		}
	}
	return repository
}
