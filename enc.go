// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package enc implements a very compact binary encoding for Go types.
// No type information is stored.
package enc

import (
	"encoding"
	"io"
	"reflect"
)

var (
	bytesType       = reflect.TypeOf([]byte{})
	marshalerType   = reflect.TypeOf(new(encoding.BinaryMarshaler)).Elem()
	unmarshalerType = reflect.TypeOf(new(encoding.BinaryUnmarshaler)).Elem()
)

// A TypeError indicates that an invalid type was passed to De- or Encode.
type TypeError struct {
	T reflect.Type
}

func (t TypeError) Error() string {
	return "enc: invalid type: " + t.T.String()
}

type writer interface {
	io.Writer
	io.ByteWriter
	WriteString(string) (int, error)
}

type reader interface {
	io.Reader
	io.ByteScanner
}

type noPanic struct {
	error
}
