package api

import "encoding/json"

// This list of postfixes is used in the API:
// - *Request postfix denotes clients calls (i.e. from a browser to the HTTP-server).
// - *Call postfix denotes IPC calls (from the coordinator to a worker).

func from(source interface{}, data string) error {
	err := json.Unmarshal([]byte(data), source)
	if err != nil {
		return err
	}
	return nil
}

func to(target interface{}) (string, error) {
	b, err := json.Marshal(target)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
