package libretro

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"unsafe"
)

// #include <stdlib.h>
import "C"

type CoreProperties struct {
	m  map[string]*C.char
	mu sync.Mutex
}

func ReadProperties(filename string) (*CoreProperties, error) {
	config := CoreProperties{
		m: make(map[string]*C.char),
	}

	if len(filename) == 0 {
		return &config, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return &config, fmt.Errorf("couldn't find the %v config file", filename)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	config.mu.Lock()
	defer config.mu.Unlock()
	for scanner.Scan() {
		line := scanner.Text()
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				config.m[key] = C.CString(value)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return &config, nil
}

func (c *CoreProperties) Get(key string) (*C.char, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	return v, ok
}

func (c *CoreProperties) Free() {
	c.mu.Lock()
	for _, element := range c.m {
		C.free(unsafe.Pointer(element))
	}
	c.mu.Unlock()
}
