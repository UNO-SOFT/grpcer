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
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"reflect"
)

func mergeStreams(w io.Writer, first interface{}, recv interface {
	Recv() (interface{}, error)
},
	Log func(...interface{}) error,
) {
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
		return
	}

	w.Write([]byte("{"))
	enc := json.NewEncoder(w)
	for _, f := range notSlice {
		tw := newTrimWriter(w, "", "\n")
		json.NewEncoder(tw).Encode(f.Name)
		tw.Close()
		w.Write([]byte{':'})
		enc.Encode(f.Value)
		w.Write([]byte{','})
	}
	tw := newTrimWriter(w, "", "\n")
	json.NewEncoder(tw).Encode(slice[0].Name)
	tw.Close()
	w.Write([]byte(":["))
	tw = newTrimWriter(w, "[", "]")
	json.NewEncoder(tw).Encode(slice[0].Value)
	tw.Close()

	files := make(map[string]*os.File, len(slice)-1)
	for _, f := range slice[1:] {
		fh, err := ioutil.TempFile("", "merge-"+f.Name+"-")
		if err != nil {
			Log("tempFile", f.Name, "error", err)
			return
		}
		os.Remove(fh.Name())
		defer fh.Close()
		files[f.Name] = fh
		tw := newTrimWriter(fh, "", "\n")
		json.NewEncoder(tw).Encode(f.Name)
		tw.Close()
		io.WriteString(fh, ":[")
		tw = newTrimWriter(fh, "[", "]")
		json.NewEncoder(tw).Encode(f.Value)
		tw.Close()
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

		S, _ := sliceFields(part)
		for i, j := 0, 0; i < len(S) && j < len(slice); {
			if S[i].Name == slice[j].Name {
				i++
				j++
				continue
			}
			j++
		}
		for _, f := range S {
			fh := files[f.Name]
			fh.Write([]byte{','})
			tw := newTrimWriter(fh, "[", "]")
			json.NewEncoder(fh).Encode(f.Value)
			tw.Close()
		}
	}

	var notFirst bool
	for _, fh := range files {
		if _, err := fh.Seek(0, 0); err != nil {
			Log("Seek", fh.Name(), "error", err)
			continue
		}
		if notFirst {
			w.Write([]byte{','})
		} else {
			notFirst = true
		}
		io.Copy(w, fh)
		w.Write([]byte{']'})
	}
	w.Write([]byte{'}'})
}

type field struct {
	Name  string
	Value interface{}
}

func sliceFields(part interface{}) (slice, notSlice []field) {
	rv := reflect.ValueOf(part)
	t := rv.Type()
	n := t.NumField()
	for i := 0; i < n; i++ {
		f := rv.Field(i)
		if f.Type().Kind() == reflect.Slice {
			slice = append(slice, field{Name: t.Field(i).Name, Value: f.Interface()})
		} else {
			notSlice = append(notSlice, field{Name: t.Field(i).Name, Value: f.Interface()})
		}
	}
	return slice, notSlice
}

type trimWriter struct {
	w              io.Writer
	prefix, suffix string
	buf            []byte
}

func newTrimWriter(w io.Writer, prefix, suffix string) *trimWriter {
	return &trimWriter{w: w, prefix: prefix, suffix: suffix}
}
func (tw *trimWriter) Write(p []byte) (int, error) {
	if len(p) <= len(tw.prefix) {
		tw.prefix = tw.prefix[len(p):]
		return len(p), nil
	}
	if len(tw.buf) > 0 && len(tw.buf) >= len(tw.suffix) {
		if _, err := tw.w.Write(tw.buf); err != nil {
			return 0, err
		}
		tw.buf = tw.buf[:0]
	}
	if len(p) <= len(tw.suffix) {
		tw.buf = append(tw.buf, p...)
		return len(p), nil
	}
	i := len(p) - len(tw.suffix) + len(tw.buf)
	tw.buf = append(tw.buf, p[i:]...)
	_, err := tw.w.Write(p[:i])
	return len(p), err
}
func (tw *trimWriter) Close() error {
	if tw.suffix == string(tw.buf) {
		return nil
	}
	_, err := tw.w.Write(tw.buf)
	return err
}
