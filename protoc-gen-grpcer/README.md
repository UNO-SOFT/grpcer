# protoc-gen-grpcer
Generate [https://godoc.org/github.com/UNO-SOFT/grpcer#Client](grpcer.Client) into `myrpc.go`
from `myrpc.proto`.

# Install

	go get github.com/UNO-SOFT/grpcer/protoc-gen-grpcer


# Usage

	protoc -I $GOPATH/src --grpcer_out=package=pkgname:/dest/dir $GOPATH/src/unosoft.hu/ws/bruno/pb/dealer/dealer.proto

Will generate `dealer.grpcer.go` under `/dest/dir`, with `package pkgname`.
