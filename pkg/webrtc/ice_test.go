package webrtc

import "testing"

func TestIce(t *testing.T) {
	tests := []struct {
		input        []IceServer
		replacements []Replacement
		output       string
	}{
		{
			input: []IceServer{
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
	}

	for _, test := range tests {
		result := ToJson(test.input, test.replacements...)

		if result != test.output {
			t.Errorf("Not exactly what is expected")
		}
	}
}
