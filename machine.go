// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package enc

import (
	"encoding"
	"math"
	"reflect"
	"sync"
)

var types = _types{m: make(map[reflect.Type]machine)}

type _types struct {
	sync.RWMutex
	m map[reflect.Type]machine
}

func (g *_types) get(t reflect.Type) machine {
	g.RLock()
	ret, ok := g.m[t]
	g.RUnlock()
	if ok {
		return ret
	}
	return g.register(t)
}

// register walks a type and generates a machine for it.
// Once a machine has been generated, the type can be encoded without the need to walk its definition.
func (g *_types) register(t reflect.Type) (ret machine) {
	// special care must be taken for recursive types
	g.Lock()
	if r, ok := g.m[t]; ok {
		g.Unlock()
		return r
	}
	lock := &recurseMachine{c: make(chan machine, 1)}
	g.m[t] = lock
	g.Unlock()

	defer func() {
		g.Lock()
		g.m[t] = ret
		g.Unlock()

		lock.c <- ret
	}()

bigswitch:
	switch t.Kind() {
	case reflect.Bool:
		return boolMachine{}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intMachine{}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintMachine{}
	case reflect.Float32, reflect.Float64:
		return floatMachine{}
	case reflect.Complex64, reflect.Complex128:
		return complexMachine{}
	case reflect.Array:
		ret = &arrayMachine{t.Len(), g.get(t.Elem())}
	case reflect.Chan:
		e, s := t.Elem(), reflect.SliceOf(t.Elem())
		return &chanMachine{reflect.Zero(t), e, s, t, g.get(e), g.get(s)}
	case reflect.Interface:
		return &interfaceMachine{reflect.Zero(t)}
	case reflect.Map:
		k, v := t.Key(), t.Elem()
		return &mapMachine{t, k, v, g.get(k), g.get(v)}
	case reflect.Ptr:
		return &ptrMachine{reflect.Zero(t), t.Elem(), g.get(t.Elem())}
	case reflect.Slice:
		if t == bytesType {
			return bytesMachine{}
		}
		return &sliceMachine{t, g.get(t.Elem())}
	case reflect.String:
		return stringMachine{}
	case reflect.Struct:
		r := make(structMachine, t.NumField())
		for i := range r {
			f := t.Field(i)
			if f.PkgPath != "" && !f.Anonymous {
				break bigswitch
			}
			r[i] = g.get(f.Type)
		}
		ret = r
	}

	// support BinaryMarshaler as a last resort
	if ret == nil {
		p := reflect.PtrTo(t)
		r := marshalerMachine{p.Implements(marshalerType), p.Implements(unmarshalerType)}
		if !(r.e || t.Implements(marshalerType)) && !(r.d || t.Implements(unmarshalerType)) {
			panic(TypeError{t})
		}
		ret = &r
	}

	// encode zero values as a single 0 byte
	if t.Comparable() {
		z := reflect.Zero(t)
		ret = &compareMachine{z, z.Interface(), ret}
	}

	return
}

func decodeZero(d *decoder, v, z reflect.Value) bool {
	if d.readByte() == 0 {
		v.Set(z)
		return true
	}
	d.unreadByte()
	return false
}

type machine interface {
	encode(*encoder, reflect.Value)
	decode(*decoder, reflect.Value)
}

// block any action until types.generate is done
type recurseMachine struct {
	o sync.Once
	c chan machine
	m machine
}

func (m *recurseMachine) encode(e *encoder, v reflect.Value) {
	m.o.Do(func() {
		m.m = <-m.c
	})
	m.m.encode(e, v)
}

func (m *recurseMachine) decode(d *decoder, v reflect.Value) {
	m.o.Do(func() {
		m.m = <-m.c
	})
	m.m.decode(d, v)
}

type compareMachine struct {
	zv reflect.Value
	z  interface{}
	m  machine
}

func (m *compareMachine) encode(e *encoder, v reflect.Value) {
	// too bad reflect.Value lacks an Equals() method
	if m.z == v.Interface() {
		e.writeByte(0)
		return
	}
	m.m.encode(e, v)
}

func (m *compareMachine) decode(d *decoder, v reflect.Value) {
	if !decodeZero(d, v, m.zv) {
		m.m.decode(d, v)
	}
}

type boolMachine struct{}

func (boolMachine) encode(e *encoder, v reflect.Value) {
	if v.Bool() {
		e.writeByte(1)
	} else {
		e.writeByte(0)
	}
}

func (boolMachine) decode(d *decoder, v reflect.Value) {
	if d.readByte() == 1 {
		v.SetBool(true)
	} else {
		v.SetBool(false)
	}
}

type intMachine struct{}

func (intMachine) encode(e *encoder, v reflect.Value) { e.encodeInt(v.Int()) }
func (intMachine) decode(d *decoder, v reflect.Value) { v.SetInt(d.decodeInt()) }

type uintMachine struct{}

func (uintMachine) encode(e *encoder, v reflect.Value) { e.encodeUint(v.Uint()) }
func (uintMachine) decode(d *decoder, v reflect.Value) { v.SetUint(d.decodeUint()) }

type floatMachine struct{}

func (floatMachine) encode(e *encoder, v reflect.Value) {
	e.encodeUint(math.Float64bits(v.Float()))
}

func (floatMachine) decode(d *decoder, v reflect.Value) {
	v.SetFloat(math.Float64frombits(d.decodeUint()))
}

type complexMachine struct{}

func (complexMachine) encode(e *encoder, v reflect.Value) {
	c := v.Complex()
	e.encodeUint(math.Float64bits(real(c)))
	e.encodeUint(math.Float64bits(imag(c)))
}

