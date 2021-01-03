package config

import (
	"os"

	"github.com/kkyr/fig"
)

// LoadConfig loads a configuration file into the given struct.
// The path param specifies a custom path to the configuration file.
// Reads and puts environment variables with the prefix CLOUD_GAME_.
// Params from the config should be in uppercase separated with _.
func LoadConfig(config interface{}, path string) error {
	envPrefix := "CLOUD_GAME"
	dirs := []string{path}
	if path == "" {
		dirs = append(dirs, ".", "configs", "../../../configs")
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, home+"/.cr")
		}
	}
	if err := fig.Load(config, fig.Dirs(dirs...), fig.UseEnv(envPrefix)); err != nil {
		return err
	}
	return nil
}
