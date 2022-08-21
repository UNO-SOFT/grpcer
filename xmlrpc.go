// Copyright 2017, 2022 Tamás Gulácsi
//
// SPDX-License-Identifier: Apache-2.0

package grpcer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-logr/logr"
	"github.com/klauspost/compress/gzhttp"
	"github.com/mitchellh/mapstructure"
	"github.com/tgulacsi/go-xmlrpc"
)

type XMLRPCHandler struct {
	Client
	logr.Logger
	Timeout time.Duration
}

func (h XMLRPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gzhttp.GzipHandler(http.HandlerFunc(h.serveHTTP)).ServeHTTP(w, r)
}
func (h XMLRPCHandler) serveHTTP(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r.Context(), h.Logger)
	name, params, err := xmlrpc.Unmarshal(r.Body)
	logger.Info("unmarshal", "name", name, "params", params, "error", err)
	if err != nil {
		http.Error(w, fmt.Sprintf("ERROR unmarshaling: %v", err), http.StatusBadRequest)
		return
	}
	inp := h.Input(name)
	if inp == nil {
		http.Error(w, fmt.Sprintf("No unmarshaler for %q.", name), http.StatusNotFound)
		return
	}

	if len(params) != 1 {
		http.Error(w, fmt.Sprintf("Wanted 1 struct param, got %d.", len(params)), http.StatusBadRequest)
		return
	}
	m, ok := params[0].(map[string]interface{})
	if !ok {
		http.Error(w, fmt.Sprintf("Wanted struct, got %T", params[0]), http.StatusBadRequest)
		return
	}

	// mapstruct
	for k, v := range m {
		if s, ok := v.(string); ok && s == "" {
			delete(m, k)
			continue
		}
		f, _ := utf8.DecodeRune([]byte(k))
		if unicode.IsLower(f) {
			m[CamelCase(k)] = v
		}
	}
	if err := mapstructure.WeakDecode(m, inp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("decoded", "inp", inp)

	ctx := r.Context()
	if u, p, ok := r.BasicAuth(); ok {
		ctx = WithBasicAuth(ctx, u, p)
	}
	if _, ok := ctx.Deadline(); !ok {
		timeout := h.Timeout
		if timeout == 0 {
			timeout = DefaultTimeout
		}
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
	}
	recv, err := h.Call(name, ctx, inp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	part, err := recv.Recv()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	parts := []interface{}{nil}[:0]
	for {
		parts = append(parts, part)
		if part, err = recv.Recv(); err != nil {
			if err != io.EOF {
				logger.Error(err, "recv")
				parts = parts[:1]
				parts[0] = xmlrpc.Fault{Code: 111, Message: err.Error()}
			}
			break
		}
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(200)
	if len(parts) == 1 {
		err = xmlrpc.Marshal(w, "", parts[0])
	} else {
		err = xmlrpc.Marshal(w, "", parts)
	}
	if err != nil {
		logger.Error(err, "marshal")
	}
}

func getLogger(ctx context.Context, logger logr.Logger) logr.Logger {
	if lgr, err := logr.FromContext(ctx); err == nil {
		return lgr
	}
	return logger
}

// vim: set fileencoding=utf-8 noet:
