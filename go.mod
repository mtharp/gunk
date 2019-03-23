module github.com/mtharp/gunk

go 1.12

require (
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/gorilla/mux v1.7.0
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.3.0+incompatible
	github.com/lib/pq v1.0.0 // indirect
	github.com/nareix/joy4 v0.0.0-20181022032202-3ddbc8f9d431
	github.com/pions/webrtc v1.2.0
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shopspring/decimal v0.0.0-20180709203117-cd690d0c9e24 // indirect
	golang.org/x/crypto v0.0.0-20190219172222-a4c6cb3142f2
	golang.org/x/oauth2 v0.0.0-20190319182350-c85d3e98c914
	golang.org/x/sync v0.0.0-20181221193216-37e7f081c4d4
)

replace github.com/nareix/joy4 => ../../joy4

replace github.com/pions/webrtc => ../../pions/webrtc

replace github.com/pions/rtp => ../../pions/rtp

replace github.com/pions/transport => ../../pions/transport

replace github.com/pions/srtp => ../../pions/srtp
