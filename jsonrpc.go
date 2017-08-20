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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type JSONHandler struct {
	Client
	MergeStreams bool
	Log          func(...interface{}) error
}

func jsonError(w http.ResponseWriter, errMsg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	if code == 0 {
		code = http.StatusInternalServerError
	}
	w.WriteHeader(code)
	e := struct {
		Error string
	}{Error: errMsg}
	json.NewEncoder(w).Encode(e)
}

func (h JSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Log := h.Log
	if Log == nil {
		Log = func(...interface{}) error { return nil }
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	inp := h.Input(name)
	if inp == nil {
		nm := path.Base(name)
		if inp = h.Input(nm); inp == nil {
			jsonError(w, errors.Errorf("No unmarshaler for %q.", name).Error(), http.StatusNotFound)
			return
		}
		name = nm
	}
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	buf.Reset()
	err := json.NewDecoder(io.TeeReader(r.Body, buf)).Decode(inp)
	Log("body", buf.String())
	if err != nil {
		Log("got", buf.String(), "inp", inp, "error", err)
		m := mapPool.Get().(map[string]interface{})
		defer func() {
			for k := range m {
				delete(m, k)
			}
			mapPool.Put(m)
		}()
		err := json.NewDecoder(
			io.MultiReader(bytes.NewReader(buf.Bytes()), r.Body),
		).Decode(&m)
		if err != nil {
			jsonError(w, errors.Wrap(err, "decode "+buf.String()).Error(), http.StatusBadRequest)
			return
		}
		buf.Reset()

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
			jsonError(w, errors.Wrapf(err, "WeakDecode(%#v)", m).Error(), http.StatusBadRequest)
			return
		}
	}
	Log("inp", fmt.Sprintf("%#v", inp))
	ctx := context.Background()
	if u, p, ok := r.BasicAuth(); ok {
		ctx = WithBasicAuth(ctx, u, p)
	}
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	recv, err := h.Call(name, ctx, inp)
	if err != nil {
		Log("call", name, "error", fmt.Sprintf("%#v", err))
		var status int
		desc := grpc.ErrorDesc(err)
		code := grpc.Code(err)
		switch code {
		case codes.Unknown, codes.PermissionDenied, codes.Unauthenticated:
			if desc == "bad username or password" {
				status = http.StatusUnauthorized
			}
		}
		jsonError(w, errors.WithMessage(err, "Call "+name).Error(), status)
		return
	}

	part, err := recv.Recv()
	if err != nil {
		Log("msg", "recv", "error", err)
		jsonError(w, errors.WithMessage(err, "recv").Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	if h.MergeStreams {
		mergeStreams(w, part, recv, Log)
		return
	}

	enc := json.NewEncoder(w)
	for {
		if err := enc.Encode(part); err != nil {
			Log("encode", part, "error", err)
			return
		}

		part, err = recv.Recv()
		if err != nil {
			if err != io.EOF {
				Log("msg", "recv", "error", err)
			}
			break
		}
	}
}

var mapPool = sync.Pool{New: func() interface{} { return make(map[string]interface{}, 16) }}
var bufPool = sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 4096)) }}

var digitUnder = strings.NewReplacer(
	"_0", "__0",
	"_1", "__1",
	"_2", "__2",
	"_3", "__3",
	"_4", "__4",
	"_5", "__5",
	"_6", "__6",
	"_7", "__7",
	"_8", "__8",
	"_9", "__9",
)

func CamelCase(text string) string {
	if text == "" {
		return text
	}
	var prefix string
	if text[0] == '*' {
		prefix, text = "*", text[1:]
	}

	text = digitUnder.Replace(text)
	var last rune
	return prefix + strings.Map(func(r rune) rune {
		defer func() { last = r }()
		if r == '_' {
			if last != '_' {
				return -1
			}
			return '_'
		}
		if last == 0 || last == '_' || '0' <= last && last <= '9' {
			return unicode.ToUpper(r)
		}
		return unicode.ToLower(r)
	},
		text,
	)
}

// vim: set fileencoding=utf-8 noet:
