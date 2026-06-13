package config

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"
)

const EnvPrefix = "CLOUD_GAME_"

var (
	//go:embed config.yaml
	conf embed.FS
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
	decoder := yaml.NewDecoder(bytes.NewReader(klw))
	if err := decoder.Decode(&out); err != nil {
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
		if parts == nil {
			continue
		}
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
		if len(parts) > 1 {
			mp[replaceArrayNotation(key)] = parts[1]
		}
	}
	return arrayify(maps.Unflatten(mp, ".")).(Kv), nil
}

// replaceArrayNotation converts array index notation like frames[0] into
// the delimiter-separated form frames.0. This allows maps.Unflatten to
// produce maps with numeric string keys, which are then converted to slices
// by arrayify.
func replaceArrayNotation(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '[' && i+1 < len(s) && unicode.IsDigit(rune(s[i+1])) {
			b.WriteByte('.')
		} else if c != ']' {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// arrayify recursively walks a nested map and converts any
// map[string]any whose keys are all numeric strings into a []any.
func arrayify(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, sub := range val {
			val[k] = arrayify(sub)
		}
		if isNumericMap(val) {
			return mapToSlice(val)
		}
		return val
	default:
		return v
	}
}

func isNumericMap(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}
	for k := range m {
		if _, err := strconv.Atoi(k); err != nil {
			return false
		}
	}
	return true
}

func mapToSlice(m map[string]any) []any {
	max := 0
	for k := range m {
		n, _ := strconv.Atoi(k)
		if n > max {
			max = n
		}
	}
	out := make([]any, max+1)
	for k, v := range m {
		n, _ := strconv.Atoi(k)
		out[n] = v
	}
	return out
}

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

	k := koanf.New("_") // move to global scope if configs become dynamic
	defer k.Delete("")
	data, err := conf.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}
	conf := Bytes(data)
	if err := k.Load(&conf, &YAML{}); err != nil {
		return nil, err
	}
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
