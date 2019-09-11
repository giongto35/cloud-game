package logging

import (
	"flag"
	"log"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

func init() {
	_ = flag.Set("logtostderr", "true")
}

// LogWriter serves as a bridge between the standard log package and the glog package.
type LogWriter struct{}

// Write implements the io.Writer interface.
func (writer LogWriter) Write(data []byte) (n int, err error) {
	glog.InfoDepth(3, string(data))
	return len(data), nil
}

// Init initializes logs the way we want.
func Init() {
	log.SetOutput(LogWriter{})
	log.SetFlags(0)

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	// Convinces goflags that we have called Parse() to avoid noisy logs.
	_ = flag.CommandLine.Parse([]string{})
}

// Flush flushes logs immediately.
func Flush() {
	glog.Flush()
}
