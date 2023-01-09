package config

import (
	"os"

	"github.com/kkyr/fig"
)

const EnvPrefix = "CLOUD_GAME"

// LoadConfig loads a configuration file into the given struct.
// The path param specifies a custom path to the configuration file.
// Reads and puts environment variables with the prefix CLOUD_GAME_.
// Params from the config should be in uppercase separated with _.
func LoadConfig(config any, path string) error {
	dirs := []string{path}
	if path == "" {
		dirs = append(dirs, ".", "configs", "../../../configs")
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, home+"/.cr")
		}
	}
	if err := fig.Load(config, fig.Dirs(dirs...), fig.UseEnv(EnvPrefix)); err != nil {
		return err
	}
	return nil
}

func LoadConfigEnv(config any) error {
	return fig.Load(config, fig.IgnoreFile(), fig.UseEnv(EnvPrefix))
}