func (complexMachine) decode(d *decoder, v reflect.Value) {
	v.SetComplex(complex(
		math.Float64frombits(d.decodeUint()),
		math.Float64frombits(d.decodeUint()),
	))
}

type arrayMachine struct {
	l int
	m machine
}

func (m *arrayMachine) encode(e *encoder, v reflect.Value) {
	e.encodeUint(uint64(m.l))
	for i := 0; i < m.l; i++ {
		m.m.encode(e, v.Index(i))
	}
}

func (m *arrayMachine) decode(d *decoder, v reflect.Value) {
	l := m.l
	if t := int(d.decodeUint()); t < l {
		l = t
	}
	for i := 0; i < l; i++ {
		m.m.decode(d, v.Index(i))
	}
}

type chanMachine struct {
	z         reflect.Value
	t, ts, tc reflect.Type
	m, ms     machine
}

func (m *chanMachine) encode(e *encoder, v reflect.Value) {
	if v.IsNil() {
		e.writeByte(0)
		return
	}
	s := reflect.MakeSlice(m.ts, 0, 8)
	for e, ok := v.Recv(); ok; e, ok = v.Recv() {
		s = reflect.Append(s, e)
	}
	m.ms.encode(e, s)
}

func (m *chanMachine) decode(d *decoder, v reflect.Value) {
	if decodeZero(d, v, m.z) {
		return
	}

	l := int(d.decodeUint())
	if v.IsNil() {
		v.Set(reflect.MakeChan(m.tc, int(l)))
	}
	for i := 0; i < l; i++ {
		e := reflect.New(m.t).Elem()
		m.m.decode(d, e)
		v.Send(e)
	}
}

type interfaceMachine struct{ z reflect.Value }

func (*interfaceMachine) encode(e *encoder, v reflect.Value) {
	if !v.IsValid() || v.IsNil() {
		e.writeByte(0)
		return
	}
	v = v.Elem()
	types.get(v.Type()).encode(e, v)
}

func (m *interfaceMachine) decode(d *decoder, v reflect.Value) {
	if !decodeZero(d, v, m.z) {
		v = v.Elem()
		types.get(v.Type()).decode(d, v)
	}
}

type mapMachine struct {
	t, tk, tv reflect.Type
	k, v      machine
}

func (m *mapMachine) encode(e *encoder, v reflect.Value) {
	e.encodeUint(uint64(v.Len()))
	for _, i := range v.MapKeys() {
		m.k.encode(e, i)
		m.v.encode(e, v.MapIndex(i))
	}
}

func (m *mapMachine) decode(d *decoder, v reflect.Value) {
	v.Set(reflect.MakeMap(m.t))
	for i, l := uint64(0), d.decodeUint(); i < l; i++ {
		key, val := reflect.New(m.tk).Elem(), reflect.New(m.tv).Elem()
		m.k.decode(d, key)
		m.v.decode(d, val)
		v.SetMapIndex(key, val)
	}
}

type ptrMachine struct {
	z reflect.Value
	t reflect.Type
	m machine
}

func (m *ptrMachine) encode(e *encoder, v reflect.Value) {
	if v.IsNil() {
		e.writeByte(0)
		return
	}
	m.m.encode(e, v.Elem())
}

func (m *ptrMachine) decode(d *decoder, v reflect.Value) {
	if decodeZero(d, v, m.z) {
		return
	}
	if v.IsNil() {
		v.Set(reflect.New(m.t))
	}
	m.m.decode(d, v.Elem())
}

type sliceMachine struct {
	t reflect.Type
	m machine
}

func (m *sliceMachine) encode(e *encoder, v reflect.Value) {
	e.encodeUint(uint64(v.Len()))
	for i, l := 0, v.Len(); i < l; i++ {
		m.m.encode(e, v.Index(i))
	}
}

func (m *sliceMachine) decode(d *decoder, v reflect.Value) {
	l := int(d.decodeUint())
	v.Set(reflect.MakeSlice(m.t, l, l))
	for i := 0; i < l; i++ {
		m.m.decode(d, v.Index(i))
	}
}

type stringMachine struct{}

func (stringMachine) encode(e *encoder, v reflect.Value) {
	e.encodeUint(uint64(v.Len()))
	e.writeString(v.String())
}

func (stringMachine) decode(d *decoder, v reflect.Value) {
	v.SetString(string(d.read(d.decodeUint())))
}

type structMachine []machine

func (m structMachine) encode(e *encoder, v reflect.Value) {
	e.encodeUint(uint64(len(m)))
	for i, m := range m {
		m.encode(e, v.Field(i))
	}
}

func (m structMachine) decode(d *decoder, v reflect.Value) {
	l := len(m)
	if t := int(d.decodeUint()); t < l {
		l = t
	}
	for i := 0; i < l; i++ {
		m[i].decode(d, v.Field(i))
	}
}

type bytesMachine struct{}

func (bytesMachine) encode(e *encoder, v reflect.Value) {
	e.encodeUint(uint64(v.Len()))
	e.write(v.Bytes())
}

func (bytesMachine) decode(d *decoder, v reflect.Value) {
	v.SetBytes(d.read(d.decodeUint()))
}

type marshalerMachine struct{ e, d bool }

func (m *marshalerMachine) encode(e *encoder, v reflect.Value) {
	if m.e {
		v = v.Addr()
	}
	ret, err := v.Interface().(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		panic(noPanic{err})
	}
	e.encodeUint(uint64(len(ret)))
	e.write(ret)
}

func (m *marshalerMachine) decode(d *decoder, v reflect.Value) {
	if m.d {
		v = v.Addr()
	}
	if err := v.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(d.read(d.decodeUint())); err != nil {
		panic(noPanic{err})
	}
}
