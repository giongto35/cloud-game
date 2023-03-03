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

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
)

type TestIn struct {
	Id      Uid             `json:"id,omitempty"`
	T       uint8           `json:"t"`
	Payload json.RawMessage `json:"p,omitempty"`
}

func (i TestIn) GetId() Uid         { return i.Id }
func (i TestIn) GetType() uint8     { return i.T }
func (i TestIn) GetPayload() []byte { return i.Payload }
func (i TestIn) HasId() bool        { return !i.Id.IsNil() }

type TestOut struct {
	Id      string `json:"id,omitempty"`
	T       uint8  `json:"t"`
	Payload any    `json:"p,omitempty"`
}

func (o *TestOut) SetId(s string)                 { o.Id = s }
func (o *TestOut) SetType(u uint8)                { o.T = u }
func (o *TestOut) SetPayload(a any)               { o.Payload = a }
func (o *TestOut) SetGetId(stringer fmt.Stringer) { o.Id = stringer.String() }
func (o *TestOut) GetPayload() any                { return o.Payload }

func TestPackets(t *testing.T) {
	r, err := json.Marshal(TestOut{Payload: "asd"})
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
	server := newServer(t)
	client := newClient(t, url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"})
	client.OnPacket(func(in TestIn) error { return nil })
	clDone := client.Listen()

	if server.conn == nil {
		t.Fatalf("couldn't make new socket")
	}

	calls := []struct {
		packet     TestOut
		concurrent bool
		value      any
	}{
		{packet: TestOut{T: 10, Payload: "test"}, value: "test", concurrent: true},
		{packet: TestOut{T: 10, Payload: "test2"}, value: "test2"},
		{packet: TestOut{T: 11, Payload: "test3"}, value: "test3"},
		{packet: TestOut{T: 99, Payload: ""}, value: ""},
		{packet: TestOut{T: 0}},
		{packet: TestOut{T: 12, Payload: 123}, value: 123},
		{packet: TestOut{T: 10, Payload: false}, value: false},
		{packet: TestOut{T: 10, Payload: true}, value: true},
		{packet: TestOut{T: 11, Payload: []string{"test", "test", "test"}}, value: []string{"test", "test", "test"}},
		{packet: TestOut{T: 22, Payload: []string{}}, value: []string{}},
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
				packet := call.packet
				go func() {
					defer wait.Done()
					time.Sleep(time.Duration(rand.Intn(200-100)+100) * time.Millisecond)
					vv, err := client.rpc.Call(client.sock.conn, &packet)
					err = checkCall(vv, err, call.value)
					if err != nil {
						t.Errorf("%v", err)
						return
					}
				}()
			}
		} else {
			for i := 0; i < n; i++ {
				packet := call.packet
				vv, err := client.rpc.Call(client.sock.conn, &packet)
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

	client.sock.conn.Close()
	client.rpc.Cleanup()
	<-clDone
	server.conn.Close()
	<-server.done
}

func newClient(t *testing.T, addr url.URL) *SocketClient[uint8, TestIn, TestOut, *TestOut] {
	connector := Client{}
	conn, err := connector.Connect(addr)
	if err != nil {
		t.Fatalf("error: couldn't connect to %v because of %v", addr.String(), err)
	}
	rpc := new(RPC[uint8, TestIn])
	rpc.calls = Map[Uid, *request]{m: make(map[Uid]*request, 10)}
	return &SocketClient[uint8, TestIn, TestOut, *TestOut]{sock: conn, log: logger.Default(), rpc: rpc}
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
	conn *websocket.Connection // ws server reference made dynamically on HTTP request
	done chan struct{}
}

func (s *serverHandler) serve(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	connector := Server{}

	return func(w http.ResponseWriter, r *http.Request) {
		sock, err := connector.Server.Connect(w, r, nil)
		if err != nil {
			t.Fatalf("couldn't init socket server")
		}
		s.conn = sock
		s.conn.SetMessageHandler(func(m []byte, err error) { s.conn.Write(m) }) // echo
		s.done = s.conn.Listen()
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
