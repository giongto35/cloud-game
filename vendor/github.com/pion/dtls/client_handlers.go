package dtls

import (
	"bytes"
	"fmt"
)

func clientHandshakeHandler(c *Conn) error {
	handleSingleHandshake := func(buf []byte) error {
		rawHandshake := &handshake{}
		if err := rawHandshake.Unmarshal(buf); err != nil {
			return err
		}

		c.log.Tracef("[handshake] <- %s", rawHandshake.handshakeMessage.handshakeType().String())
		switch h := rawHandshake.handshakeMessage.(type) {
		case *handshakeMessageHelloVerifyRequest:
			c.cookie = append([]byte{}, h.cookie...)

		case *handshakeMessageServerHello:
			for _, extension := range h.extensions {
				if e, ok := extension.(*extensionUseSRTP); ok {
					profile, ok := findMatchingSRTPProfile(e.protectionProfiles, c.localSRTPProtectionProfiles)
					if !ok {
						return fmt.Errorf("Server responded with SRTP Profile we do not support")
					}
					c.state.srtpProtectionProfile = profile
				}
			}
			if len(c.localSRTPProtectionProfiles) > 0 && c.state.srtpProtectionProfile == 0 {
				return fmt.Errorf("SRTP support was requested but server did not respond with use_srtp extension")
			}
			if _, ok := findMatchingCipherSuite([]cipherSuite{h.cipherSuite}, c.localCipherSuites); !ok {
				return errCipherSuiteNoIntersection
			}

			c.state.cipherSuite = h.cipherSuite
			c.state.remoteRandom = h.random
			c.log.Tracef("[handshake] use cipher suite: %s", h.cipherSuite.String())

		case *handshakeMessageCertificate:
			c.state.remoteCertificate = h.certificate

		case *handshakeMessageServerKeyExchange:
			c.remoteKeypair = &namedCurveKeypair{h.namedCurve, h.publicKey, nil}

			clientRandom, err := c.state.localRandom.Marshal()
			if err != nil {
				return err
			}
			serverRandom, err := c.state.remoteRandom.Marshal()
			if err != nil {
				return err
			}

			c.localKeypair, err = generateKeypair(h.namedCurve)
			if err != nil {
				return err
			}

			preMasterSecret, err := prfPreMasterSecret(c.remoteKeypair.publicKey, c.localKeypair.privateKey, c.localKeypair.curve)
			if err != nil {
				return err
			}

			c.state.masterSecret, err = prfMasterSecret(preMasterSecret, clientRandom, serverRandom, c.state.cipherSuite.hashFunc())
			if err != nil {
				return err
			}

			if err := c.state.cipherSuite.init(c.state.masterSecret, clientRandom, serverRandom /* isClient */, true); err != nil {
				return err
			}

			expectedHash := valueKeySignature(clientRandom, serverRandom, h.publicKey, h.namedCurve, h.hashAlgorithm)
			if err := verifyKeySignature(expectedHash, h.signature, h.hashAlgorithm, c.state.remoteCertificate); err != nil {
				return err
			}

		case *handshakeMessageCertificateRequest:
			c.remoteRequestedCertificate = true
		case *handshakeMessageServerHelloDone:
		case *handshakeMessageFinished:
			plainText := c.handshakeCache.pullAndMerge(
				handshakeCachePullRule{handshakeTypeClientHello, true},
				handshakeCachePullRule{handshakeTypeServerHello, false},
				handshakeCachePullRule{handshakeTypeCertificate, false},
				handshakeCachePullRule{handshakeTypeServerKeyExchange, false},
				handshakeCachePullRule{handshakeTypeCertificateRequest, false},
				handshakeCachePullRule{handshakeTypeServerHelloDone, false},
				handshakeCachePullRule{handshakeTypeCertificate, true},
				handshakeCachePullRule{handshakeTypeClientKeyExchange, true},
				handshakeCachePullRule{handshakeTypeCertificateVerify, true},
				handshakeCachePullRule{handshakeTypeFinished, true},
			)

			expectedVerifyData, err := prfVerifyDataServer(c.state.masterSecret, plainText, c.state.cipherSuite.hashFunc())
			if err != nil {
				return err
			}
			if !bytes.Equal(expectedVerifyData, h.verifyData) {
				return errVerifyDataMismatch
			}
		default:
			return fmt.Errorf("unhandled handshake %d", h.handshakeType())
		}

		return nil
	}

	switch c.currFlight.get() {
	case flight1:
		// HelloVerifyRequest can be skipped by the server, so allow ServerHello during flight1 also
		expectedMessages := c.handshakeCache.pull(
			handshakeCachePullRule{handshakeTypeHelloVerifyRequest, false},
			handshakeCachePullRule{handshakeTypeServerHello, false},
		)

		switch {
		case expectedMessages[0] != nil:
			if err := handleSingleHandshake(expectedMessages[0].data); err != nil {
				return err
			}
			c.state.localSequenceNumber++
		case expectedMessages[1] != nil:
			if err := handleSingleHandshake(expectedMessages[1].data); err != nil {
				return err
			}
		default:
			return nil // We have no messages we can handle yet
		}

		c.log.Tracef("[handshake] Flight 1 changed to %s", flight3.String())
		if err := c.currFlight.set(flight3); err != nil {
			return err
		}
	case flight3:
		expectedMessages := c.handshakeCache.pull(
			handshakeCachePullRule{handshakeTypeServerHello, false},
			handshakeCachePullRule{handshakeTypeCertificate, false},
			handshakeCachePullRule{handshakeTypeServerKeyExchange, false},
			handshakeCachePullRule{handshakeTypeCertificateRequest, false},
			handshakeCachePullRule{handshakeTypeServerHelloDone, false},
		)
		// We don't have enough data to even assert validity
		if expectedMessages[0] == nil {
			return nil
		}

		expectedSeqnum := expectedMessages[0].messageSequence
		for i, msg := range expectedMessages {
			switch {
			// handshakeMessageCertificateRequest can be nil, just make sure we have no gaps
			case i == 3 && msg == nil:
				continue
			case msg == nil:
				return nil // We don't have all messages yet, try again later
			case msg.messageSequence != expectedSeqnum:
				return nil // We have a gap, still waiting on messages
			}
			expectedSeqnum++
		}

		for _, msg := range expectedMessages {
			if msg != nil {
				if err := handleSingleHandshake(msg.data); err != nil {
					return err
				}
			}
		}
		c.state.localSequenceNumber++
		c.log.Tracef("[handshake] Flight 3 changed to %s", flight5.String())
		if err := c.currFlight.set(flight5); err != nil {
			return err
		}
	case flight5:
		expectedMessages := c.handshakeCache.pull(
			handshakeCachePullRule{handshakeTypeFinished, false},
		)

		if expectedMessages[0] == nil {
			return nil
		} else if err := handleSingleHandshake(expectedMessages[0].data); err != nil {
			return err
		}

		c.setLocalEpoch(1)
		c.state.localSequenceNumber = 1
		c.signalHandshakeComplete()
	default:
		return fmt.Errorf("client asked to handle unknown flight (%d)", c.currFlight.get())
	}

	return nil
}

