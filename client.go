// Copyright 2017, 2022 Tamás Gulácsi
//
// SPDX-License-Identifier: Apache-2.0

// Package grpcer provide helpers for calling UNO-SOFT gRPC server.
package grpcer

import (
	"context"
	"fmt"
	"strings"

	"github.com/UNO-SOFT/otel"
	"github.com/UNO-SOFT/otel/gtrace"

	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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
	*slog.Logger
	PathPrefix                     string
	CAFile                         string
	ServerHostOverride             string
	Username, Password             string
	ServiceName, ServiceVersion    string
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

	if prefix, logger := conf.PathPrefix, conf.Logger; logger.Enabled(context.Background(), slog.LevelInfo) {
		serviceName, serviceVersion := conf.ServiceName, conf.ServiceVersion
		if serviceName == "" {
			serviceName = conf.Username + "@" + conf.ServerHostOverride + conf.PathPrefix
		}
		tp, _, _, err := otel.LogTraceProvider(
			slog.NewLogLogger(logger.Handler(), slog.LevelInfo),
			serviceName, serviceVersion,
		)
		if err != nil {
			return nil, err
		}
		providerOpt := gtrace.WithTracerProvider(tp)
		propOpt := gtrace.WithPropagators(otel.HTTPPropagators)
		gzipOpt := grpc.UseCompressor("gzip")
		dialOpts = append(dialOpts,
			grpc.WithChainStreamInterceptor(
				func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
					logger.Info("chain", "method", method)
					return streamer(ctx, desc, cc, prefix+method, append(opts, gzipOpt)...)
				},
				gtrace.StreamClientInterceptor(providerOpt, propOpt),
			),
			grpc.WithChainUnaryInterceptor(
				func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
					logger.Info("unary", "method", method)
					return invoker(ctx, prefix+method, req, reply, cc, append(opts, gzipOpt)...)
				},
				gtrace.UnaryClientInterceptor(providerOpt, propOpt),
			),
		)
	}
	if conf.CAFile == "" {
		if conf.AllowInsecurePasswordTransport {
			ba := NewInsecureBasicAuth(conf.Username, conf.Password)
			dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(ba))
		}
		return append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials())), nil
	}
	ba := NewBasicAuth(conf.Username, conf.Password)
	dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(ba))
	conf.Info("dial", "config", conf)
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
