package pipe

import (
	"os"

	"github.com/giongto35/cloud-game/v2/pkg/compression"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

func Unpack(dest string, files []string, log *logger.Logger) []string {
	var res []string
	for _, file := range files {
		if unpack := compression.NewFromExt(file, log); unpack != nil {
			if _, err := unpack.Extract(file, dest); err == nil {
				res = append(res, file)
			}
		}
	}
	return res
}

func Delete(_ string, files []string, _ *logger.Logger) []string {
	var res []string
	for _, file := range files {
		if e := os.Remove(file); e == nil {
			res = append(res, file)
		}
	}
	return res
}
