[![Build Status](https://travis-ci.com/gortc/turn.svg?branch=master)](https://travis-ci.com/gortc/turn)
[![Master status](https://tc.gortc.io/app/rest/builds/buildType:(id:stun_MasterStatus)/statusIcon.svg)](https://tc.gortc.io/project.html?projectId=turn&tab=projectOverview&guest=1)
[![Build status](https://ci.appveyor.com/api/projects/status/bodd3l5hgu1agxpf/branch/master?svg=true)](https://ci.appveyor.com/project/ernado/turn-gvuk2/branch/master)
[![GoDoc](https://godoc.org/github.com/gortc/turn?status.svg)](http://godoc.org/github.com/gortc/turn)
[![codecov](https://codecov.io/gh/gortc/turn/branch/master/graph/badge.svg)](https://codecov.io/gh/gortc/turn)
[![Go Report](https://goreportcard.com/badge/github.com/gortc/turn)](http://goreportcard.com/report/gortc/turn)
[![stability-beta](https://img.shields.io/badge/stability-beta-33bbff.svg)](https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#beta)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fgortc%2Fturn.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fgortc%2Fturn?ref=badge_shield)

# TURN

Package turn implements TURN [[RFC5766](https://tools.ietf.org/html/rfc5766)] Traversal Using Relays around NAT.
Based on [pion/stun](https://github.com/pion/stun) package.
See [gortcd](https://github.com/gortc/gortcd) for TURN server and [stunc](https://github.com/pion/turnc) for TURN client.

## Supported RFCs

- [x] [RFC 5766](https://tools.ietf.org/html/rfc5766) — Traversal Using Relays around NAT
    - [x] UDP transport for client
    - [ ] TCP or TLS transport for client
- [x] [RFC 6156](https://tools.ietf.org/html/rfc6156) — TURN Extension for IPv6
- [x] [RFC 7065](https://tools.ietf.org/html/rfc7065) — TURN URI
- [ ] [RFC 5928](https://tools.ietf.org/html/rfc5928) — TURN Resolution Mechanism [#13](https://github.com/gortc/turn/issues/13)
- [ ] [RFC 6062](https://tools.ietf.org/html/rfc6062) — TURN Extension for TCP Allocations [#14](https://github.com/gortc/turn/issues/14)

# Testing
Client behavior is tested and verified in many ways:
  * End-To-End with long-term credentials
    * **coturn**: The coturn [server](https://github.com/coturn/coturn/wiki/turnserver) (linux)
    * **gortcd**: The [gortcd](https://github.com/gortc/gortcd) server (windows)
  * Bunch of code static checkers (linters)
  * Unit-tests (linux {amd64, **arm**64}, windows}
  * Explicit API backward compatibility [check](https://github.com/gortc/api), see `api` directory (relaxed until v1)

See [TeamCity project](https://tc.gortc.io/project.html?projectId=turn&guest=1) and `e2e` directory
for more information. Also the Wireshark `.pcap` files are available for some of e2e tests in
artifacts for build.

## Benchmarks

```
goos: linux
goarch: amd64
pkg: github.com/gortc/turn
PASS
benchmark                                 iter     time/iter     throughput   bytes alloc        allocs
---------                                 ----     ---------     ----------   -----------        ------
BenchmarkIsChannelData-12           2000000000    1.64 ns/op   6694.29 MB/s        0 B/op   0 allocs/op
BenchmarkChannelData_Encode-12       200000000    9.11 ns/op   1317.35 MB/s        0 B/op   0 allocs/op
BenchmarkChannelData_Decode-12       500000000    3.92 ns/op   3061.45 MB/s        0 B/op   0 allocs/op
BenchmarkChannelNumber/AddTo-12      100000000   12.60 ns/op                       0 B/op   0 allocs/op
BenchmarkChannelNumber/GetFrom-12    200000000    7.23 ns/op                       0 B/op   0 allocs/op
BenchmarkData/AddTo-12               100000000   18.80 ns/op                       0 B/op   0 allocs/op
BenchmarkData/AddToRaw-12            100000000   16.80 ns/op                       0 B/op   0 allocs/op
BenchmarkLifetime/AddTo-12           100000000   13.70 ns/op                       0 B/op   0 allocs/op
BenchmarkLifetime/GetFrom-12         200000000    7.10 ns/op                       0 B/op   0 allocs/op
ok  	github.com/gortc/turn	19.110s
```
