module eaglesong.dev/gunk

go 1.14

replace github.com/nareix/joy4 => ../joy4

replace eaglesong.dev/hls => ../hls

require (
	eaglesong.dev/hls v1.1.1-0.20200426043048-f48cdd63f891
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/kr/pretty v0.2.0
	github.com/lib/pq v1.3.0 // indirect
	github.com/nareix/joy4 v0.0.0-20181022032202-3ddbc8f9d431
	github.com/pion/rtp v1.4.0
	github.com/pion/sdp/v2 v2.3.5
	github.com/pion/webrtc/v2 v2.2.4
	github.com/pkg/errors v0.9.1
	github.com/shopspring/decimal v0.0.0-20200227202807-02e2044944cc // indirect
	golang.org/x/crypto v0.0.0-20200323165209-0ec3e9974c59
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	layeh.com/gopus v0.0.0-20161224163843-0ebf989153aa
)
