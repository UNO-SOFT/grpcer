module github.com/UNO-SOFT/grpcer

go 1.15

require (
	github.com/goccy/go-json v0.7.0
	github.com/klauspost/compress v1.13.0
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/tgulacsi/go v0.15.1
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	golang.org/x/net v0.0.0-20210610132358-84b48f89b13b // indirect
	golang.org/x/sys v0.0.0-20210611083646-a4fc73990273 // indirect
	google.golang.org/genproto v0.0.0-20210611144927-798beca9d670 // indirect
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
