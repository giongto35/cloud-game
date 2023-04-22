package config

type Webrtc struct {
	DisableDefaultInterceptors bool
	DtlsRole                   byte
	IceServers                 []IceServer
	IcePorts                   struct {
		Min uint16
		Max uint16
	}
	IceIpMap   string
	IceLite    bool
	SinglePort int
	LogLevel   int
}

type IceServer struct {
	Urls       string `json:"urls,omitempty"`
	Username   string `json:"username,omitempty"`
	Credential string `json:"credential,omitempty"`
}

func (w *Webrtc) HasDtlsRole() bool   { return w.DtlsRole > 0 }
func (w *Webrtc) HasPortRange() bool  { return w.IcePorts.Min > 0 && w.IcePorts.Max > 0 }
func (w *Webrtc) HasSinglePort() bool { return w.SinglePort > 0 }
func (w *Webrtc) HasIceIpMap() bool   { return w.IceIpMap != "" }
