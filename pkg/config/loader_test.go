package config

import (
	"os"
	"reflect"
	"testing"
)

func TestConfigEnv(t *testing.T) {
	var out WorkerConfig

	_ = os.Setenv("CLOUD_GAME_ENCODER_AUDIO_FRAME", "33")
	defer func() { _ = os.Unsetenv("CLOUD_GAME_ENCODER_AUDIO_FRAME") }()

	_ = os.Setenv("CLOUD_GAME_EMULATOR_LIBRETRO_CORES_LIST_PCSX_OPTIONS__PCSX_REARMED_DRC", "x")
	defer func() {
		_ = os.Unsetenv("CLOUD_GAME_EMULATOR_LIBRETRO_CORES_LIST_PCSX_OPTIONS__PCSX_REARMED_DRC")
	}()

	err := LoadConfig(&out, "../../configs")
	if err != nil {
		t.Fatal(err)
	}

	if out.Encoder.Audio.Frame != 33 {
		t.Errorf("%v is not 33", out.Encoder.Audio.Frame)
	}

	v := out.Emulator.Libretro.Cores.List["pcsx"].Options["pcsx_rearmed_drc"]
	if v != "x" {
		t.Errorf("%v is not x", v)
	}
}

func Test_keysToLower(t *testing.T) {
	type args struct {
		in []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{name: "empty", args: args{in: []byte{}}, want: []byte{}},
		{name: "case", args: args{
			in: []byte("KEY:1\n#Comment with:\n      	KeY123_NamE: 1\n\n\n\nAAA:123\n  \"KeyKey\":2\n"),
		},
			want: []byte("key:1\n#Comment with:\n      	key123_name: 1\n\n\n\naaa:123\n  \"KeyKey\":2\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keysToLower(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("keysToLower() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}
