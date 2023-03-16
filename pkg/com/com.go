package com

import "github.com/giongto35/cloud-game/v3/pkg/logger"

type NetClient interface {
	Disconnect()
	Id() Uid
}

type NetMap[T NetClient] struct{ Map[Uid, T] }

func NewNetMap[T NetClient]() NetMap[T] { return NetMap[T]{Map: Map[Uid, T]{m: make(map[Uid]T, 10)}} }

func (m *NetMap[T]) Add(client T)              { m.Put(client.Id(), client) }
func (m *NetMap[T]) Remove(client T)           { m.Map.Remove(client.Id()) }
func (m *NetMap[T]) RemoveDisconnect(client T) { client.Disconnect(); m.Remove(client) }

type SocketClient[T ~uint8, P Packet[T], X any, P2 Packet2[X]] struct {
	id   Uid
	rpc  *RPC[T, P]
	sock *Connection
	log  *logger.Logger // a special logger for showing x -> y directions
}

func NewConnection[T ~uint8, P Packet[T], X any, P2 Packet2[X]](conn *Connection, id Uid, log *logger.Logger) *SocketClient[T, P, X, P2] {
	if id.IsNil() {
		id = NewUid()
	}
	dir := logger.MarkOut
	if conn.IsServer() {
		dir = logger.MarkIn
	}
	dirClLog := log.Extend(log.With().
		Str("cid", id.Short()).
		Str(logger.DirectionField, dir),
	)
	dirClLog.Debug().Msg("Connect")
	return &SocketClient[T, P, X, P2]{sock: conn, id: id, log: dirClLog}
}

func (c *SocketClient[T, P, _, _]) ProcessPackets(fn func(in P) error) chan struct{} {
	c.rpc = NewRPC[T, P]()
	c.rpc.Handler = func(p P) {
		c.log.Debug().Str(logger.DirectionField, logger.MarkIn).Msgf("%v", p.GetType())
		if err := fn(p); err != nil { // 3rd handler
			c.log.Error().Err(err).Send()
		}
	}
	c.sock.conn.SetMessageHandler(c.handleMessage) // 1st handler
	return c.sock.conn.Listen()
}

func (c *SocketClient[_, _, _, _]) handleMessage(message []byte, err error) {
	if err != nil {
		c.log.Error().Err(err).Send()
		return
	}
	if err = c.rpc.handleMessage(message); err != nil { // 2nd handler
		c.log.Error().Err(err).Send()
		return
	}
}

func (c *SocketClient[_, P, X, P2]) Route(in P, out P2) {
	rq := P2(new(X))
	rq.SetId(in.GetId().String())
	rq.SetType(uint8(in.GetType()))
	rq.SetPayload(out.GetPayload())
	if err := c.rpc.Send(c.sock.conn, rq); err != nil {
		c.log.Error().Err(err).Msgf("message route fail")
	}
}

// Send makes a blocking call.
func (c *SocketClient[T, P, X, P2]) Send(t T, data any) ([]byte, error) {
	c.log.Debug().Str(logger.DirectionField, logger.MarkOut).Msgf("ᵇ%v", t)
	rq := P2(new(X))
	rq.SetType(uint8(t))
	rq.SetPayload(data)
	return c.rpc.Call(c.sock.conn, rq)
}

// Notify just sends a message and goes further.
func (c *SocketClient[T, P, X, P2]) Notify(t T, data any) {
	c.log.Debug().Str(logger.DirectionField, logger.MarkOut).Msgf("%v", t)
	rq := P2(new(X))
	rq.SetType(uint8(t))
	rq.SetPayload(data)
	if err := c.rpc.Send(c.sock.conn, rq); err != nil {
		c.log.Error().Err(err).Msgf("notify fail")
	}
}

func (c *SocketClient[_, _, _, _]) Disconnect() {
	c.sock.conn.Close()
	c.rpc.Cleanup()
	c.log.Debug().Str(logger.DirectionField, logger.MarkCross).Msg("Close")
}

func (c *SocketClient[_, _, _, _]) Id() Uid        { return c.id }
func (c *SocketClient[_, _, _, _]) String() string { return c.Id().String() }
