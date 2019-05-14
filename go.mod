module github.com/mtharp/gunk

go 1.12

require (
	cloud.google.com/go v0.39.0 // indirect
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/golang/mock v1.3.1 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/google/pprof v0.0.0-20190502144155-8358a9778bd1 // indirect
	github.com/gorilla/mux v1.7.1
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.4.0+incompatible
	github.com/kr/pretty v0.1.0 // indirect
	github.com/kr/pty v1.1.4 // indirect
	github.com/lib/pq v1.1.1 // indirect
	github.com/lucas-clemente/quic-go v0.11.1 // indirect
	github.com/nareix/joy4 v0.0.0-20181022032202-3ddbc8f9d431
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pion/webrtc/v2 v2.0.14
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shopspring/decimal v0.0.0-20180709203117-cd690d0c9e24 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	golang.org/x/crypto v0.0.0-20190513172903-22d7a77e9e5f
	golang.org/x/exp v0.0.0-20190510132918-efd6b22b2522 // indirect
	golang.org/x/image v0.0.0-20190507092727-e4e5bf290fec // indirect
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422 // indirect
	golang.org/x/mobile v0.0.0-20190509164839-32b2708ab171 // indirect
	golang.org/x/net v0.0.0-20190509222800-a4d6f7feada5 // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20190509141414-a5b02f93d862 // indirect
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	golang.org/x/tools v0.0.0-20190513233021-7d589f28aaf4 // indirect
	google.golang.org/appengine v1.5.0 // indirect
	google.golang.org/genproto v0.0.0-20190513181449-d00d292a067c // indirect
	google.golang.org/grpc v1.20.1 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
	honnef.co/go/tools v0.0.0-20190418001031-e561f6794a2a // indirect
	layeh.com/gopus v0.0.0-20161224163843-0ebf989153aa
)

replace github.com/nareix/joy4 => github.com/mtharp/joy4 v0.0.0-20190323221533-84311da2c824

replace github.com/pions/srtp => ./_srtp
