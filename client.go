// Copyright 2017 Tamás Gulácsi
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

// Package grpcer provide helpers for calling UNO-SOFT gRPC server.
package grpcer

import (
	"fmt"
	"log"
	"strings"

	"golang.org/x/net/context"
	errors "golang.org/x/xerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Receiver is an interface for Recv()-ing streamed responses from the server.
type Receiver interface {
	Recv() (interface{}, error)
}

// Client is the client interface for calling a gRPC server.
type Client interface {
	// List the available names
	List() []string
	// Input returns the input struct for the name.
	Input(name string) interface{}
	// Call the named function.
	Call(name string, ctx context.Context, input interface{}, opts ...grpc.CallOption) (Receiver, error)
}

// DialConfig contains the configuration variables.
type DialConfig struct {
	PathPrefix         string
	CAFile             string
	ServerHostOverride string
	Username, Password string
	Log                func(keyvals ...interface{}) error
}

// DialOpts renders the dial options for calling a gRPC server.
//
// * prefix is inserted before the standard request path - if your server serves on different path.
// * caFile is the PEM file with the server's CA.
// * serverHostOverride is to override the CA's host.
func DialOpts(conf DialConfig) ([]grpc.DialOption, error) {
	dialOpts := make([]grpc.DialOption, 0, 6)
	dialOpts = append(dialOpts,
		//lint:ignore SA1019 the UseCompressor API is experimental yet.
		grpc.WithCompressor(grpc.NewGZIPCompressor()),
		//lint:ignore SA1019 the UseCompressor API is experimental yet.
		grpc.WithDecompressor(grpc.NewGZIPDecompressor()))

	if prefix, Log := conf.PathPrefix, conf.Log; prefix != "" || Log != nil {
		if Log == nil {
			Log = func(keyvals ...interface{}) error { return nil }
		}
		dialOpts = append(dialOpts,
			grpc.WithStreamInterceptor(
				func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
					Log("method", method)
					//opts = append(opts, grpc.UseCompressor("gzip"))
					return streamer(ctx, desc, cc, prefix+method, opts...)
				}),
			grpc.WithUnaryInterceptor(
				func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
					Log("method", method)
					//opts = append(opts, grpc.UseCompressor("gzip"))
					return invoker(ctx, prefix+method, req, reply, cc, opts...)
				}),
		)
	}
	if conf.CAFile == "" {
		ba := NewInsecureBasicAuth(conf.Username, conf.Password)
		return append(dialOpts, grpc.WithInsecure(), grpc.WithPerRPCCredentials(ba)), nil
	}
	ba := NewBasicAuth(conf.Username, conf.Password)
	dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(ba))
	log.Printf("dialConf=%+v", conf)
	creds, err := credentials.NewClientTLSFromFile(conf.CAFile, conf.ServerHostOverride)
	if err != nil {
		return dialOpts, errors.Errorf("%q,%q: %w", conf.CAFile, conf.ServerHostOverride, err)
	}
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))

	return dialOpts, nil
}

// Connect to the given endpoint, with the Certificate Authority and hostOverride.
func Connect(endpoint, CAFile, serverHostOverride string) (*grpc.ClientConn, error) {
	var prefix string
	if i := strings.IndexByte(endpoint, '/'); i >= 0 {
		endpoint, prefix = (endpoint)[:i], (endpoint)[i:]
	}
	dc := DialConfig{
		PathPrefix:         prefix,
		CAFile:             CAFile,
		ServerHostOverride: serverHostOverride,
		Log: func(keyvals ...interface{}) error {
			for i := 0; i < len(keyvals); i += 2 {
				keyvals[i] = fmt.Sprintf("%v=", keyvals[i])
			}
			log.Println(keyvals...)
			return nil
		},
	}
	opts, err := DialOpts(dc)
	if err != nil {
		return nil, errors.Errorf("%#v: %w", dc, err)
	}
	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		return nil, errors.Errorf("%s:  %w", endpoint, err)
	}
	return conn, nil
}

// vim: se noet fileencoding=utf-8:
