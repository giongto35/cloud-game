package os

import (
	"os"
	"os/signal"
	"syscall"
)

type Signal struct {
	event chan os.Signal
	done  chan struct{}
}

func ExpectTermination() chan struct{} {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{}, 1)
	go func() {
		<-signals
		done <- struct{}{}
	}()
	return done
}
