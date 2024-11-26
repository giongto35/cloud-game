package os

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"os/user"
	"syscall"
)

const ReadChunk = 1024

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

func MakeDirAll(path string) error {
	return os.MkdirAll(path, os.ModeDir|os.ModePerm)
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

func CopyFile(from string, to string) error {
	bytesRead, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	err = os.WriteFile(to, bytesRead, 0755)
	if err != nil {
		return err
	}
	return nil
}

func WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func ReadFile(name string) (dat []byte, err error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	r := bufio.NewReader(f)
	buf := bytes.NewBuffer(make([]byte, 0))
	chunk := make([]byte, ReadChunk)

	c := 0
	for {
		if c, err = r.Read(chunk); err != nil {
			break
		}
		buf.Write(chunk[:c])
	}

	if err == io.EOF {
		err = nil
	}

	return buf.Bytes(), err
}

func StatSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func RemoveAll(path string) error {
	return os.RemoveAll(path)
}
