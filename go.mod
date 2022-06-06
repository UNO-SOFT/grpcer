module github.com/UNO-SOFT/grpcer

go 1.15

require (
	github.com/UNO-SOFT/otel v0.3.1
	github.com/go-logr/logr v1.2.3
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.15.6
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/tgulacsi/go v0.15.1
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.32.0 // indirect
	golang.org/x/net v0.0.0-20220531201128-c960675eff93 // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306
	google.golang.org/genproto v0.0.0-20220602131408-e326c6e8e9c8 // indirect
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.28.0
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
//replace github.com/UNO-SOFT/otel => ../otel
