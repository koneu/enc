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

	F32 float32
	F64 float64

	C64  complex64
	C128 complex128

	P *uintptr
	B bool
	S string
	L []byte
	A [8]int
	T Time

	M map[string][]*Test
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
	v := randomValue(t, reflect.TypeOf(Test{}))
	testEquals(t, new(interface{}), &v)
}

func TestChan(t *testing.T) {
	v := *(randomValue(t, reflect.TypeOf([]int{})).(*[]int))

	c := make(chan int)
	go func() {
		for _, i := range v {
			c <- i
		}
		close(c)
	}()

	var buf bytes.Buffer

	if err := Encode(&buf, c); err != nil {
		t.Error(err)
	}
	c = nil
	if err := Decode(&buf, &c); err != nil {
		t.Error(err)
	}

	if cap(c) != len(v) {
		t.Error("decoded data does not match encoded data")
	}
	for _, i := range v {
		if i != <-c {
			t.Error("decoded data does not match encoded data")
		}
	}
}

func TestNilChan(t *testing.T) {
	v := make(chan int)
	testEquals(t, new(chan int), &v)
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
