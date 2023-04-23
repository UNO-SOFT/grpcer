module github.com/UNO-SOFT/grpcer

go 1.19

require (
	github.com/UNO-SOFT/otel v0.5.0
	github.com/go-logr/logr v1.2.4
	github.com/klauspost/compress v1.16.5
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/tgulacsi/go v0.15.1
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	golang.org/x/exp v0.0.0-20230420155640-133eef4313cb
	golang.org/x/time v0.3.0
	google.golang.org/grpc v1.54.0
	google.golang.org/protobuf v1.30.0
)

require (
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/godror/godror v0.24.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.40.0 // indirect
	go.opentelemetry.io/otel v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.37.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.14.0 // indirect
	go.opentelemetry.io/otel/metric v0.37.0 // indirect
	go.opentelemetry.io/otel/sdk v1.14.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.14.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/oauth2 v0.5.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
//replace github.com/UNO-SOFT/otel => ../otel
