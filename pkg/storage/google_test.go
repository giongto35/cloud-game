package storage

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestSaveGame(t *testing.T) {
	client, _ := NewGoogleCloudClient()
	if client == nil {
		t.Skip("Cloud storage is not initialized")
	}
	data := []byte("Test Hello")

	file, err := ioutil.TempFile("", "test_cloud_save")
	if err != nil {
		t.Errorf("Temp dir is not accessable %v", err)
	}
	defer func(name string) {
		err = os.Remove(name)
	}(file.Name())

	if err = ioutil.WriteFile(file.Name(), data, 0644); err != nil {
		t.Errorf("File is not writable %v", err)
	}

	err = client.Save("Test", file.Name())
	if err != nil {
		log.Panic(err)
	}
	loadData, err := client.Load("Test")
	if err != nil {
		log.Panic(err)
	}
	if string(data) != string(loadData) {
		log.Panic("Failed")
	}
}
