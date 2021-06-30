module github.com/UNO-SOFT/grpcer

go 1.15

require (
	github.com/goccy/go-json v0.7.3
	github.com/klauspost/compress v1.13.1
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/tgulacsi/go v0.15.1
	github.com/tgulacsi/go-xmlrpc v0.2.2
	github.com/tgulacsi/oracall v0.19.0
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	google.golang.org/genproto v0.0.0-20210629200056-84d6f6074151 // indirect
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
)

//replace github.com/tgulacsi/oracall => ../../tgulacsi/oracall
