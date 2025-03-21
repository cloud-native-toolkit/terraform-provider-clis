// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

// +build appengine

package internal

func BytesToString(b []byte) string {
	return string(b)
}

func StringToBytes(s string) []byte {
	return []byte(s)
}
