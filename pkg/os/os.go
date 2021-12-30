package os

import (
	"os"
	"os/signal"
	"os/user"
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

func GetUserHome() (string, error) {
	me, err := user.Current()
	if err != nil {
		return "", err
	}
	return me.HomeDir, nil
}