func clientFlightHandler(c *Conn) (bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	switch c.currFlight.get() {
	case flight1:
		fallthrough
	case flight3:
		extensions := []extension{
			&extensionSupportedEllipticCurves{
				ellipticCurves: []namedCurve{namedCurveX25519, namedCurveP256},
			},
			&extensionSupportedPointFormats{
				pointFormats: []ellipticCurvePointFormat{ellipticCurvePointFormatUncompressed},
			},
			&extensionSupportedSignatureAlgorithms{
				signatureHashAlgorithms: []signatureHashAlgorithm{
					{HashAlgorithmSHA256, signatureAlgorithmECDSA},
					{HashAlgorithmSHA384, signatureAlgorithmECDSA},
					{HashAlgorithmSHA512, signatureAlgorithmECDSA},
					{HashAlgorithmSHA256, signatureAlgorithmRSA},
					{HashAlgorithmSHA384, signatureAlgorithmRSA},
					{HashAlgorithmSHA512, signatureAlgorithmRSA},
				},
			},
		}
		if len(c.localSRTPProtectionProfiles) > 0 {
			extensions = append(extensions, &extensionUseSRTP{
				protectionProfiles: c.localSRTPProtectionProfiles,
			})
		}

		c.internalSend(&recordLayer{
			recordLayerHeader: recordLayerHeader{
				sequenceNumber:  c.state.localSequenceNumber,
				protocolVersion: protocolVersion1_2,
			},
			content: &handshake{
				// sequenceNumber and messageSequence line up, may need to be re-evaluated
				handshakeHeader: handshakeHeader{
					messageSequence: uint16(c.state.localSequenceNumber),
				},
				handshakeMessage: &handshakeMessageClientHello{
					version:            protocolVersion1_2,
					cookie:             c.cookie,
					random:             c.state.localRandom,
					cipherSuites:       c.localCipherSuites,
					compressionMethods: defaultCompressionMethods,
					extensions:         extensions,
				}},
		}, false)
	case flight5:
		// TODO: Better way to end handshake
		if c.getRemoteEpoch() != 0 && c.getLocalEpoch() == 1 {
			// Handshake is done
			return true, nil
		}

		sequenceNumber := c.state.localSequenceNumber
		if c.remoteRequestedCertificate {
			c.internalSend(&recordLayer{
				recordLayerHeader: recordLayerHeader{
					sequenceNumber:  c.state.localSequenceNumber,
					protocolVersion: protocolVersion1_2,
				},
				content: &handshake{
					// sequenceNumber and messageSequence line up, may need to be re-evaluated
					handshakeHeader: handshakeHeader{
						messageSequence: uint16(c.state.localSequenceNumber),
					},
					handshakeMessage: &handshakeMessageCertificate{
						certificate: c.localCertificate,
					}},
			}, false)
			sequenceNumber++
		}

		c.internalSend(&recordLayer{
			recordLayerHeader: recordLayerHeader{
				sequenceNumber:  sequenceNumber,
				protocolVersion: protocolVersion1_2,
			},
			content: &handshake{
				// sequenceNumber and messageSequence line up, may need to be re-evaluated
				handshakeHeader: handshakeHeader{
					messageSequence: uint16(sequenceNumber),
				},
				handshakeMessage: &handshakeMessageClientKeyExchange{
					publicKey: c.localKeypair.publicKey,
				}},
		}, false)
		sequenceNumber++

		if c.remoteRequestedCertificate {
			if len(c.localCertificateVerify) == 0 {
				plainText := c.handshakeCache.pullAndMerge(
					handshakeCachePullRule{handshakeTypeClientHello, true},
					handshakeCachePullRule{handshakeTypeServerHello, false},
					handshakeCachePullRule{handshakeTypeCertificate, false},
					handshakeCachePullRule{handshakeTypeServerKeyExchange, false},
					handshakeCachePullRule{handshakeTypeCertificateRequest, false},
					handshakeCachePullRule{handshakeTypeServerHelloDone, false},
					handshakeCachePullRule{handshakeTypeCertificate, true},
					handshakeCachePullRule{handshakeTypeClientKeyExchange, true},
				)

				certVerify, err := generateCertificateVerify(plainText, c.localPrivateKey)
				if err != nil {
					return false, err
				}
				c.localCertificateVerify = certVerify
			}

			c.internalSend(&recordLayer{
				recordLayerHeader: recordLayerHeader{
					sequenceNumber:  sequenceNumber,
					protocolVersion: protocolVersion1_2,
				},
				content: &handshake{
					// sequenceNumber and messageSequence line up, may need to be re-evaluated
					handshakeHeader: handshakeHeader{
						messageSequence: uint16(sequenceNumber),
					},
					handshakeMessage: &handshakeMessageCertificateVerify{
						hashAlgorithm:      HashAlgorithmSHA256,
						signatureAlgorithm: signatureAlgorithmECDSA,
						signature:          c.localCertificateVerify,
					}},
			}, false)
			sequenceNumber++
		}

		c.internalSend(&recordLayer{
			recordLayerHeader: recordLayerHeader{
				sequenceNumber:  sequenceNumber,
				protocolVersion: protocolVersion1_2,
			},
			content: &changeCipherSpec{},
		}, false)

		if len(c.localVerifyData) == 0 {
			plainText := c.handshakeCache.pullAndMerge(
				handshakeCachePullRule{handshakeTypeClientHello, true},
				handshakeCachePullRule{handshakeTypeServerHello, false},
				handshakeCachePullRule{handshakeTypeCertificate, false},
				handshakeCachePullRule{handshakeTypeServerKeyExchange, false},
				handshakeCachePullRule{handshakeTypeCertificateRequest, false},
				handshakeCachePullRule{handshakeTypeServerHelloDone, false},
				handshakeCachePullRule{handshakeTypeCertificate, true},
				handshakeCachePullRule{handshakeTypeClientKeyExchange, true},
				handshakeCachePullRule{handshakeTypeCertificateVerify, true},
			)

			var err error
			c.localVerifyData, err = prfVerifyDataClient(c.state.masterSecret, plainText, c.state.cipherSuite.hashFunc())
			if err != nil {
				return false, err
			}
		}

		// TODO: Fix hard-coded epoch & sequenceNumber, taking retransmitting into account.
		c.internalSend(&recordLayer{
			recordLayerHeader: recordLayerHeader{
				epoch:           1,
				sequenceNumber:  0, // sequenceNumber restarts per epoch
				protocolVersion: protocolVersion1_2,
			},
			content: &handshake{
				// sequenceNumber and messageSequence line up, may need to be re-evaluated
				handshakeHeader: handshakeHeader{
					messageSequence: uint16(sequenceNumber), // KeyExchange + 1
				},
				handshakeMessage: &handshakeMessageFinished{
					verifyData: c.localVerifyData,
				}},
		}, true)
	default:
		return false, fmt.Errorf("unhandled flight %s", c.currFlight.get())
	}
	return false, nil
}
