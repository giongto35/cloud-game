package pipe

import (
	"github.com/giongto35/cloud-game/v2/pkg/extractor"
	"os"
)

func Unpack(dest string, strings []string) []string {
	for _, file := range strings {
		unpack := extractor.NewFromExt(file)
		if unpack != nil {
			unpack.Extract(file, dest)
		}
	}
	return strings
}

func Delete(_ string, strings []string) []string {
	var files []string
	for _, file := range strings {
		e := os.Remove(file)
		if e != nil {
			files = append(files, file)
		}
	}
	return files
}
