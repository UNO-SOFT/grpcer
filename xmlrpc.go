// Copyright 2017, 2022 Tamás Gulácsi
//
// SPDX-License-Identifier: Apache-2.0

package grpcer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/klauspost/compress/gzhttp"
	"github.com/mitchellh/mapstructure"
	"github.com/tgulacsi/go-xmlrpc"
)

type XMLRPCHandler struct {
	Client
	*slog.Logger
	GetLogger func(ctx context.Context) *slog.Logger
	Timeout   time.Duration
}

func (h XMLRPCHandler) getLogger(ctx context.Context) *slog.Logger {
	if h.GetLogger != nil {
		if lgr := h.GetLogger(ctx); lgr != nil {
			return lgr
		}
	}
	return h.Logger
}

func (h XMLRPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gzhttp.GzipHandler(http.HandlerFunc(h.serveHTTP)).ServeHTTP(w, r)
}
func (h XMLRPCHandler) serveHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := h.getLogger(ctx)
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
			if !errors.Is(err, io.EOF) {
				logger.Error("recv", "error", err)
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
		logger.Error("marshal", "error", err)
	}
}

// vim: set fileencoding=utf-8 noet:
