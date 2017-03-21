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

package grpcer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mitchellh/mapstructure"
	"github.com/tgulacsi/go-xmlrpc"
)

type XMLRPCHandler struct {
	Client
	Log func(...interface{}) error
}

func (h XMLRPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Log := h.Log
	if Log == nil {
		Log = func(...interface{}) error { return nil }
	}
	name, params, fault, err := xmlrpc.Unmarshal(r.Body)
	Log("name", name, "params", params, "fault", fault, "error", err)
	if err != nil {
		http.Error(w, fmt.Sprintf("ERROR unmarshaling: %v", err), http.StatusBadRequest)
		return
	}
	if fault != nil {
		http.Error(w, fmt.Sprintf("ERROR got fault %s", fault), http.StatusBadRequest)
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
	Log("inp", inp)

	ctx := context.Background()
	if u, p, ok := r.BasicAuth(); ok {
		ctx = WithBasicAuth(ctx, u, p)
	}
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
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
				Log("msg", "recv", "error", err)
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
		Log("msg", "marshal", "error", err)
	}
}

// vim: set fileencoding=utf-8 noet:
