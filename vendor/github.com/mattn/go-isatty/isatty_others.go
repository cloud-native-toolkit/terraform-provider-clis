// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

//go:build (appengine || js || nacl || tinygo || wasm) && !windows
// +build appengine js nacl tinygo wasm
// +build !windows

package isatty

// IsTerminal returns true if the file descriptor is terminal which
// is always false on js and appengine classic which is a sandboxed PaaS.
func IsTerminal(fd uintptr) bool {
	return false
}

// IsCygwinTerminal() return true if the file descriptor is a cygwin or msys2
// terminal. This is also always false on this environment.
func IsCygwinTerminal(fd uintptr) bool {
	return false
}
