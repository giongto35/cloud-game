package ipc

import (
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
)

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
	// setup
	// socket handler
	var socket *websocket.WS
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		socket = websocket.NewServer(w, r)
		socket.OnMessage = func(message []byte, err error) {
			// simple echo response
			socket.Write(message)
		}
	})
	// http handler
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		if err := http.ListenAndServe(":8080", nil); err != nil {
			t.Errorf("no server")
			return
		}
	}()
	wg.Wait()

	client := connect(t, url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"})

	calls := []struct {
		typ        uint8
		opts       []PacketOption
		concurrent bool

		value interface{}
	}{
		{
			typ:        api.PtEcho,
			opts:       []PacketOption{Payload("test")},
			value:      "test",
			concurrent: true,
		},
		{
			typ:   api.PtEcho,
			opts:  []PacketOption{Payload("test2")},
			value: "test2",
		},
		{
			typ:   api.PtEcho,
			opts:  []PacketOption{Payload("test3")},
			value: "test3",
		},
		{
			typ: api.PtUnknown,
		},
	}

	rand.Seed(time.Now().UnixNano())

	// test
	for _, call := range calls {
		for i := 1; i < 1000; i++ {
			if call.concurrent {
				call := call
				go func() {
					w := rand.Intn(800-100) + 100
					time.Sleep(time.Duration(w) * time.Millisecond)
					vv, err := client.Call(call.typ, call.opts...)
					checkCall(t, vv, err, call.value)
				}()
			} else {
				vv, err := client.Call(call.typ, call.opts...)
				checkCall(t, vv, err, call.value)
			}
		}
	}

	// teardown
	client.Close()

	<-socket.Done
	<-client.Conn.Done
}

func connect(t *testing.T, addr url.URL) *Client {
	conn, err := NewClient(addr)
	if err != nil {
		t.Fatalf("error: couldn't connect to %v because of %v", addr.String(), err)
	}
	return conn
}

func checkCall(t *testing.T, v interface{}, err error, need interface{}) {
	if err != nil {
		t.Fatalf("should be no error but %v", err)
		return
	}
	if v != need {
		t.Fatalf("expected %v is not expected %v", need, v)
	}
}
