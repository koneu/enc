// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package enc

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

var rd = rand.New(rand.NewSource(rand.Int63()))

type Test struct {
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

	P *int
	B bool
	S string
	L []byte
	A [8]int
	T Time

	M map[string][]Test
}

type Time struct{ time.Time }

func (Time) Generate(*rand.Rand, int) reflect.Value {
	return reflect.ValueOf(Time{time.Now()})
}

func TestTime(t *testing.T) {
	n := time.Now()
	testEquals(t, &n, new(time.Time))
}

func TestSlice(t *testing.T) {
	testRandom(t, reflect.TypeOf([][]byte{}))
}

func TestMap(t *testing.T) {
	testRandom(t, reflect.TypeOf(map[string][123]int{}))
}

func TestStruct(t *testing.T) {
	testRandom(t, reflect.TypeOf(Test{}))
}

func TestInterface(t *testing.T) {
	a := randomValue(t, reflect.TypeOf(Test{}))
	var b interface{} = new(Test)
	testEquals(t, &a, &b)
}

func TestNilInterface(t *testing.T) {
	var a, b interface{}
	testEquals(t, &a, &b)
}

func testEquals(t *testing.T, a, b interface{}) {
	var buf bytes.Buffer

	if err := Encode(&buf, a); err != nil {
		t.Error(err)
	}
	if err := Decode(&buf, b); err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Error("decoded data does not match encoded data")
	}
}

func testRandom(t *testing.T, typ reflect.Type) {
	testEquals(t, randomValue(t, typ), reflect.New(typ).Interface())
}

func randomValue(t testing.TB, typ reflect.Type) interface{} {
	a, ok := quick.Value(typ, rd)
	if !ok {
		t.Error("could not create random data")
	}

	return a.Addr().Interface()
}
