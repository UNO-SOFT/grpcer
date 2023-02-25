module github.com/UNO-SOFT/grpcer

go 1.15

require (
	github.com/UNO-SOFT/otel v0.4.0
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/go-logr/logr v1.2.3
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.15.15
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/tgulacsi/go v0.15.1
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.36.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.13.0 // indirect
	golang.org/x/time v0.3.0
	google.golang.org/genproto v0.0.0-20230223222841-637eb2293923 // indirect
	google.golang.org/grpc v1.53.0
	google.golang.org/protobuf v1.28.1
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
//replace github.com/UNO-SOFT/otel => ../otel
