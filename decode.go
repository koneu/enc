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

// Decode reads data from r and unmarshals it.
// It panics if the value is of an invalid type.
func Decode(r io.Reader, v interface{}) error {
	return DecodeValue(r, reflect.ValueOf(v))
}

// DecodeValue reads data from r and unmarshals it.
// It panics if the value is of an invalid type.
func DecodeValue(r io.Reader, v reflect.Value) (err error) {
	defer func() {
		switch p := recover(); p := p.(type) {
		case nil:
		case noPanic:
			err = p.error
		default:
			panic(p)
		}
	}()

	var d decoder
	if r, ok := r.(reader); ok {
		d.r = r
	} else {
		d.r = bufio.NewReader(r)
	}

	if !v.CanSet() {
		v = reflect.Indirect(v)
	}
	types.get(v.Type()).decode(&d, v)
	return
}

type decoder struct {
	r reader
}

func (d *decoder) decodeInt() int64 {
	ret, err := binary.ReadVarint(d.r)
	if err != nil {
		panic(noPanic{err})
	}
	return ret
}

func (d *decoder) decodeUint() uint64 {
	ret, err := binary.ReadUvarint(d.r)
	if err != nil {
		panic(noPanic{err})
	}
	return ret
}

func (d *decoder) read(size uint64) []byte {
	ret := make([]byte, size)
	if _, err := io.ReadFull(d.r, ret); err != nil {
		if err == io.EOF {
			panic(noPanic{io.ErrUnexpectedEOF})
		}
		panic(noPanic{err})
	}
	return ret
}

func (d *decoder) readByte() byte {
	ret, err := d.r.ReadByte()
	if err == nil {
		return ret
	}
	panic(noPanic{err})
}

func (d *decoder) unreadByte() {
	if err := d.r.UnreadByte(); err != nil {
		panic(noPanic{err})
	}
}
