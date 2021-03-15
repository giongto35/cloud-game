package ice

import (
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
)

type Replacement struct {
	From string
	To   string
}

func NewIceServer(url string) webrtc.IceServer {
	return webrtc.IceServer{
		Url: url,
	}
}

func NewIceServerCredentials(url string, user string, credential string) webrtc.IceServer {
	return webrtc.IceServer{
		Url:        url,
		Username:   user,
		Credential: credential,
	}
}

func ToJson(iceServers []webrtc.IceServer, replacements ...Replacement) string {
	var sb strings.Builder
	sn, n := len(iceServers), len(replacements)
	if sn > 0 {
		sb.Grow(sn * 64)
	}
	sb.WriteString("[")
	for i, ice := range iceServers {
		if i > 0 {
			sb.WriteString(",{")
		} else {
			sb.WriteString("{")
		}
		if n > 0 {
			for _, replacement := range replacements {
				ice.Url = strings.Replace(ice.Url, "{"+replacement.From+"}", replacement.To, -1)
			}
		}
		sb.WriteString("\"urls\":\"" + ice.Url + "\"")
		if ice.Username != "" {
			sb.WriteString(",\"username\":\"" + ice.Username + "\"")
		}
		if ice.Credential != "" {
			sb.WriteString(",\"credential\":\"" + ice.Credential + "\"")
		}
		sb.WriteString("}")
	}
	sb.WriteString("]")
	return sb.String()
}
