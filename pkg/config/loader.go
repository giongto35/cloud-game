package config

import (
	"os"
	"path/filepath"
	"strings"

	_ "embed"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"
)

const EnvPrefix = "CLOUD_GAME_"

var (
	//go:embed config.yaml
	conf Bytes
)

type Kv = map[string]any
type Bytes []byte

func (b *Bytes) ReadBytes() ([]byte, error) { return *b, nil }
func (b *Bytes) Read() (Kv, error)          { return nil, nil }

type File string

func (f *File) ReadBytes() ([]byte, error) { return os.ReadFile(string(*f)) }
func (f *File) Read() (Kv, error)          { return nil, nil }

type YAML struct{}

func (p *YAML) Marshal(Kv) ([]byte, error) { return nil, nil }
func (p *YAML) Unmarshal(b []byte) (Kv, error) {
	var out Kv
	klw := keysToLower(b)
	if err := yaml.Unmarshal(klw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// keysToLower iterates YAML bytes and tries to lower the keys.
// Used for merging with environment vars which are lowered as well.
func keysToLower(in []byte) []byte {
	l, r, ignore := 0, 0, false
	for i, b := range in {
		switch b {
		case '#': // skip comments
			ignore = true
		case ':': // lower left chunk before the next : symbol
			if ignore {
				continue
			}
			r = i
			ignore = true
			for j := l; j <= r; j++ {
				c := in[j]
				// we skip the line with the first explicit " string symbol
				if c == '"' {
					break
				}
				if 'A' <= c && c <= 'Z' {
					in[j] += 'a' - 'A'
				}
			}
		case '\n':
			l = i
			ignore = false
		}
	}
	return in
}

type Env string

func (e *Env) ReadBytes() ([]byte, error) { return nil, nil }
func (e *Env) Read() (Kv, error) {
	var keys []string
	for _, k := range os.Environ() {
		if strings.HasPrefix(k, string(*e)) {
			keys = append(keys, k)
		}
	}
	mp := make(Kv)
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
func LoadConfig(config any, path string) (loaded []string, err error) {
	dirs := []string{".", "configs", "../../../configs"}
	if path != "" {
		dirs = append([]string{path}, dirs...)
	}

	homeDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		homeDir = home + "/.cr"
		dirs = append(dirs, homeDir)
	}

	if err := k.Load(&conf, &YAML{}); err != nil {
		return nil, err
	}
	conf = nil
	loaded = append(loaded, "default")

	for _, dir := range dirs {
		path := filepath.Join(filepath.Clean(dir), "config.yaml")
		f := File(path)
		if _, err := os.Stat(string(f)); !os.IsNotExist(err) {
			if err := k.Load(&f, &YAML{}); err != nil {
				return loaded, err
			}
			loaded = append(loaded, path)
		}
	}

	env := Env(EnvPrefix)
	if err := k.Load(&env, nil); err != nil {
		return loaded, err
	}

	if err := k.Unmarshal("", config); err != nil {
		return loaded, err
	}

	return loaded, nil
}
