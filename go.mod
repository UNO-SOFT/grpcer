module github.com/UNO-SOFT/grpcer

go 1.21

toolchain go1.21.0

require (
	github.com/UNO-SOFT/otel v0.5.1
	github.com/UNO-SOFT/zlog v0.7.4
	github.com/klauspost/compress v1.16.7
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/tgulacsi/go v0.24.3
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	golang.org/x/exp v0.0.0-20230713183714-613f0c0eb8a1
	golang.org/x/time v0.3.0
	google.golang.org/grpc v1.57.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zerologr v1.2.3 // indirect
	github.com/godror/godror v0.24.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/rs/zerolog v1.29.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.42.0 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/oauth2 v0.7.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/term v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/genproto v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230807174057-1744710a1577 // indirect
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
//replace github.com/UNO-SOFT/otel => ../otel
