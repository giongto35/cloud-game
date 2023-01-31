package os

import (
	"errors"
	"io/fs"
	"os"
	"os/signal"
	"os/user"
	"syscall"
)

var ErrNotExist = os.ErrNotExist

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func CheckCreateDir(path string) error {
	if !Exists(path) {
		return os.MkdirAll(path, os.ModeDir|0755)
	}
	return nil
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

func WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
