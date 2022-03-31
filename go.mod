module github.com/UNO-SOFT/grpcer

go 1.15

require (
	github.com/UNO-SOFT/otel v0.2.0
	github.com/go-logr/logr v1.2.3
	github.com/klauspost/compress v1.13.4
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/tgulacsi/go v0.15.1
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	golang.org/x/time v0.0.0-20210611083556-38a9dc6acbc6
	google.golang.org/grpc v1.43.0
	google.golang.org/protobuf v1.27.1
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
//replace github.com/UNO-SOFT/otel => ../otel
