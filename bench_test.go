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

var benchValue = randomValue(nil, reflect.TypeOf(Test{}))
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
		if err := Decode(bytes.NewReader(encValue), new(Test)); err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkDecodeJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := json.NewDecoder(bytes.NewReader(encJSON)).Decode(new(Test)); err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkDecodeGob(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := gob.NewDecoder(bytes.NewReader(encGob)).Decode(new(Test)); err != nil {
			b.Log(err)
		}
	}
}
