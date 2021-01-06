package webrtc

import (
	"github.com/pion/interceptor"
	. "github.com/pion/webrtc/v3"
)

func NewInterceptedPeerConnection(conf Configuration, interceptors []interceptor.Interceptor) (*PeerConnection, error) {
	m := &MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}

	i := &interceptor.Registry{}
	if err := RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}
	for _, itc := range interceptors {
		i.Add(itc)
	}

	api := NewAPI(WithMediaEngine(m), WithInterceptorRegistry(i))
	return api.NewPeerConnection(conf)
}
