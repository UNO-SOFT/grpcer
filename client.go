// Copyright 2017, 2020 Tamás Gulácsi
//
// SPDX-License-Identifier: Apache-2.0

// Package grpcer provide helpers for calling UNO-SOFT gRPC server.
package grpcer

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/UNO-SOFT/otel"
	"github.com/UNO-SOFT/otel/gtrace"

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
	Log                            func(keyvals ...interface{}) error
	PathPrefix                     string
	CAFile                         string
	ServerHostOverride             string
	Username, Password             string
	AllowInsecurePasswordTransport bool
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

	if prefix, Log := conf.PathPrefix, conf.Log; Log != nil {
		provider, err := otel.LogTraceProvider(Log)
		if err != nil {
			return nil, err
		}
		providerOpt := gtrace.WithTracerProvider(provider)
		dialOpts = append(dialOpts,
			grpc.WithChainStreamInterceptor(
				func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
					Log("method", method)
					return streamer(ctx, desc, cc, prefix+method, opts...)
				},
				gtrace.StreamClientInterceptor(providerOpt),
			),
			grpc.WithChainUnaryInterceptor(
				func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
					Log("method", method)
					return invoker(ctx, prefix+method, req, reply, cc, opts...)
				},
				gtrace.UnaryClientInterceptor(providerOpt),
			),
		)
	}
	if conf.CAFile == "" {
		if conf.AllowInsecurePasswordTransport {
			ba := NewInsecureBasicAuth(conf.Username, conf.Password)
			dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(ba))
		}
		return append(dialOpts, grpc.WithInsecure()), nil
	}
	ba := NewBasicAuth(conf.Username, conf.Password)
	dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(ba))
	log.Printf("dialConf=%+v", conf)
	creds, err := credentials.NewClientTLSFromFile(conf.CAFile, conf.ServerHostOverride)
	if err != nil {
		return dialOpts, fmt.Errorf("%q,%q: %w", conf.CAFile, conf.ServerHostOverride, err)
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
		return nil, fmt.Errorf("%#v: %w", dc, err)
	}
	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("%s:  %w", endpoint, err)
	}
	return conn, nil
}

// vim: se noet fileencoding=utf-8:
