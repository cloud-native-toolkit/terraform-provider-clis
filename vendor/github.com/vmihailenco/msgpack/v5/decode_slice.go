// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package msgpack

import (
	"fmt"
	"reflect"

	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

var sliceStringPtrType = reflect.TypeOf((*[]string)(nil))

// DecodeArrayLen decodes array length. Length is -1 when array is nil.
func (d *Decoder) DecodeArrayLen() (int, error) {
	c, err := d.readCode()
	if err != nil {
		return 0, err
	}
	return d.arrayLen(c)
}

func (d *Decoder) arrayLen(c byte) (int, error) {
	if c == msgpcode.Nil {
		return -1, nil
	} else if c >= msgpcode.FixedArrayLow && c <= msgpcode.FixedArrayHigh {
		return int(c & msgpcode.FixedArrayMask), nil
	}
	switch c {
	case msgpcode.Array16:
		n, err := d.uint16()
		return int(n), err
	case msgpcode.Array32:
		n, err := d.uint32()
		return int(n), err
	}
	return 0, fmt.Errorf("msgpack: invalid code=%x decoding array length", c)
}

func decodeStringSliceValue(d *Decoder, v reflect.Value) error {
	ptr := v.Addr().Convert(sliceStringPtrType).Interface().(*[]string)
	return d.decodeStringSlicePtr(ptr)
}

func (d *Decoder) decodeStringSlicePtr(ptr *[]string) error {
	n, err := d.DecodeArrayLen()
	if err != nil {
		return err
	}
	if n == -1 {
		return nil
	}

	ss := makeStrings(*ptr, n, d.flags&disableAllocLimitFlag != 0)
	for i := 0; i < n; i++ {
		s, err := d.DecodeString()
		if err != nil {
			return err
		}
		ss = append(ss, s)
	}
	*ptr = ss

	return nil
}

func makeStrings(s []string, n int, noLimit bool) []string {
	if !noLimit && n > sliceAllocLimit {
		n = sliceAllocLimit
	}

	if s == nil {
		return make([]string, 0, n)
	}

	if cap(s) >= n {
		return s[:0]
	}

	s = s[:cap(s)]
	s = append(s, make([]string, n-len(s))...)
	return s[:0]
}

func decodeSliceValue(d *Decoder, v reflect.Value) error {
	n, err := d.DecodeArrayLen()
	if err != nil {
		return err
	}

	if n == -1 {
		v.Set(reflect.Zero(v.Type()))
		return nil
	}
	if n == 0 && v.IsNil() {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		return nil
	}

	if v.Cap() >= n {
		v.Set(v.Slice(0, n))
	} else if v.Len() < v.Cap() {
		v.Set(v.Slice(0, v.Cap()))
	}

	noLimit := d.flags&disableAllocLimitFlag != 1

	if noLimit && n > v.Len() {
		v.Set(growSliceValue(v, n, noLimit))
	}

	for i := 0; i < n; i++ {
		if !noLimit && i >= v.Len() {
			v.Set(growSliceValue(v, n, noLimit))
		}

		elem := v.Index(i)
		if err := d.DecodeValue(elem); err != nil {
			return err
		}
	}

	return nil
}

func growSliceValue(v reflect.Value, n int, noLimit bool) reflect.Value {
	diff := n - v.Len()
	if !noLimit && diff > sliceAllocLimit {
		diff = sliceAllocLimit
	}
	v = reflect.AppendSlice(v, reflect.MakeSlice(v.Type(), diff, diff))
	return v
}

func decodeArrayValue(d *Decoder, v reflect.Value) error {
	n, err := d.DecodeArrayLen()
	if err != nil {
		return err
	}

	if n == -1 {
		return nil
	}
	if n > v.Len() {
		return fmt.Errorf("%s len is %d, but msgpack has %d elements", v.Type(), v.Len(), n)
	}

	for i := 0; i < n; i++ {
		sv := v.Index(i)
		if err := d.DecodeValue(sv); err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) DecodeSlice() ([]interface{}, error) {
	c, err := d.readCode()
	if err != nil {
		return nil, err
	}
	return d.decodeSlice(c)
}

func (d *Decoder) decodeSlice(c byte) ([]interface{}, error) {
	n, err := d.arrayLen(c)
	if err != nil {
		return nil, err
	}
	if n == -1 {
		return nil, nil
	}

	s := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		v, err := d.decodeInterfaceCond()
		if err != nil {
			return nil, err
		}
		s = append(s, v)
	}

	return s, nil
}

func (d *Decoder) skipSlice(c byte) error {
	n, err := d.arrayLen(c)
	if err != nil {
		return err
	}

	for i := 0; i < n; i++ {
		if err := d.Skip(); err != nil {
			return err
		}
	}

	return nil
}
