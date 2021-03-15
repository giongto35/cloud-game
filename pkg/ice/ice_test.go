package ice

import (
	"testing"

	"github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
)

func TestIce(t *testing.T) {
	tests := []struct {
		input        []webrtc.IceServer
		replacements []Replacement
		output       string
	}{
		{
			input: []webrtc.IceServer{
				NewIceServer("stun:stun.l.google.com:19302"),
				NewIceServer("stun:{server-ip}:3478"),
				NewIceServerCredentials("turn:{server-ip}:3478", "root", "root"),
			},
			replacements: []Replacement{
				{
					From: "server-ip",
					To:   "localhost",
				},
			},
			output: "[" +
				"{\"urls\":\"stun:stun.l.google.com:19302\"}," +
				"{\"urls\":\"stun:localhost:3478\"}," +
				"{\"urls\":\"turn:localhost:3478\",\"username\":\"root\",\"credential\":\"root\"}" +
				"]",
		},
		{
			input: []webrtc.IceServer{
				NewIceServer("stun:stun.l.google.com:19302"),
			},
			output: "[{\"urls\":\"stun:stun.l.google.com:19302\"}]",
		},
		{
			input:        []webrtc.IceServer{},
			replacements: []Replacement{},
			output:       "[]",
		},
	}

	for _, test := range tests {
		result := ToJson(test.input, test.replacements...)

		if result != test.output {
			t.Errorf("Not exactly what is expected")
		}
	}
}

func BenchmarkIces(b *testing.B) {
	benches := []struct {
		name string
		f    func(iceServers []webrtc.IceServer, replacements ...Replacement) string
	}{
		{name: "toJson", f: ToJson},
	}
	servers := []webrtc.IceServer{
		NewIceServer("stun:stun.l.google.com:19302"),
		NewIceServer("stun:{server-ip}:3478"),
		NewIceServerCredentials("turn:{server-ip}:3478", "root", "root"),
	}
	replacements := []Replacement{
		{From: "server-ip", To: "localhost"},
	}

	for _, bench := range benches {
		b.Run(bench.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				bench.f(servers, replacements...)
			}
		})
	}
}
