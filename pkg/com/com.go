package com

import "github.com/giongto35/cloud-game/v2/pkg/logger"

type NetClient[K comparable] interface {
	Disconnect()
	Id() K
}

type NetMap[K comparable, T NetClient[K]] struct{ Map[K, T] }

func NewNetMap[K comparable, T NetClient[K]]() NetMap[K, T] {
	return NetMap[K, T]{Map: Map[K, T]{m: make(map[K]T, 10)}}
}

func (m *NetMap[K, T]) Add(client T)              { m.Put(client.Id(), client) }
func (m *NetMap[K, T]) Remove(client T)           { m.RemoveByKey(client.Id()) }
func (m *NetMap[K, T]) RemoveDisconnect(client T) { client.Disconnect(); m.Remove(client) }

type SocketClient[I Id, T ~uint8, P Packet[I, T], X any, P2 Packet2[X]] struct {
	id        Uid
	sock      *Connection
	transport *Transport[I, T, P]
	log       *logger.Logger // a special logger for showing x -> y directions
}

func NewConnection[I Id, T ~uint8, P Packet[I, T], X any, P2 Packet2[X]](conn *Connection, id Uid, isServer bool, log *logger.Logger) *SocketClient[I, T, P, X, P2] {
	if id.IsNil() {
		id = NewUid()
	}
	dir := "→"
	if isServer {
		dir = "←"
	}
	dirClLog := log.Extend(log.With().
		Str("cid", id.Short()).
		Str(logger.DirectionField, dir),
	)
	dirClLog.Debug().Msg("Connect")
	return &SocketClient[I, T, P, X, P2]{sock: conn, id: id, log: dirClLog}
}

func (c *SocketClient[I, T, P, _, _]) OnPacket(fn func(in P) error) {
	transport := new(Transport[I, T, P])
	transport.calls = Map[I, *request]{m: make(map[I]*request, 10)}
	transport.Handler = func(p P) {
		c.log.Debug().Str(logger.DirectionField, "←").Msgf("%v", p.GetType())
		if err := fn(p); err != nil {
			c.log.Error().Err(err).Send()
		}
	}
	c.transport = transport
	c.sock.conn.SetMessageHandler(c.handleMessage)
}

func (c *SocketClient[_, _, _, _, _]) handleMessage(message []byte, err error) {
	if err != nil {
		c.log.Error().Err(err).Send()
		return
	}
	if err = c.transport.handleMessage(message); err != nil {
		c.log.Error().Err(err).Send()
		return
	}
}

func (c *SocketClient[_, _, P, X, P2]) Route(in P, out P2) {
	rq := P2(new(X))
	rq.SetId(in.GetId().String())
	rq.SetType(uint8(in.GetType()))
	rq.SetPayload(out.GetPayload())
	_ = c.transport.SendAsync(c.sock.conn, rq)
}

// Send makes a blocking call.
func (c *SocketClient[_, T, P, X, P2]) Send(t T, data any) ([]byte, error) {
	c.log.Debug().Str(logger.DirectionField, "→").Msgf("ᵇ%v", t)
	rq := P2(new(X))
	rq.SetType(uint8(t))
	rq.SetPayload(data)
	return c.transport.SendSync(c.sock.conn, rq)
}

// Notify just sends a message and goes further.
func (c *SocketClient[_, T, P, X, P2]) Notify(t T, data any) {
	c.log.Debug().Str(logger.DirectionField, "→").Msgf("%v", t)
	rq := P2(new(X))
	rq.SetType(uint8(t))
	rq.SetPayload(data)
	_ = c.transport.SendAsync(c.sock.conn, rq)
}

func (c *SocketClient[_, _, _, _, _]) Disconnect() {
	c.sock.conn.Close()
	c.transport.Clean()
	c.log.Debug().Str(logger.DirectionField, "x").Msg("Close")
}

func (c *SocketClient[_, _, _, _, _]) Id() Uid               { return c.id }
func (c *SocketClient[_, _, _, _, _]) Listen() chan struct{} { return c.sock.conn.Listen() }
func (c *SocketClient[_, _, _, _, _]) String() string        { return c.Id().String() }
