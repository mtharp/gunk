module github.com/mtharp/gunk

go 1.12

require (
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/golang/mock v1.3.1 // indirect
	github.com/golang/protobuf v1.3.1 // indirect
	github.com/gorilla/mux v1.7.2
	github.com/gorilla/websocket v1.4.0
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.4.0+incompatible
	github.com/kr/pretty v0.1.0
	github.com/lib/pq v1.1.1 // indirect
	github.com/lucas-clemente/quic-go v0.11.1 // indirect
	github.com/nareix/joy4 v0.0.0-20181022032202-3ddbc8f9d431
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pion/ice v0.3.1 // indirect
	github.com/pion/rtp v1.1.2
	github.com/pion/sdp v1.3.0
	github.com/pion/webrtc/v2 v2.0.16
	github.com/pkg/errors v0.8.1
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shopspring/decimal v0.0.0-20180709203117-cd690d0c9e24 // indirect
	golang.org/x/crypto v0.0.0-20190513172903-22d7a77e9e5f
	golang.org/x/net v0.0.0-20190522155817-f3200d17e092 // indirect
	golang.org/x/oauth2 v0.0.0-20190523182746-aaccbc9213b0
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20190526052359-791d8a0f4d09 // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/appengine v1.6.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
	layeh.com/gopus v0.0.0-20161224163843-0ebf989153aa
)

replace github.com/nareix/joy4 => github.com/mtharp/joy4 v0.0.0-20190323221533-84311da2c824
