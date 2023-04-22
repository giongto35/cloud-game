package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"
)

const EnvPrefix = "CLOUD_GAME_"

type File string

func (f *File) ReadBytes() ([]byte, error)            { return os.ReadFile(string(*f)) }
func (f *File) Read() (map[string]interface{}, error) { return nil, nil }

type YAML struct{}

func (p *YAML) Marshal(map[string]interface{}) ([]byte, error) { return nil, nil }
func (p *YAML) Unmarshal(b []byte) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type Env string

func (e *Env) ReadBytes() ([]byte, error) { return nil, nil }
func (e *Env) Read() (map[string]interface{}, error) {
	var keys []string
	for _, k := range os.Environ() {
		if strings.HasPrefix(k, string(*e)) {
			keys = append(keys, k)
		}
	}
	mp := make(map[string]interface{})
	for _, k := range keys {
		parts := strings.SplitN(k, "=", 2)
		n := strings.ToLower(strings.TrimPrefix(parts[0], string(*e)))
		if n == "" {
			continue
		}
		// convert VAR_VAR to VAR.VAR or if we need to preserve _
		// i.e. VAR_VAR__KEY_HAS_SLASHES to VAR.VAR.KEY_HAS_SLASHES
		// with the result: VAR: { VAR: { KEY_HAS_SLASHES: '' } } }
		x := strings.Index(n, "__")
		var key string
		if x == -1 {
			key = strings.Replace(n, "_", ".", -1)
		} else {
			key = strings.Replace(n[:x+1], "_", ".", -1) + n[x+2:]
		}
		mp[key] = parts[1]
	}
	return maps.Unflatten(mp, "."), nil
}

var k = koanf.New("_")

// LoadConfig loads a configuration file into the given struct.
// The path param specifies a custom path to the configuration file.
// Reads and puts environment variables with the prefix CLOUD_GAME_.
func LoadConfig(config any, path string) error {
	dirs := []string{path}
	if path == "" {
		dirs = append(dirs, ".", "configs", "../../../configs")
	}

	homeDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		homeDir = home + "/.cr"
		dirs = append(dirs, homeDir)
	}

	for _, dir := range dirs {
		f := File(filepath.Join(filepath.Clean(dir), "config.yaml"))
		if _, err := os.Stat(string(f)); !os.IsNotExist(err) {
			if err := k.Load(&f, &YAML{}); err != nil {
				return err
			}
		}
	}

	env := Env(EnvPrefix)
	if err := k.Load(&env, nil); err != nil {
		return err
	}

	if err := k.Unmarshal("", config); err != nil {
		return err
	}

	return nil
}
