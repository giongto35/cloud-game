package turnc

import (
	"fmt"
	"net"
	"time"
)

// bypassWriter wraps reader and writer connections,
// passing Read and SetReadDeadline to reader conn,
// Write, LocalAddr, RemoteAddr, SetWriteDeadline
// calls to writer.
type bypassWriter struct {
	reader net.Conn
	writer net.Conn
}

func (w bypassWriter) Close() error {
	rErr := w.reader.Close()
	wErr := w.writer.Close()
	if rErr == nil && wErr == nil {
		return nil
	}
	return fmt.Errorf("reader: %v, writer: %v", rErr, wErr)
}

func (w bypassWriter) LocalAddr() net.Addr {
	return w.writer.LocalAddr()
}

func (w bypassWriter) Read(b []byte) (n int, err error) {
	return w.reader.Read(b)
}

func (w bypassWriter) RemoteAddr() net.Addr {
	return w.writer.RemoteAddr()
}

func (w bypassWriter) SetDeadline(t time.Time) error {
	if err := w.writer.SetDeadline(t); err != nil {
		return err
	}
	return w.reader.SetDeadline(t)
}

func (w bypassWriter) SetReadDeadline(t time.Time) error {
	return w.reader.SetReadDeadline(t)
}

func (w bypassWriter) SetWriteDeadline(t time.Time) error {
	return w.writer.SetWriteDeadline(t)
}

func (w bypassWriter) Write(b []byte) (n int, err error) {
	return w.writer.Write(b)
}
