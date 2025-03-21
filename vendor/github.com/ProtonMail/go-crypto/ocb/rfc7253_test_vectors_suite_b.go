// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package ocb

// Second set of test vectors from https://tools.ietf.org/html/rfc7253
var rfc7253TestVectorTaglen96 = struct {
	key, nonce, header, plaintext, ciphertext string
}{"0F0E0D0C0B0A09080706050403020100",
	"BBAA9988776655443322110D",
	"000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F2021222324252627",
	"000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F2021222324252627",
	"1792A4E31E0755FB03E31B22116E6C2DDF9EFD6E33D536F1A0124B0A55BAE884ED93481529C76B6AD0C515F4D1CDD4FDAC4F02AA"}

var rfc7253AlgorithmTest = []struct {
	KEYLEN, TAGLEN int
	OUTPUT         string
}{
	{128, 128, "67E944D23256C5E0B6C61FA22FDF1EA2"},
	{192, 128, "F673F2C3E7174AAE7BAE986CA9F29E17"},
	{256, 128, "D90EB8E9C977C88B79DD793D7FFA161C"},
	{128, 96, "77A3D8E73589158D25D01209"},
	{192, 96, "05D56EAD2752C86BE6932C5E"},
	{256, 96, "5458359AC23B0CBA9E6330DD"},
	{128, 64, "192C9B7BD90BA06A"},
	{192, 64, "0066BC6E0EF34E24"},
	{256, 64, "7D4EA5D445501CBE"},
}
