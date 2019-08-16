package dtls

import "sync"

/*
  DTLS messages are grouped into a series of message flights, according
  to the diagrams below.  Although each flight of messages may consist
  of a number of messages, they should be viewed as monolithic for the
  purpose of timeout and retransmission.
  https://tools.ietf.org/html/rfc4347#section-4.2.4
  Client                                          Server
  ------                                          ------
                                      Waiting                 Flight 0

  ClientHello             -------->                           Flight 1

                          <-------    HelloVerifyRequest      Flight 2

  ClientHello              -------->                           Flight 3

                                             ServerHello    \
                                            Certificate*     \
                                      ServerKeyExchange*      Flight 4
                                     CertificateRequest*     /
                          <--------      ServerHelloDone    /

  Certificate*                                              \
  ClientKeyExchange                                          \
  CertificateVerify*                                          Flight 5
  [ChangeCipherSpec]                                         /
  Finished                -------->                         /

                                      [ChangeCipherSpec]    \ Flight 6
                          <--------             Finished    /

*/

type flightVal uint8

const (
	flight0 flightVal = iota + 1
	flight1
	flight2
	flight3
	flight4
	flight5
	flight6
)

func (f flightVal) String() string {
	switch f {
	case flight0:
		return "Flight 0"
	case flight1:
		return "Flight 1"
	case flight2:
		return "Flight 2"
	case flight3:
		return "Flight 3"
	case flight4:
		return "Flight 4"
	case flight5:
		return "Flight 5"
	case flight6:
		return "Flight 6"
	default:
		return "Invalid Flight"
	}
}

type flight struct {
	sync.RWMutex
	val           flightVal
	workerTrigger chan struct{} // Temporary way to trigger next flight
}

func newFlight(isClient bool) *flight {
	val := flight0
	if isClient {
		val = flight1
	}
	return &flight{val: val, workerTrigger: make(chan struct{}, 1)}
}

func (f *flight) get() flightVal {
	f.RLock()
	defer f.RUnlock()
	return f.val
}

func (f *flight) set(val flightVal) error {
	f.Lock()
	f.val = val // TODO ensure no invalid transitions
	f.Unlock()

	select {
	case f.workerTrigger <- struct{}{}:
	default:
	}

	return nil
}
