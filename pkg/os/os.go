package os

import (
	"errors"
	"io/fs"
	"os"
	"os/signal"
	"os/user"
	"syscall"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
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
