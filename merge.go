// Copyright 2019, 2021 Tamás Gulácsi
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
	"github.com/klauspost/compress/zstd"
)

var (
	errNewField  = errors.New("new field")
	errWrongType = errors.New("wrong type")
)

func mergeStreams(w io.Writer, first interface{}, recv interface{ Recv() (interface{}, error) }, Log func(...interface{}) error) error {
	if Log == nil {
		Log = func(...interface{}) error { return nil }
	}

	slice, notSlice := SliceFields(first, "json")
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
		jenc.Encode(f.TagName)
		w.Write(bytes.TrimSpace(buf.Bytes()))

		w.Write([]byte{':'})
		buf.Reset()
		jenc.Encode(f.Value)
		w.Write(bytes.TrimSpace(buf.Bytes()))
		w.Write([]byte{','})

		names[f.Name] = false
	}
	buf.Reset()
	jenc.Encode(slice[0].TagName)
	w.Write(bytes.TrimSpace(buf.Bytes()))
	w.Write([]byte(":"))

	buf.Reset()
	jenc.Encode(slice[0].Value)
	w.Write(bytes.TrimSuffix(bytes.TrimSpace(buf.Bytes()), []byte{']'}))

	names[slice[0].Name] = true
	files := make(map[string]*TempFile, len(slice)-1)
	openFile := func(f Field) error {
		fh, err := NewTempFile("", "merge-"+f.Name+"-*.json.zst")
		if err != nil {
			Log("tempFile", f.Name, "error", err)
			return fmt.Errorf("%s: %w", f.Name, err)
		}
		files[f.Name] = fh
		buf.Reset()
		jenc.Encode(f.TagName)
		fh.Write(bytes.TrimSpace(buf.Bytes()))
		io.WriteString(fh, ":[")

		buf.Reset()
		jenc.Encode(f.Value)
		fh.Write(trimSqBrs(buf.Bytes()))

		names[f.Name] = true
		return nil
	}
	defer func() {
		for nm, fh := range files {
			fh.Close()
			delete(files, nm)
		}
	}()

	for _, f := range slice[1:] {
		if err := openFile(f); err != nil {
			return err
		}
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

		S, nS := SliceFields(part, "json")
		for _, f := range S {
			if isSlice, ok := names[f.Name]; !ok {
				if err = openFile(f); err != nil {
					break
				}
				//err = fmt.Errorf("%s: %w", f.Name, errNewField)
				//break
			} else if !isSlice {
				err = fmt.Errorf("%s not slice: %w", f.Name, errWrongType)
				break
			}
		}
		for _, f := range nS {
			if isSlice, ok := names[f.Name]; !ok {
				err = fmt.Errorf("%s: %w", f.Name, errNewField)
				break
			} else if isSlice {
				err = fmt.Errorf("%s slice: %w", f.Name, errWrongType)
				break
			}
		}
		if len(S) == 0 {
			break
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
				if Log != nil {
					Log("msg", "write", "error", err)
				}
			}
			buf.Reset()
			jenc.Encode(f.Value)
			fh.Write(trimSqBrs(buf.Bytes()))
		}
	}
	w.Write([]byte("]"))

	for nm, fh := range files {
		rc, err := fh.GetReader()
		if err != nil {
			Log("msg", "GetReader", "name", nm, "error", err)
			continue
		}
		w.Write([]byte{','})
		io.Copy(w, rc)
		rc.Close()
		w.Write([]byte{']'})
		fh.Close()
		delete(files, nm)
	}
	w.Write([]byte{'}', '\n'})
	return nil
}

type Field struct {
	Name    string
	TagName string
	Value   interface{}
}

func SliceFields(part interface{}, tagName string) (slice, notSlice []Field) {
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
		fld := Field{Name: tf.Name, Value: f.Interface(), TagName: tf.Name}
		if tagName != "" {
			if fld.TagName = tf.Tag.Get(tagName); fld.TagName == "" {
				fld.TagName = fld.Name
			} else {
				if i := strings.IndexByte(fld.TagName, ','); i >= 0 {
					fld.TagName = fld.TagName[:i]
				}
				if fld.TagName == "-" { // Skip field
					continue
				}
			}
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

type TempFile struct {
	io.WriteCloser
	file *os.File
}

// NewTempFile creates a new compressed tempfile, that can be read back.
func NewTempFile(dir, name string) (*TempFile, error) {
	fh, err := ioutil.TempFile(dir, name)
	if err != nil {
		return nil, err
	}
	os.Remove(fh.Name())
	zw, err := zstd.NewWriter(fh, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		fh.Close()
		return nil, err
	}
	return &TempFile{WriteCloser: zw, file: fh}, nil
}
func (f *TempFile) Close() error {
	zw, file := f.WriteCloser, f.file
	f.WriteCloser, f.file = nil, nil
	var firstErr error
	if zw != nil {
		firstErr = zw.Close()
	}
	if file != nil {
		if err := file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		os.Remove(file.Name())
	}
	return firstErr
}

// GetReader finishes the writing of the temp file, and returns an io.ReadCloser for reading it back.
func (f *TempFile) GetReader() (io.ReadCloser, error) {
	zw := f.WriteCloser
	f.WriteCloser = nil
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if _, err := f.file.Seek(0, 0); err != nil {
		f.Close()
		return nil, err
	}
	zr, err := zstd.NewReader(f.file)
	if err != nil {
		f.Close()
		return nil, err
	}
	return struct {
		io.Reader
		io.Closer
	}{zr, closerFunc(func() error { zr.Close(); return f.file.Close() })}, nil
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }
