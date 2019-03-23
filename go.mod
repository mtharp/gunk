module github.com/mtharp/gunk

go 1.12

require (
	github.com/gorilla/mux v1.7.0
	github.com/nareix/joy4 v0.0.0-20181022032202-3ddbc8f9d431
	github.com/pions/webrtc v1.2.0
	golang.org/x/crypto v0.0.0-20190219172222-a4c6cb3142f2
	golang.org/x/oauth2 v0.0.0-20190319182350-c85d3e98c914
	golang.org/x/sync v0.0.0-20181221193216-37e7f081c4d4
)

replace github.com/nareix/joy4 => ../../joy4

replace github.com/pions/webrtc => ../../pions/webrtc

replace github.com/pions/rtp => ../../pions/rtp

replace github.com/pions/transport => ../../pions/transport

replace github.com/pions/srtp => ../../pions/srtp
