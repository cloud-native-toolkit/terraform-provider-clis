// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

// +build !appengine

package internal

import (
	"unsafe"
)

// BytesToString converts byte slice to string.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes converts string to byte slice.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
