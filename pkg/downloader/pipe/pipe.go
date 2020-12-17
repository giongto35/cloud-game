package pipe

import (
	"os"

	"github.com/giongto35/cloud-game/v2/pkg/extractor"
)

func Unpack(dest string, files []string) []string {
	var res []string
	for _, file := range files {
		if unpack := extractor.NewFromExt(file); unpack != nil {
			if _, err := unpack.Extract(file, dest); err == nil {
				res = append(res, file)
			}
		}
	}
	return res
}

func Delete(_ string, files []string) []string {
	var res []string
	for _, file := range files {
		if e := os.Remove(file); e == nil {
			res = append(res, file)
		}
	}
	return res
}
