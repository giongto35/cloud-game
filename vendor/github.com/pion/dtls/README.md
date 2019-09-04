<h1 align="center">
  <br>
  Pion DTLS
  <br>
</h1>
<h4 align="center">A Go implementation of DTLS</h4>
<p align="center">
  <a href="https://pion.ly"><img src="https://img.shields.io/badge/pion-dtls-gray.svg?longCache=true&colorB=brightgreen" alt="Pion DTLS"></a>
  <a href="https://sourcegraph.com/github.com/pion/dtls/pkg/dtls?badge"><img src="https://sourcegraph.com/github.com/pion/dtls/pkg/dtls/-/badge.svg" alt="Sourcegraph Widget"></a>
  <a href="https://pion.ly/slack"><img src="https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen" alt="Slack Widget"></a>
  <br>
  <a href="https://travis-ci.org/pion/dtls"><img src="https://travis-ci.org/pion/dtls.svg?branch=master" alt="Build Status"></a>
  <a href="https://godoc.org/github.com/pion/dtls"><img src="https://godoc.org/github.com/pion/dtls?status.svg" alt="GoDoc"></a>
  <a href="https://codecov.io/gh/pion/dtls"><img src="https://codecov.io/gh/pion/dtls/branch/master/graph/badge.svg" alt="Coverage Status"></a>
  <a href="https://goreportcard.com/report/github.com/pion/dtls"><img src="https://goreportcard.com/badge/github.com/pion/dtls" alt="Go Report Card"></a>
  <a href="https://www.codacy.com/app/Sean-Der/dtls"><img src="https://api.codacy.com/project/badge/Grade/18f4aec384894e6aac0b94effe51961d" alt="Codacy Badge"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
</p>
<br>

Go DTLS 1.2 implementation. The original user is pion-WebRTC, but we would love to see it work for everyone.

A long term goal is a professional security review, and maye inclusion in stdlib.

### Goals/Progress
This will only be targeting DTLS 1.2, and the most modern/common cipher suites.
We would love contributes that fall under the 'Planned Features' and fixing any bugs!

#### Current features
* DTLS 1.2 Client/Server
* Key Exchange via ECDHE(curve25519 and nistp256) and PSK
* Packet loss and re-ordering is handled during handshaking
* Key export ([RFC 5705][rfc5705])
* Serialization and Resumption of sessions

[rfc5705]: https://tools.ietf.org/html/rfc5705

#### Supported ciphers

##### ECDHE
* TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 ([RFC 5289][rfc5289])
* TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 ([RFC 5289][rfc5289])
* TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA ([RFC 8422][rfc8422])
* TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA ([RFC 8422][rfc8422])

##### PSK
* TLS_PSK_WITH_AES_128_CCM_8 ([RFC 6655][rfc6655])
* TLS_PSK_WITH_AES_128_GCM_SHA256 ([RFC 5487][rfc5487])

[rfc5289]: https://tools.ietf.org/html/rfc5289
[rfc8422]: https://tools.ietf.org/html/rfc8422
[rfc6655]: https://tools.ietf.org/html/rfc6655
[rfc5487]: https://tools.ietf.org/html/rfc5487

#### Planned Features
* Extended master secret support (RFC7627)
* Chacha20Poly1305

#### Excluded Features
* DTLS 1.0
* Renegotiation
* Compression

### Using

#### Pion DTLS
For a DTLS 1.2 Server that listens on 127.0.0.1:4444
```sh
go run examples/listen/main.go
```

For a DTLS 1.2 Client that connects to 127.0.0.1:4444
```sh
go run examples/dial/main.go
```

#### OpenSSL
Pion DTLS can connect to itself and OpenSSL.
```
  // Generate a certificate
  openssl ecparam -out key.pem -name prime256v1 -genkey
  openssl req -new -sha256 -key key.pem -out server.csr
  openssl x509 -req -sha256 -days 365 -in server.csr -signkey key.pem -out cert.pem

  // Use with examples/dial/main.go
  openssl s_server -dtls1_2 -cert cert.pem -key key.pem -accept 4444

  // Use with examples/listen/main.go
  openssl s_client -dtls1_2 -connect 127.0.0.1:4444 -debug -cert cert.pem -key key.pem
```

### Using with PSK
Pion DTLS also comes with examples that do key exchange via PSK


#### Pion DTLS
```sh
go run examples/listen-psk/main.go
```

```sh
go run examples/dial-psk/main.go
```

#### OpenSSL
```
  // Use with examples/dial-psk/main.go
  openssl s_server -dtls1_2 -accept 4444 -nocert -psk abc123 -cipher PSK-AES128-CCM8

  // Use with examples/listen-psk/main.go
  openssl s_client -dtls1_2 -connect 127.0.0.1:4444 -psk abc123 -cipher PSK-AES128-CCM8
```

### Contributing
Check out the **[contributing wiki](https://github.com/pion/webrtc/wiki/Contributing)** to join the group of amazing people making this project possible:

* [Sean DuBois](https://github.com/Sean-Der) - *Original Author*
* [Michiel De Backker](https://github.com/backkem) - *Public API*
* [Chris Hiszpanski](https://github.com/thinkski) - *Support Signature Algorithms Extension*
* [Iñigo Garcia Olaizola](https://github.com/igolaizola) - *Serialization & resumption, cert verification*
* [Daniele Sluijters](https://github.com/daenney) - *AES-CCM support*
* [Jin Lei](https://github.com/jinleileiking) - *Logging*
* [Hugo Arregui](https://github.com/hugoArregui)
* [Lander Noterman](https://github.com/LanderN)
* [Aleksandr Razumov](https://github.com/ernado) - *Fuzzing*

### License
MIT License - see [LICENSE](LICENSE) for full text
