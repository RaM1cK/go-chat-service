module spotify-chat

go 1.26.4

require (
	github.com/RaM1cK/protos v0.0.14
	github.com/gocql/gocql v1.7.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/nats-io/nats.go v1.53.1
	github.com/pressly/goose/v3 v3.27.3
	github.com/scylladb/gocqlx/v3 v3.0.4
	github.com/zishang520/socket.io/servers/socket/v3 v3.0.5
	github.com/zishang520/socket.io/v3 v3.0.5
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/andybalholm/brotli v1.2.4 // indirect
	github.com/dunglas/httpsfv v1.1.1 // indirect
	github.com/gookit/color v1.6.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.62.0 // indirect
	github.com/quic-go/webtransport-go v0.13.0 // indirect
	github.com/scylladb/go-reflectx v1.0.1 // indirect
	github.com/sethvargo/go-retry v0.4.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xo/terminfo v1.2.0 // indirect
	github.com/zishang520/socket.io/parsers/engine/v3 v3.0.5 // indirect
	github.com/zishang520/socket.io/parsers/socket/v3 v3.0.5 // indirect
	github.com/zishang520/socket.io/servers/engine/v3 v3.0.5 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260720211330-0afa2a65878a // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)

// Use the latest version of scylladb/gocql; check for updates at https://github.com/scylladb/gocql/releases
replace github.com/gocql/gocql => github.com/scylladb/gocql v1.16.0
