package storage

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type OracleDataStorageClient struct {
	accessURL string
	client    *http.Client
}

// NewOracleDataStorageClient returns either a new Oracle Data Storage
// client or some error in case of failure.
// Oracle infrastructure access is based on pre-authenticated requests,
// see: https://docs.oracle.com/en-us/iaas/Content/Object/Tasks/usingpreauthenticatedrequests.htm
//
// It follows broken Google Cloud Storage client design.
func NewOracleDataStorageClient(accessURL string) (*OracleDataStorageClient, error) {
	if accessURL == "" {
		return nil, errors.New("pre-authenticated request was not specified")
	}
	return &OracleDataStorageClient{
		accessURL: accessURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (s *OracleDataStorageClient) Save(name string, localPath string) (err error) {
	if s == nil {
		return nil
	}

	dat, err := ioutil.ReadFile(localPath)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", s.accessURL+name, bytes.NewBuffer(dat))
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	dstMD5 := resp.Header.Get("Opc-Content-Md5")
	srcMD5 := base64.StdEncoding.EncodeToString(md5Hash(dat))
	if dstMD5 != srcMD5 {
		return fmt.Errorf("MD5 mismatch %v != %v", srcMD5, dstMD5)
	}

	return nil
}

func (s *OracleDataStorageClient) Load(name string) (data []byte, err error) {
	if s == nil {
		return nil, errors.New("cloud storage was not initialized")
	}

	res, err := s.client.Get(s.accessURL + name)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}

	dat, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	dstMD5 := res.Header.Get("Content-Md5")
	srcMD5 := base64.StdEncoding.EncodeToString(md5Hash(dat))
	if dstMD5 != srcMD5 {
		return nil, fmt.Errorf("MD5 mismatch %v != %v", srcMD5, dstMD5)
	}

	return dat, nil
}

func md5Hash(data []byte) []byte {
	hash := md5.New()
	hash.Write(data)
	return hash.Sum(nil)
}
