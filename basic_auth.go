// Copyright 2016 Tamás Gulácsi
//
// SPDX-License-Identifier: Apache-2.0

package grpcer

import (
	"context"

	"google.golang.org/grpc/credentials"
)

type contextKey string

// BasicAuthKey is the context key for the Basic Auth.
const BasicAuthKey = contextKey("authorization-basic")

// WithBasicAuth returns a context prepared with the given username and password.
func WithBasicAuth(ctx context.Context, username, password string) context.Context {
	return context.WithValue(ctx, BasicAuthKey, username+":"+password)
}

var _ = credentials.PerRPCCredentials(basicAuthCreds{})

type basicAuthCreds struct {
	up       string
	insecure bool
}

// NewBasicAuth returns a PerRPCCredentials with the username and password.
func NewBasicAuth(username, password string) credentials.PerRPCCredentials {
	return basicAuthCreds{up: username + ":" + password}
}

// NewInsecureBasicAuth returns an INSECURE (not requiring secure transport) PerRPCCredentials with the username and password.
func NewInsecureBasicAuth(username, password string) credentials.PerRPCCredentials {
	return basicAuthCreds{up: username + ":" + password, insecure: true}
}

// RequireTransportSecurity returns true - Basic Auth is unsecure in itself.
func (ba basicAuthCreds) RequireTransportSecurity() bool { return !ba.insecure }

// GetRequestMetadata extracts the authorization data from the context.
func (ba basicAuthCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	var up string
	if upI := ctx.Value(BasicAuthKey); upI != nil {
		up = upI.(string)
	}
	if up == "" {
		up = ba.up
	}
	return map[string]string{"authorization": up}, nil
}

// vim: se noet fileencoding=utf-8:
