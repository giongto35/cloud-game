package turnc

import (
	"io"
	"io/ioutil"
	"net"

	"github.com/gortc/turn"

	"github.com/pion/logging"
	"github.com/pion/stun"
)

// multiplexer de-multiplexes STUN, TURN and application data
// from one connection into separate ones.
type multiplexer struct {
	log      logging.LeveledLogger
	capacity int
	conn     net.Conn

	stunL, stunR net.Conn
	turnL, turnR net.Conn
	dataL, dataR net.Conn
}

func newMultiplexer(conn net.Conn, log logging.LeveledLogger) *multiplexer {
	m := &multiplexer{conn: conn, capacity: 1500, log: log}
	m.stunL, m.stunR = net.Pipe()
	m.turnL, m.turnR = net.Pipe()
	m.dataL, m.dataR = net.Pipe()
	go m.readUntilClosed()
	return m
}

func (m *multiplexer) discardData() {
	discardLogged(m.log, "mux: failed to discard dataL: %v", m.dataL)
}

func discardLogged(l logging.LeveledLogger, msg string, r io.Reader) {
	_, err := io.Copy(ioutil.Discard, r)
	if err != nil {
		l.Errorf(msg, err)
	}
}

func closeLogged(l logging.LeveledLogger, msg string, conn io.Closer) {
	if closeErr := conn.Close(); closeErr != nil {
		l.Errorf(msg, closeErr)
	}
}

func (m *multiplexer) close() {
	closeLogged(m.log, "mux: failed to close turnR: %v", m.turnR)
	closeLogged(m.log, "mux: failed to close stunR: %v", m.stunR)
	closeLogged(m.log, "mux: failed to close dataR: %v", m.dataR)
}

func (m *multiplexer) readUntilClosed() {
	buf := make([]byte, m.capacity)
	for {
		n, err := m.conn.Read(buf)
		m.log.Debugf("mux: read %d err: %v", n, err)
		if err != nil {
			// End of cycle.
			// TODO: Handle timeouts and temporary errors.
			m.log.Info("connection closed")
			m.close()
			break
		}
		data := buf[:n]
		conn := m.dataR
		switch {
		case stun.IsMessage(data):
			m.log.Debug("mux: got STUN data")
			conn = m.stunR
		case turn.IsChannelData(data):
			m.log.Debug("mux: got TURN data")
			conn = m.turnR
		default:
			m.log.Debug("mux: got APP data")
		}
		_, err = conn.Write(data)
		if err != nil {
			m.log.Warnf("failed to write: %v", err)
		}
	}
}
