// Copyright 2017, 2022 Tamás Gulácsi
//
// SPDX-License-Identifier: Apache-2.0

package grpcer

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/klauspost/compress/gzhttp"
	"github.com/mitchellh/mapstructure"
	"github.com/tgulacsi/go/iohlp"

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
	Client       `json:"-"`
	*slog.Logger `json:"-"`
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

const debugDecodeHook = false

var msDecConf = mapstructure.DecoderConfig{
	Squash:           true,
	WeaklyTypedInput: true,
	DecodeHook: func(f reflect.Type, t reflect.Type, data interface{}) (res interface{}, err error) {
		if debugDecodeHook {
			fmt.Printf("\nf:%+v t:%+v data:%#v\n\n", f, t, data)
			defer func() {
				fmt.Printf("res=%#v err=%+v\n", res, err)
			}()
		}
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t == reflect.TypeOf(time.Time{}) {
			//fmt.Println("t1")
			return time.ParseInLocation(time.RFC3339, data.(string), time.Local)
		}
		//fmt.Printf("t=%v\n", t.Kind())
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
			if t == reflect.TypeOf(time.Time{}) {
				//fmt.Println("t2")
				return time.ParseInLocation(time.RFC3339, data.(string), time.Local)
			}
		}

		s := data.(string)
		switch t.Kind() {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			return strings.TrimLeft(s, " 0"), nil

		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
			s = strings.TrimSpace(s)
			neg := strings.HasPrefix(s, "-")
			if neg {
				s = s[1:]
			}
			s = strings.TrimLeft(s, " 0")
			if neg {
				s = "-" + s
			}
			return s, nil
		}

		//fmt.Printf("t=%#v (%v)\n", t, t.Kind())
		tv := reflect.New(t)
		if tu, ok := tv.Interface().(encoding.TextUnmarshaler); ok {
			//fmt.Println("tu\n")
			err := tu.UnmarshalText([]byte(data.(string)))
			return tu, err
		}
		if t.Kind() == reflect.Struct {
			//fmt.Println("struct")
			if _, ok := t.FieldByName("Time"); ok {
				tim, err := time.ParseInLocation(time.RFC3339, data.(string), time.Local)
				if err != nil {
					return data, err
				}
				tv.FieldByName("Time").Set(reflect.ValueOf(tim))
				return tv.Interface(), nil
			}
		}
		return data, nil
	},
}

func (h JSONHandler) DecodeRequest(ctx context.Context, r *http.Request) (RequestInfo, interface{}, error) {
	logger := h.Logger

	request := requestInfo{name: path.Base(r.URL.Path)}
	logger.Info("DecodeRequest", "name", request.name)
	inp := h.Input(request.name)
	if inp == nil {
		return request, nil, fmt.Errorf("no unmarshaler for %q: %w", request.name, ErrNotFound)
	}
	sr, err := iohlp.MakeSectionReader(r.Body, 1<<20)
	if err != nil {
		return request, nil, err
	}

	if err = json.NewDecoder(io.NewSectionReader(sr, 0, sr.Size())).Decode(inp); err == nil {
		return request, inp, nil
	}
	logger.Error("decode", "body", sr.Read, "error", err)
	b, _ := ReadHeadTail(io.NewSectionReader(sr, 0, sr.Size()), 1024)
	origErr := fmt.Errorf("%s: %w", string(b), err)
	m := mapPool.Get().(map[string]interface{})
	defer func() {
		for k := range m {
			delete(m, k)
		}
		mapPool.Put(m)
	}()
	if err = json.NewDecoder(sr).Decode(&m); err != nil {
		return request, nil, fmt.Errorf("decode %s: %w (was: %+v)", string(b), err, origErr)
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
	decConf := msDecConf
	decConf.Result = inp
	if dec, err := mapstructure.NewDecoder(&decConf); err != nil {
		return request, inp, fmt.Errorf("mapstructure.NewDecoder: %w (was: %+v)", err, origErr)
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
	gzhttp.GzipHandler(http.HandlerFunc(h.serveHTTP)).ServeHTTP(w, r)
}
func (h JSONHandler) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r != nil && r.Body != nil {
		defer r.Body.Close()
	}
	ctx := r.Context()
	logger := h.Logger
	request, inp, err := h.DecodeRequest(ctx, r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.Body.Close()
	name := request.Name()

	ht := iohlp.HeadTailKeeper{Limit: MaxLogWidth / 2}
	jenc := json.NewEncoder(&ht)
	{
		_ = jenc.Encode(inp)
		u, p, ok := r.BasicAuth()
		logger.Info("basicAuth", "inp", ht.String(), "username", u)
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
	logger.Info("call", "name", name, "deadline", dl)

	recv, err := h.Call(name, ctx, inp)
	if err != nil {
		logger.Error("call", name, "error", err)
		jsonError(w, fmt.Sprintf("Call %s: %s", name, err), statusCodeFromError(err))
		return
	}

	part, err := recv.Recv()
	if err != nil {
		logger.Error("recv", "error", err)
		jsonError(w, fmt.Sprintf("recv: %s", err), statusCodeFromError(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	if m := r.URL.Query().Get("merge"); h.MergeStreams && m != "0" || !h.MergeStreams && m == "1" {
		ht.Reset()
		_ = jenc.Encode(part)
		logger.Debug("merge", "part", ht.String())
		if err := mergeStreams(w, part, recv, logger); err != nil {
			logger.Error("mergeStreams", "error", err)
		}
		return
	}

	enc := json.NewEncoder(w)
	for {
		ht.Reset()
		_ = jenc.Encode(part)
		logger.Debug("cycle", "part", ht.String())
		if err := enc.Encode(part); err != nil {
			logger.Error("encode", part, "error", err)
			return
		}

		part, err = recv.Recv()
		if err != nil {
			if err != io.EOF {
				logger.Error("msg", "recv", "error", err)
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

func ReadHeadTail(sr *io.SectionReader, maxSize int64) ([]byte, error) {
	size := sr.Size()
	if size <= maxSize {
		p := make([]byte, int(size))
		_, err := sr.ReadAt(p, 0)
		return p, err
	}
	p := make([]byte, int(maxSize))
	_, firstErr := sr.ReadAt(p[:len(p)/2], 0)
	_, secondErr := sr.ReadAt(p[len(p)/2:], size-int64(len(p)/2))
	return p, errors.Join(firstErr, secondErr)
}

// vim: set fileencoding=utf-8 noet:
