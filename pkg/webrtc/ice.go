package webrtc

import "strings"

type IceServer struct {
	Url        string
	Username   string
	Credential string
}

type Replacement struct {
	From string
	To   string
}

func NewIceServer(url string) IceServer {
	return IceServer{
		Url: url,
	}
}

func NewIceServerCredentials(url string, user string, credential string) IceServer {
	return IceServer{
		Url:        url,
		Username:   user,
		Credential: credential,
	}
}

func ToJson(iceServers []IceServer, replacements ...Replacement) string {
	var sb strings.Builder

	n := len(replacements)
	serversN := len(iceServers)
	sb.WriteString("[")
	delim := ","
	for i, ice := range iceServers {
		sb.WriteString("{")

		var params []string
		url := ice.Url
		if n > 0 {
			for _, replacement := range replacements {
				url = strings.Replace(url, "{"+replacement.From+"}", replacement.To, -1)
			}
		}
		params = append(params, "\"urls\":\""+url+"\"")
		if ice.Username != "" {
			params = append(params, "\"username\":\""+ice.Username+"\"")
		}
		if ice.Credential != "" {
			params = append(params, "\"credential\":\""+ice.Credential+"\"")
		}
		sb.WriteString(strings.Join(params, ","))

		if i == serversN-1 {
			delim = ""
		}
		sb.WriteString("}" + delim)
	}
	sb.WriteString("]")

	return sb.String()
}
