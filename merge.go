// Copyright 2019, 2020 Tamás Gulácsi
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
	"fmt"
	//json "encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	json "github.com/json-iterator/go"
)

var errNewField = errors.New("new field")

func mergeStreams(w io.Writer, first interface{}, recv interface {
	Recv() (interface{}, error)
},
	Log func(...interface{}) error,
) error {
	if Log == nil {
		Log = func(...interface{}) error { return nil }
	}

	slice, notSlice := sliceFields(first)
	if len(slice) == 0 {
		var err error
		part := first
		enc := json.NewEncoder(w)
		for {
			if err := enc.Encode(part); err != nil {
				Log("encode", part, "error", err)
				return fmt.Errorf("encode part: %w", err)
			}

			part, err = recv.Recv()
			if err != nil {
				if err != io.EOF {
					Log("msg", "recv", "error", err)
				}
				break
			}
		}
		Log("slice", len(slice))
		return nil
	}

	names := make(map[string]bool, len(slice)+len(notSlice))

	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()
	jenc := json.NewEncoder(buf)

	//Log("slices", slice)
	w.Write([]byte("{"))
	for _, f := range notSlice {
		buf.Reset()
		jenc.Encode(f.JSONName)
		w.Write(bytes.TrimSpace(buf.Bytes()))

		w.Write([]byte{':'})
		buf.Reset()
		jenc.Encode(f.Value)
		w.Write(bytes.TrimSpace(buf.Bytes()))
		w.Write([]byte{','})

		names[f.Name] = false
	}
	buf.Reset()
	jenc.Encode(slice[0].JSONName)
	w.Write(bytes.TrimSpace(buf.Bytes()))
	w.Write([]byte(":"))

	buf.Reset()
	jenc.Encode(slice[0].Value)
	w.Write(bytes.TrimSuffix(bytes.TrimSpace(buf.Bytes()), []byte{']'}))

	names[slice[0].Name] = true

	files := make(map[string]*os.File, len(slice)-1)
	for _, f := range slice[1:] {
		fh, err := ioutil.TempFile("", "merge-"+f.Name+"-")
		if err != nil {
			Log("tempFile", f.Name, "error", err)
			return fmt.Errorf("%s: %w", f.Name, err)
		}
		os.Remove(fh.Name())
		Log("fn", fh.Name())
		defer fh.Close()
		files[f.Name] = fh
		buf.Reset()
		jenc.Encode(f.JSONName)
		fh.Write(bytes.TrimSpace(buf.Bytes()))
		io.WriteString(fh, ":[")

		buf.Reset()
		jenc.Encode(f.Value)
		fh.Write(trimSqBrs(buf.Bytes()))

		names[f.Name] = true
	}

	var part interface{}
	var err error
	for {
		part, err = recv.Recv()
		if err != nil {
			if err != io.EOF {
				Log("msg", "recv", "error", err)
			}
			break
		}
		buf.Reset()
		jenc.Encode(part)
		Log("part", limitWidth(buf.Bytes(), 256))

		S, nS := sliceFields(part)
		for _, f := range S {
			if isSlice, ok := names[f.Name]; !(ok && isSlice) {
				err = fmt.Errorf("%s: %w", f.Name, errNewField)
			}
		}
		for _, f := range nS {
			if isSlice, ok := names[f.Name]; !(ok && !isSlice) {
				err = fmt.Errorf("%s: %w", f.Name, errNewField)
			}
		}
		if err != nil {
			Log("error", err)
			//TODO(tgulacsi): close the merge and send as is
			return err
		}

		if S[0].Name == slice[0].Name {
			w.Write([]byte{','})
			buf.Reset()
			jenc.Encode(S[0].Value)
			w.Write(trimSqBrs(buf.Bytes()))
			S = S[1:]
		}
		for _, f := range S {
			fh := files[f.Name]
			if _, err := fh.Write([]byte{','}); err != nil {
				Log("write", fh.Name(), "error", err)
			}
			buf.Reset()
			jenc.Encode(f.Value)
			fh.Write(trimSqBrs(buf.Bytes()))
		}
	}
	w.Write([]byte("]"))

	for _, fh := range files {
		if err := fh.Sync(); err != nil {
			Log("Sync", fh.Name(), "error", err)
		}
		if _, err := fh.Seek(0, 0); err != nil {
			Log("Seek", fh.Name(), "error", err)
			continue
		}
		w.Write([]byte{','})
		io.Copy(w, fh)
		w.Write([]byte{']'})
	}
	w.Write([]byte{'}', '\n'})
	return nil
}

type field struct {
	Name     string
	JSONName string
	Value    interface{}
}

func sliceFields(part interface{}) (slice, notSlice []field) {
	rv := reflect.ValueOf(part)
	t := rv.Type()
	if t.Kind() == reflect.Ptr {
		rv = rv.Elem()
		t = rv.Type()
	}
	n := t.NumField()
	for i := 0; i < n; i++ {
		f := rv.Field(i)
		tf := t.Field(i)
		fld := field{Name: tf.Name, Value: f.Interface()}
		fld.JSONName = tf.Tag.Get("json")
		if i := strings.IndexByte(fld.JSONName, ','); i >= 0 {
			fld.JSONName = fld.JSONName[:i]
		}
		if fld.JSONName == "" {
			fld.JSONName = fld.Name
		}

		if f.Type().Kind() != reflect.Slice {
			notSlice = append(notSlice, fld)
			continue
		}
		if f.IsNil() {
			continue
		}
		slice = append(slice, fld)
	}
	return slice, notSlice
}
func trimSqBrs(b []byte) []byte {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return b
	}
	if b[0] == '[' {
		b = b[1:]
	}
	if len(b) == 0 {
		return b
	}
	if b[len(b)-1] == ']' {
		b = b[:len(b)-1]
	}
	return b
}
