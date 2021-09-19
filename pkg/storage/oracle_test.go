package storage

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

type rtFunc func(req *http.Request) *http.Response

func (f rtFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req), nil }

func newTestClient(fn rtFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestOracleSave(t *testing.T) {
	client, err := NewOracleDataStorageClient("test-url/")
	client.client = newTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader("")),
			Header: map[string][]string{
				"Opc-Content-Md5": {"CY9rzUYh03PK3k6DJie09g=="},
			},
		}
	})

	tempFile, err := ioutil.TempFile("", "oracle_test.file")
	if err != nil {
		t.Errorf("%v", err)
	}
	defer func() {
		_ = tempFile.Close()
		err := os.Remove(tempFile.Name())
		if err != nil {
			t.Errorf("%v", err)
		}
	}()

	_, err = tempFile.WriteString("test")
	if err != nil {
		return
	}

	err = client.Save("oracle_test.file", tempFile.Name())
	if err != nil {
		t.Errorf("can't save, err: %v", err)
	}
}
