// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package enc

import (
	"bufio"
	"encoding/binary"
	"io"
	"reflect"
)

// Encode marshals a value and writes it to w.
// It panics if the value is of an invalid type.
func Encode(w io.Writer, v interface{}) error {
	return EncodeValue(w, reflect.ValueOf(v))
}

// EncodeValue marshals a reflection value and writes it to w.
// It panics if the value is of an invalid type.
func EncodeValue(w io.Writer, v reflect.Value) (err error) {
	defer func() {
		switch p := recover(); p := p.(type) {
		case nil:
		case noPanic:
			err = p.error
		default:
			panic(p)
		}
	}()

	var e encoder
	if w, ok := w.(writer); ok {
		e.w = w
	} else {
		tmp := bufio.NewWriter(w)
		defer func() { err = tmp.Flush() }()
		w = tmp
	}

	if !v.CanSet() {
		v = reflect.Indirect(v)
	}
	types.get(v.Type()).encode(&e, v)
	return
}

type encoder struct {
	w   writer
	buf [binary.MaxVarintLen64]byte
}

func (e *encoder) encodeInt(i int64) {
	e.write(e.buf[:binary.PutVarint(e.buf[:], i)])
}

func (e *encoder) encodeUint(u uint64) {
	e.write(e.buf[:binary.PutUvarint(e.buf[:], u)])
}

func (e *encoder) write(b []byte) {
	if _, err := e.w.Write(b); err != nil {
		panic(noPanic{err})
	}
}

func (e *encoder) writeByte(b byte) {
	if err := e.w.WriteByte(b); err != nil {
		panic(noPanic{err})
	}
}

func (e *encoder) writeString(s string) {
	if _, err := e.w.WriteString(s); err != nil {
		panic(noPanic{err})
	}
}
