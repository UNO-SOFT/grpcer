package grpcer_test

import (
	"strings"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/tgulacsi/oracall/custom"
)

func TestDateTime(t *testing.T) {
	type testStruct struct {
		T   time.Time
		TP  *time.Time
		DT  custom.DateTime
		DTP *custom.DateTime
	}

	now := time.Date(2006, 1, 2, 15, 4, 5, 6, time.UTC)
	x := testStruct{
		TP:  &now,
		DTP: &custom.DateTime{Time: now},
	}

	var w strings.Builder
	err := jsoniter.NewEncoder(&w).Encode(x)
	if err != nil {
		t.Fatalf("encode %#v: %+v", x, err)
	}
	t.Log(w.String())
	s := w.String()

	var y testStruct
	if err = jsoniter.NewDecoder(strings.NewReader(s)).Decode(&y); err != nil {
		t.Fatal(err)
	}
	t.Log(y)

	y = testStruct{}
	if err = jsoniter.NewDecoder(strings.NewReader(
		`{"DT":"2006-01-02 16:04"}`,
	)).Decode(&y); err != nil {
		t.Error(err)
	}
}
