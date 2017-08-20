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
	"io"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMerge(t *testing.T) {
	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	Log := func(keyvals ...interface{}) error {
		t.Log(keyvals...)
		return nil
	}
	for tN, tC := range map[string]struct {
		Input []interface{}
		Want  string
	}{
		"noSlice": {
			Input: toIntf([]struct {
				A int
				B string
			}{{A: 1, B: "x"}}),
			Want: `{"A":1,"B":"x"}` + "\n",
		},
		"onlyOneSlice": {
			Input: toIntf([]struct {
				A []string
			}{{A: []string{"1"}}, {A: []string{"2"}}, {A: []string{"3"}}}),
			Want: `{"A":["1","2","3"]}` + "\n",
		},
	} {
		buf.Reset()
		recv := &receiver{parts: tC.Input}
		first, _ := recv.Recv()
		mergeStreams(buf, first, recv, Log)
		if d := cmp.Diff(buf.String(), tC.Want); d != "" {
			t.Error(tN+":", d)
		}
	}
}

func toIntf(someSlice interface{}) []interface{} {
	rv := reflect.ValueOf(someSlice)
	res := make([]interface{}, rv.Len())
	for i := range res {
		res[i] = rv.Index(i).Interface()
	}
	return res
}

type receiver struct {
	parts []interface{}
}

func (r *receiver) Recv() (interface{}, error) {
	if len(r.parts) == 0 {
		return nil, io.EOF
	}
	p := r.parts[0]
	r.parts = r.parts[1:]
	return p, nil
}

func TestTrimWriter(t *testing.T) {
	buf := bufPool.Get().(*bytes.Buffer)
	bufPool.Put(buf)
	for tN, tC := range map[string]struct {
		Input, Prefix, Suffix, Want string
	}{
		"\\n": {Input: "\"A\"\n", Prefix: "", Suffix: "\n", Want: "\"A\""},
		"[]":  {Input: "[1]", Prefix: "[", Suffix: "]", Want: "1"},
	} {
		buf.Reset()
		tw := newTrimWriter(buf, tC.Prefix, tC.Suffix)
		if _, err := io.WriteString(tw, tC.Input); err != nil {
			t.Error(tN+":", err)
		}
		if err := tw.Close(); err != nil {
			t.Error(tN+":", err)
		}
	}
}
