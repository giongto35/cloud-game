package test

import (
	"os"
	"path"
	"runtime"
)

// runs tests from the root dir when imported

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	if err := os.Chdir(dir); err != nil {
		panic(err)
	}
}
