package dtls

import (
	"bytes"
	"fmt"
)

func serverHandshakeHandler(c *Conn) error {
	handleSingleHandshake := func(buf []byte) error {
		rawHandshake := &handshake{}
		if err := rawHandshake.Unmarshal(buf); err != nil {
			return err
		}

		switch h := rawHandshake.handshakeMessage.(type) {
		case *handshakeMessageClientHello:
			if c.currFlight.get() == flight2 {
				if !bytes.Equal(c.cookie, h.cookie) {
					return errCookieMismatch
				}
				c.state.localSequenceNumber = 1
				if err := c.currFlight.set(flight4); err != nil {
					return err
				}
				break
			}

			c.state.remoteRandom = h.random

			if _, ok := findMatchingCipherSuite(h.cipherSuites, c.localCipherSuites); !ok {
				return errCipherSuiteNoIntersection
			}
			c.state.cipherSuite = h.cipherSuites[0]

			for _, extension := range h.extensions {
				switch e := extension.(type) {
				case *extensionSupportedEllipticCurves:
					c.namedCurve = e.ellipticCurves[0]
				case *extensionUseSRTP:
					profile, ok := findMatchingSRTPProfile(e.protectionProfiles, c.localSRTPProtectionProfiles)
					if !ok {
						return fmt.Errorf("Client requested SRTP but we have no matching profiles")
					}
					c.state.srtpProtectionProfile = profile
				}
			}

			if c.localKeypair == nil {
				var err error
				c.localKeypair, err = generateKeypair(c.namedCurve)
				if err != nil {
					return err
				}
			}

			if err := c.currFlight.set(flight2); err != nil {
				return err
			}

		case *handshakeMessageCertificateVerify:
			if c.state.remoteCertificate == nil {
				return errCertificateVerifyNoCertificate
			}

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

			verified := false
			if !c.insecureSkipVerify && c.clientAuth >= RequireAnyClientCert {
				if err := verifyCertificateVerify(plainText, h.hashAlgorithm, h.signature, c.state.remoteCertificate); err != nil {
					return err
				}
				if c.clientAuth >= VerifyClientCertIfGiven {
					if err := verifyClientCert(c.state.remoteCertificate, c.rootCAs); err != nil {
						return err
					}
					verified = true
				}
			}
			if c.verifyPeerCertificate != nil {
				if err := c.verifyPeerCertificate(c.state.remoteCertificate, verified); err != nil {
					return err
				}
			}
			c.remoteCertificateVerified = verified

		case *handshakeMessageCertificate:
			c.state.remoteCertificate = h.certificate

		case *handshakeMessageClientKeyExchange:
			serverRandom, err := c.state.localRandom.Marshal()
			if err != nil {
				return err
			}
			clientRandom, err := c.state.remoteRandom.Marshal()
			if err != nil {
				return err
			}

			var preMasterSecret []byte
			if c.localPSKCallback != nil {
				var psk []byte
				if psk, err = c.localPSKCallback(h.identityHint); err != nil {
					return err
				}

				preMasterSecret = prfPSKPreMasterSecret(psk)
			} else {
				preMasterSecret, err = prfPreMasterSecret(h.publicKey, c.localKeypair.privateKey, c.localKeypair.curve)
				if err != nil {
					return err
				}
			}

			c.state.masterSecret, err = prfMasterSecret(preMasterSecret, clientRandom, serverRandom, c.state.cipherSuite.hashFunc())
			if err != nil {
				return err
			}

			if err := c.state.cipherSuite.init(c.state.masterSecret, clientRandom, serverRandom /* isClient */, false); err != nil {
				return err
			}
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
			)
			expectedVerifyData, err := prfVerifyDataClient(c.state.masterSecret, plainText, c.state.cipherSuite.hashFunc())
			if err != nil {
				return err
			} else if !bytes.Equal(expectedVerifyData, h.verifyData) {
				return errVerifyDataMismatch
			}

		default:
			return fmt.Errorf("unhandled handshake %d", h.handshakeType())
		}

		return nil
	}

	switch c.currFlight.get() {
	case flight0:
		expectedMessages := c.handshakeCache.pull(
			handshakeCachePullRule{handshakeTypeClientHello, true},
		)
		if expectedMessages[0] != nil && expectedMessages[0].messageSequence == 0 {
			return handleSingleHandshake(expectedMessages[0].data)
		}
	case flight2:
		expectedMessages := c.handshakeCache.pull(
			handshakeCachePullRule{handshakeTypeClientHello, true},
		)
		if expectedMessages[0] != nil && expectedMessages[0].messageSequence == 1 {
			return handleSingleHandshake(expectedMessages[0].data)
		}
	case flight4:
		expectedMessages := c.handshakeCache.pull(
			handshakeCachePullRule{handshakeTypeCertificate, true},
			handshakeCachePullRule{handshakeTypeClientKeyExchange, true},
			handshakeCachePullRule{handshakeTypeCertificateVerify, true},
		)

		var expectedSeqnum uint16
		switch {
		case expectedMessages[0] != nil:
			expectedSeqnum = expectedMessages[0].messageSequence
		case expectedMessages[1] != nil:
			expectedSeqnum = expectedMessages[1].messageSequence
		default:
			return nil
		}

		for i, msg := range expectedMessages {
			// handshakeTypeCertificate and handshakeTypeCertificateVerify can be nil, just make sure we have no gaps
			switch {
			case (i == 0 || i == 2) && msg == nil:
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

		finishedMsg := c.handshakeCache.pull(handshakeCachePullRule{handshakeTypeFinished, true})
		if finishedMsg[0] == nil {
			return nil
		} else if err := handleSingleHandshake(finishedMsg[0].data); err != nil {
			return err
		}

		switch c.clientAuth {
		case RequireAnyClientCert:
			if c.state.remoteCertificate == nil {
				return errClientCertificateRequired
			}
		case VerifyClientCertIfGiven:
			if c.state.remoteCertificate != nil && !c.remoteCertificateVerified {
				return errClientCertificateNotVerified
			}
		case RequireAndVerifyClientCert:
			if c.state.remoteCertificate == nil {
				return errClientCertificateRequired
			}
			if !c.remoteCertificateVerified {
				return errClientCertificateNotVerified
			}
		}

		switch {
		case c.localPSKIdentityHint != nil:
			c.state.localSequenceNumber = 4
		case c.localPSKCallback != nil:
			c.state.localSequenceNumber = 3
		case c.clientAuth > NoClientCert:
			c.state.localSequenceNumber = 6
		default:
			c.state.localSequenceNumber = 5
		}
		c.setLocalEpoch(1)

		if err := c.currFlight.set(flight6); err != nil {
			return err
		}
	}
	return nil
}

func serverFlightHandler(c *Conn) (bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	switch c.currFlight.get() {
	case flight0:
		// Waiting for ClientHello
	case flight2:
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
				handshakeMessage: &handshakeMessageHelloVerifyRequest{
					version: protocolVersion1_2,
					cookie:  c.cookie,
				},
			},
		}, false)

	case flight4:
		extensions := []extension{}
		if c.state.srtpProtectionProfile != 0 {
			extensions = append(extensions, &extensionUseSRTP{
				protectionProfiles: []SRTPProtectionProfile{c.state.srtpProtectionProfile},
			})
		}
		if c.localPSKCallback == nil {
			extensions = append(extensions, []extension{
				&extensionSupportedEllipticCurves{
					ellipticCurves: []namedCurve{namedCurveX25519, namedCurveP256},
				},
				&extensionSupportedPointFormats{
					pointFormats: []ellipticCurvePointFormat{ellipticCurvePointFormatUncompressed},
				},
			}...)
		}

		sequenceNumber := c.state.localSequenceNumber
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
				handshakeMessage: &handshakeMessageServerHello{
					version:           protocolVersion1_2,
					random:            c.state.localRandom,
					cipherSuite:       c.state.cipherSuite,
					compressionMethod: defaultCompressionMethods[0],
					extensions:        extensions,
				}},
		}, false)
		sequenceNumber++

		if c.localPSKCallback == nil {
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
					handshakeMessage: &handshakeMessageCertificate{
						certificate: c.localCertificate,
					}},
			}, false)
			sequenceNumber++

			if len(c.localKeySignature) == 0 {
				serverRandom, err := c.state.localRandom.Marshal()
				if err != nil {
					return false, err
				}
				clientRandom, err := c.state.remoteRandom.Marshal()
				if err != nil {
					return false, err
				}

				signature, err := generateKeySignature(clientRandom, serverRandom, c.localKeypair.publicKey, c.namedCurve, c.localPrivateKey, HashAlgorithmSHA256)
				if err != nil {
					return false, err
				}
				c.localKeySignature = signature
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
					handshakeMessage: &handshakeMessageServerKeyExchange{
						ellipticCurveType:  ellipticCurveTypeNamedCurve,
						namedCurve:         c.namedCurve,
						publicKey:          c.localKeypair.publicKey,
						hashAlgorithm:      HashAlgorithmSHA256,
						signatureAlgorithm: signatureAlgorithmECDSA,
						signature:          c.localKeySignature,
					}},
			}, false)
			sequenceNumber++

			if c.clientAuth > NoClientCert {
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
						handshakeMessage: &handshakeMessageCertificateRequest{
							certificateTypes: []clientCertificateType{clientCertificateTypeRSASign, clientCertificateTypeECDSASign},
							signatureHashAlgorithms: []signatureHashAlgorithm{
								{HashAlgorithmSHA256, signatureAlgorithmRSA},
								{HashAlgorithmSHA384, signatureAlgorithmRSA},
								{HashAlgorithmSHA512, signatureAlgorithmRSA},
								{HashAlgorithmSHA256, signatureAlgorithmECDSA},
								{HashAlgorithmSHA384, signatureAlgorithmECDSA},
								{HashAlgorithmSHA512, signatureAlgorithmECDSA},
							},
						},
					},
				}, false)
				sequenceNumber++
			}
		} else if c.localPSKIdentityHint != nil {
			/* To help the client in selecting which identity to use, the server
			*  can provide a "PSK identity hint" in the ServerKeyExchange message.
			*  If no hint is provided, the ServerKeyExchange message is omitted.
			*
			*  https://tools.ietf.org/html/rfc4279#section-2
			 */
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
					handshakeMessage: &handshakeMessageServerKeyExchange{
						identityHint: c.localPSKIdentityHint,
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
				handshakeMessage: &handshakeMessageServerHelloDone{},
			},
		}, false)
	case flight6:
		c.internalSend(&recordLayer{
			recordLayerHeader: recordLayerHeader{
				sequenceNumber:  c.state.localSequenceNumber,
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
				handshakeCachePullRule{handshakeTypeFinished, true},
			)

			var err error
			c.localVerifyData, err = prfVerifyDataServer(c.state.masterSecret, plainText, c.state.cipherSuite.hashFunc())
			if err != nil {
				return false, err
			}
		}

		c.internalSend(&recordLayer{
			recordLayerHeader: recordLayerHeader{
				epoch:           1,
				sequenceNumber:  0, // sequenceNumber restarts per epoch
				protocolVersion: protocolVersion1_2,
			},
			content: &handshake{
				// sequenceNumber and messageSequence line up, may need to be re-evaluated
				handshakeHeader: handshakeHeader{
					messageSequence: uint16(c.state.localSequenceNumber), // KeyExchange + 1
				},

				handshakeMessage: &handshakeMessageFinished{
					verifyData: c.localVerifyData,
				}},
		}, true)

		// TODO: Better way to end handshake
		c.signalHandshakeComplete()
		return true, nil
	default:
		return false, fmt.Errorf("unhandled flight %s", c.currFlight.get())
	}
	return false, nil
}
