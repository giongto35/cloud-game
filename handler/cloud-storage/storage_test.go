package storage

import (
	"io/ioutil"
	"log"
	"testing"
)

func TestSaveGame(t *testing.T) {
	client := NewInitClient()
	data := []byte("Test Hello")
	ioutil.WriteFile("/tmp/TempFile", data, 0644)
	err := client.SaveFile("Test", "/tmp/TempFile")
	if err != nil {
		log.Panic(err)
	}
	loadData, err := client.LoadFile("Test")
	if err != nil {
		log.Panic(err)
	}
	if string(data) != string(loadData) {
		log.Panic("Failed")
	}
}
