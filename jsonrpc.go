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
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
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
	jsoniter.NewEncoder(w).Encode(e)
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
	err := jsoniter.NewDecoder(io.TeeReader(r.Body, buf)).Decode(inp)
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
		err := jsoniter.NewDecoder(
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
	buf.Reset()
	jenc := jsoniter.NewEncoder(buf)
	_ = jenc.Encode(inp)
	Log("inp", buf.String())
	ctx := context.Background()
	if u, p, ok := r.BasicAuth(); ok {
		ctx = WithBasicAuth(ctx, u, p)
	}
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	recv, err := h.Call(name, ctx, inp)
	if err != nil {
		Log("call", name, "error", fmt.Sprintf("%#v", err))
		jsonError(w, errors.WithMessage(err, "Call "+name).Error(), statusCodeFromError(err))
		return
	}

	part, err := recv.Recv()
	if err != nil {
		Log("msg", "recv", "error", err)
		jsonError(w, errors.WithMessage(err, "recv").Error(), statusCodeFromError(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	if m := r.URL.Query().Get("merge"); h.MergeStreams && m != "0" || !h.MergeStreams && m == "1" {
		buf.Reset()
		_ = jenc.Encode(part)
		Log("part", limitWidth(buf.Bytes(), 256))
		if err := mergeStreams(w, part, recv, Log); err != nil {
			Log("mergeStreams", "error", err)
		}
		return
	}

	enc := jsoniter.NewEncoder(w)
	for {
		buf.Reset()
		_ = jenc.Encode(part)
		Log("part", limitWidth(buf.Bytes(), 256))
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

func statusCodeFromError(err error) int {
	code := grpc.Code(errors.Cause(err))
	switch code {
	case codes.PermissionDenied, codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.Unknown:
		if desc := grpc.ErrorDesc(err); desc == "bad username or password" {
			return http.StatusUnauthorized
		}
	}
	return http.StatusInternalServerError
}

func limitWidth(b []byte, width int) string {
	if width == 0 {
		width = 1024
	}
	if len(b) <= width {
		return string(b)
	}
	if len(b) <= width-4 {
		return string(b[:width-4]) + " ..."
	}
	n := len(b) - width - 12
	return fmt.Sprintf("%s ...%d... %s", b[:width/2-6], n, b[len(b)-width/2-6:])
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

func SnakeCase(text string) string {
	if text == "" {
		return text
	}
	b := make([]rune, 0, len(text)*2)
	strings.Map(func(r rune) rune {
		if 'A' <= r && r <= 'Z' {
			b = append(b, unicode.ToLower(r), '_')
		} else {
			b = append(b, r)
		}
		return -1
	},
		text)
	return string(b)
}

func init() {
	extra.RegisterFuzzyDecoders()
	SetNoOmit(func(nm string) bool { return strings.HasSuffix(nm, "_Output") })
}
func SetNoOmit(filter func(string) bool) {
	jsoniter.RegisterExtension(&JSNoOmitEmptyExtension{filter: filter})
}

type JSNoOmitEmptyExtension struct {
	jsoniter.DummyExtension
	filter func(string) bool
}

func (no *JSNoOmitEmptyExtension) UpdateStructDescriptor(sd *jsoniter.StructDescriptor) {
	if !no.filter(sd.Type.Name()) {
		return
	}
	for _, binding := range sd.Fields {
		binding.Encoder = nonEmptyEncoder{binding.Encoder}
	}
}

type nonEmptyEncoder struct {
	jsoniter.ValEncoder
}

func (ne nonEmptyEncoder) IsEmpty(ptr unsafe.Pointer) bool { return false }

// vim: set fileencoding=utf-8 noet:
