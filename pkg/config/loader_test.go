package config

import (
	"os"
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
