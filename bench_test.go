// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// run with `go test -bench . -benchmem`

package enc

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"reflect"
	"testing"
)

type Bench struct {
	I   int
	I8  int8
	I16 int16
	I32 int32
	I64 int64
	U   uint
	U8  uint8
	U16 uint16
	U32 uint32
	U64 uint64

	F32 float32
	F64 float64

	P *uintptr
	B bool
	S string
	L []byte
	A [8]int
	T Time

	M map[string][]Bench
}

var benchValue = randomValue(nil, reflect.TypeOf(Bench{}))
var encValue, encGob, encJSON []byte

func init() {
	var buf bytes.Buffer

	Encode(&buf, benchValue)
	encValue = append(encValue, buf.Bytes()...)
	buf.Reset()

	gob.NewEncoder(&buf).Encode(benchValue)
	encGob = append(encGob, buf.Bytes()...)
	buf.Reset()

	json.NewEncoder(&buf).Encode(benchValue)
	encJSON = append(encJSON, buf.Bytes()...)
}

type nilWriter struct{}

func (nilWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (nilWriter) WriteByte(byte) error {
	return nil
}

func (nilWriter) WriteString(s string) (int, error) {
	return len(s), nil
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := Encode(nilWriter{}, benchValue); err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkEncodeJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := json.NewEncoder(nilWriter{}).Encode(benchValue); err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkEncodeGob(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := gob.NewEncoder(nilWriter{}).Encode(benchValue); err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := Decode(bytes.NewReader(encValue), new(Bench)); err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkDecodeJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := json.NewDecoder(bytes.NewReader(encJSON)).Decode(new(Bench)); err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkDecodeGob(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := gob.NewDecoder(bytes.NewReader(encGob)).Decode(new(Bench)); err != nil {
			b.Log(err)
		}
	}
}
