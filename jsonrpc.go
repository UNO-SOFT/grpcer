// Copyright 2017, 2021 Tamás Gulácsi
//
// SPDX-License-Identifier: Apache-2.0

package grpcer

import (
	"bytes"
	"context"
	"encoding"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	json "github.com/goccy/go-json"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	DefaultTimeout = 5 * time.Minute
	MaxLogWidth    = 1 << 10

	ErrNotFound = errors.New("not found")
)

type RequestInfo interface {
	Name() string
}

type JSONHandler struct {
	Client
	Log          func(...interface{}) error
	Timeout      time.Duration
	MergeStreams bool
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

var msDecConf = mapstructure.DecoderConfig{
	Squash:           true,
	WeaklyTypedInput: true,
	DecodeHook: func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t == reflect.TypeOf(time.Time{}) {
			return time.ParseInLocation(time.RFC3339, data.(string), time.Local)
		}
		if t.Kind() != reflect.Ptr {
			return data, nil
		}
		tv := reflect.New(t.Elem()).Interface()
		if tu, ok := tv.(encoding.TextUnmarshaler); ok {
			err := tu.UnmarshalText([]byte(data.(string)))
			return tu, err
		}
		if t.Kind() == reflect.Struct {
			if _, ok := t.FieldByName("Time"); ok {
				return time.ParseInLocation(time.RFC3339, data.(string), time.Local)
			}
		}
		return data, nil
	},
}

func (h JSONHandler) DecodeRequest(ctx context.Context, r *http.Request) (RequestInfo, interface{}, error) {
	Log := h.Log
	if Log == nil {
		Log = func(...interface{}) error { return nil }
	}

	request := requestInfo{name: path.Base(r.URL.Path)}
	Log("name", request.name)
	inp := h.Input(request.name)
	if inp == nil {
		return request, nil, fmt.Errorf("no unmarshaler for %q: %w", request.name, ErrNotFound)
	}
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	buf.Reset()
	err := json.NewDecoder(io.TeeReader(r.Body, buf)).Decode(inp)
	Log("body", buf.String(), "error", err)
	if err == nil {
		return request, inp, nil
	}
	origErr := fmt.Errorf("%s: %w", buf.String(), err)
	Log("got", buf.String(), "inp", inp, "error", origErr)
	m := mapPool.Get().(map[string]interface{})
	defer func() {
		for k := range m {
			delete(m, k)
		}
		mapPool.Put(m)
	}()
	if err = json.NewDecoder(
		io.MultiReader(bytes.NewReader(buf.Bytes()), r.Body),
	).Decode(&m); err != nil {
		return request, nil, fmt.Errorf("decode %s: %w (was: %+v)", buf.String(), err, origErr)
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
	decConf := msDecConf
	decConf.Result = inp
	if dec, err := mapstructure.NewDecoder(&decConf); err != nil {
		return request, inp, err
	} else if err = dec.Decode(m); err != nil {
		return request, inp, fmt.Errorf("weakdecode(%#v): %w (was: %+v)", m, err, origErr)
	}
	return request, inp, nil
}

type requestInfo struct {
	name string
}

func (info requestInfo) Name() string { return info.name }

func (h JSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Log := h.Log
	if Log == nil {
		Log = func(...interface{}) error { return nil }
	}

	ctx := r.Context()
	request, inp, err := h.DecodeRequest(ctx, r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := request.Name()

	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()
	buf.Reset()
	jenc := json.NewEncoder(buf)
	_ = jenc.Encode(inp)
	{
		u, p, ok := r.BasicAuth()
		Log("inp", buf.String(), "username", u)
		if ok {
			ctx = WithBasicAuth(ctx, u, p)
		}
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
	dl, _ := ctx.Deadline()
	Log("call", name, "deadline", dl)

	recv, err := h.Call(name, ctx, inp)
	if err != nil {
		Log("call", name, "error", fmt.Sprintf("%#v", err))
		jsonError(w, fmt.Sprintf("Call %s: %s", name, err), statusCodeFromError(err))
		return
	}

	part, err := recv.Recv()
	if err != nil {
		Log("msg", "recv", "error", err)
		jsonError(w, fmt.Sprintf("recv: %s", err), statusCodeFromError(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	if m := r.URL.Query().Get("merge"); h.MergeStreams && m != "0" || !h.MergeStreams && m == "1" {
		buf.Reset()
		_ = jenc.Encode(part)
		Log("part", limitWidth(buf.Bytes(), MaxLogWidth))
		if err := mergeStreams(w, part, recv, Log); err != nil {
			Log("mergeStreams", "error", err)
		}
		return
	}

	enc := json.NewEncoder(w)
	for {
		buf.Reset()
		_ = jenc.Encode(part)
		Log("part", limitWidth(buf.Bytes(), MaxLogWidth))
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
	st := status.Convert(errors.Unwrap(err))
	switch st.Code() {
	case codes.PermissionDenied, codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.Unknown:
		if desc := st.Message(); desc == "bad username or password" {
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
	_ = strings.Map(func(r rune) rune {
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

// vim: set fileencoding=utf-8 noet:
