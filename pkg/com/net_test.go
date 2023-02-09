package com

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
)

var log = logger.Default()

func TestPackets(t *testing.T) {
	r, err := json.Marshal(Out{Payload: "asd"})
	if err != nil {
		t.Fatalf("can't marshal packet")
	}

	t.Logf("PACKET: %v", string(r))
}

func TestWebsocket(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{"If WebSocket implementation is OK in general", testWebsocket},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func testWebsocket(t *testing.T) {
	var wg sync.WaitGroup
	sh := newServer(t)
	client := newClient(t, url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"})
	client.OnPacket(func(packet In) {
		//	nop
	})
	client.Listen()
	wg.Wait()

	server := sh.s

	if server == nil {
		t.Fatalf("couldn't make new socket")
	}

	calls := []struct {
		typ        api.PT
		payload    any
		concurrent bool
		value      any
	}{
		{typ: 10, payload: "test", value: "test", concurrent: true},
		{typ: 10, payload: "test2", value: "test2"},
		{typ: 11, payload: "test3", value: "test3"},
		{typ: 99, payload: "", value: ""},
		{typ: 0},
		{typ: 12, payload: 123, value: 123},
		{typ: 10, payload: false, value: false},
		{typ: 10, payload: true, value: true},
		{typ: 11, payload: []string{"test", "test", "test"}, value: []string{"test", "test", "test"}},
		{typ: 22, payload: []string{}, value: []string{}},
	}

	const n = 42
	var wait sync.WaitGroup
	wait.Add(n * len(calls))

	// test
	for _, call := range calls {
		call := call
		if call.concurrent {
			rand.New(rand.NewSource(time.Now().UnixNano()))
			for i := 0; i < n; i++ {
				go func() {
					defer wait.Done()
					time.Sleep(time.Duration(rand.Intn(200-100)+100) * time.Millisecond)
					vv, err := client.Call(call.typ, call.payload)
					err = checkCall(vv, err, call.value)
					if err != nil {
						t.Errorf("%v", err)
						return
					}
				}()
			}
		} else {
			for i := 0; i < n; i++ {
				vv, err := client.Call(call.typ, call.payload)
				err = checkCall(vv, err, call.value)
				if err != nil {
					wait.Done()
					t.Fatalf("%v", err)
				} else {
					wait.Done()
				}
			}
		}
	}
	wait.Wait()

	client.Close()
	<-client.conn.Done
	server.Close()
	<-server.Done
}

func newClient(t *testing.T, addr url.URL) *Client {
	conn, err := NewConnector().NewClient(addr, log)
	if err != nil {
		t.Fatalf("error: couldn't connect to %v because of %v", addr.String(), err)
	}
	return conn
}

func checkCall(v []byte, err error, need any) error {
	if err != nil {
		return err
	}
	var value any
	if v != nil {
		if err = json.Unmarshal(v, &value); err != nil {
			return fmt.Errorf("can't unmarshal %v", v)
		}
	}

	nice := true
	// cast values after default unmarshal
	switch value.(type) {
	default:
		nice = value == need
	case bool:
		nice = value == need.(bool)
	case float64:
		nice = value == float64(need.(int))
	case string:
		nice = value == need.(string)
	case []any:
		// let's assume that's strings
		vv := value.([]any)
		for i := 0; i < len(need.([]string)); i++ {
			if vv[i].(string) != need.([]string)[i] {
				nice = false
				break
			}
		}
	case map[string]any:
		// ???
	}

	if !nice {
		return fmt.Errorf("expected %v, but got %v", need, v)
	}
	return nil
}

type serverHandler struct {
	s *websocket.WS // ws server reference made dynamically on HTTP request
}

func (s *serverHandler) serve(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.DefaultUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("no socket, %v", err)
		}
		sock, err := websocket.NewServerWithConn(conn, log)
		if err != nil {
			t.Fatalf("couldn't init socket server")
		}
		s.s = sock
		s.s.OnMessage = func(m []byte, err error) { s.s.Write(m) } // echo
		s.s.Listen()
	}
}

func newServer(t *testing.T) *serverHandler {
	var wg sync.WaitGroup
	handler := serverHandler{}
	http.HandleFunc("/ws", handler.serve(t))
	wg.Add(1)
	go func() {
		wg.Done()
		if err := http.ListenAndServe(":8080", nil); err != nil {
			t.Errorf("no server")
			return
		}
	}()
	wg.Wait()
	return &handler
}
