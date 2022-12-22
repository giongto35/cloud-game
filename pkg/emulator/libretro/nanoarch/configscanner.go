package nanoarch

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

import "C"

type ConfigProperties map[string]*C.char

func ScanConfigFile(filename string) (ConfigProperties, error) {
	config := ConfigProperties{}

	if len(filename) == 0 {
		return config, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return config, fmt.Errorf("couldn't find the %v config file", filename)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				config[key] = C.CString(value)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return config, nil
}
