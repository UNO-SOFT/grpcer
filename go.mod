module github.com/UNO-SOFT/grpcer

go 1.15

require (
	github.com/goccy/go-json v0.4.11
	github.com/golang/protobuf v1.5.0
	github.com/klauspost/compress v1.11.8
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/tgulacsi/go v0.15.1
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210225134936-a50acf3fe073 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/genproto v0.0.0-20210226172003-ab064af71705 // indirect
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.26.0
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
