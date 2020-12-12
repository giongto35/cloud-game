package config

import (
	"os"

	"github.com/kkyr/fig"
)

type Loader interface {
	LoadConfig(config interface{}, path string) interface{}
}

// LoadConfig loads a configuration file into the given struct.
// The path param specifies a custom path to the configuration file.
func LoadConfig(config interface{}, path string) interface{} {
	var err error

	if path == "" {
		dirs := []string{".", "configs", "../../../configs"}
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, home+"/.cr")
		}
		err = fig.Load(config, fig.Dirs(dirs...))
	} else {
		err = fig.Load(config, fig.Dirs(path))
	}
	if err != nil {
		panic(err)
	}
	return config
}
