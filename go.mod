module github.com/UNO-SOFT/grpcer

go 1.15

require (
	github.com/goccy/go-json v0.4.11
	github.com/golang/protobuf v1.5.2
	github.com/klauspost/compress v1.11.13
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/tgulacsi/go v0.15.1
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	golang.org/x/net v0.0.0-20210331212208-0fccb6fa2b5c // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210402192133-700132347e07 // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/genproto v0.0.0-20210402141018-6c239bbf2bb1 // indirect
	google.golang.org/grpc v1.36.1
	google.golang.org/protobuf v1.26.0
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
