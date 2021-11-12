package ipc

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
)

var log = logger.Default()

func TestPackets(t *testing.T) {
	r, err := json.Marshal(OutPacket{Payload: "asd"})
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
	// setup
	// socket handler
	var socket *websocket.WS
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		sock, err := websocket.NewServer(w, r, log)
		if err != nil {
			t.Fatalf("couldn't init socket server")
		}
		socket = sock
		socket.OnMessage = func(message []byte, err error) {
			// echo response
			socket.Write(message)
		}
		socket.Listen()
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

	client := newClient(t, url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"})
	client.Listen()

	calls := []struct {
		typ        uint8
		payload    interface{}
		concurrent bool
		value      interface{}
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

	rand.Seed(time.Now().UnixNano())

	n := 42 * 2 * 2
	var wait sync.WaitGroup
	wait.Add(n * len(calls))

	// test
	for _, call := range calls {
		for i := 0; i < n; i++ {
			if call.concurrent {
				call := call
				go func() {
					w := rand.Intn(600-100) + 100
					time.Sleep(time.Duration(w) * time.Millisecond)
					vv, err := client.Call(call.typ, call.payload)
					checkCall(t, vv, err, call.value)
					wait.Done()
				}()
			} else {
				vv, err := client.Call(call.typ, call.payload)
				checkCall(t, vv, err, call.value)
				wait.Done()
			}
		}
	}
	wait.Wait()

	client.Close()

	<-socket.Done
	<-client.Conn.Done

}

func newClient(t *testing.T, addr url.URL) *Client {
	conn, err := NewClient(addr, log)
	if err != nil {
		t.Fatalf("error: couldn't connect to %v because of %v", addr.String(), err)
	}
	return conn
}

func checkCall(t *testing.T, v []byte, err error, need interface{}) {
	if err != nil {
		t.Fatalf("should be no error but %v", err)
		return
	}
	var value interface{}
	if v != nil {
		if err = json.Unmarshal(v, &value); err != nil {
			t.Fatalf("can't unmarshal %v", v)
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
	case []interface{}:
		// let's assume that's strings
		vv := value.([]interface{})
		for i := 0; i < len(need.([]string)); i++ {
			if vv[i].(string) != need.([]string)[i] {
				nice = false
				break
			}
		}
	case map[string]interface{}:
		// ???
	}

	if !nice {
		t.Fatalf("expected %v is not expected %v", need, v)
	}
}
