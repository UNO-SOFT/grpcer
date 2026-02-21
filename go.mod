module github.com/UNO-SOFT/grpcer

go 1.25.4

require (
	github.com/UNO-SOFT/zlog v0.8.6
	github.com/klauspost/compress v1.18.0
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/tgulacsi/go v0.28.2
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	github.com/valyala/quicktemplate v1.7.0
	google.golang.org/grpc v1.79.1
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/UNO-SOFT/w3ctrace v0.0.1 // indirect
	github.com/dgryski/go-linebreak v0.0.0-20180812204043-d8f37254e7d3 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/godror/godror v0.40.2 // indirect
	github.com/godror/knownpb v0.1.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.39.0 // indirect
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
//replace github.com/UNO-SOFT/otel => ../otel
