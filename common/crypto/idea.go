/*
 * This file is licensed under the MIT License.
 *
 * It is translated from https://github.com/StephenGenusa/DCPCrypt/blob/5fc435d45fb0a8ae93fdb02be2f114a3e935a9c7/Ciphers/DCPidea.pas.
 * DCPidea.pas is licensed under the MIT License.
 *
 * The license text of DCPidea.pas:
 *
 * Copyright (c) 1999-2002 David Barton
 * Permission is hereby granted, free of charge, to any person obtaining a
 * copy of this software and associated documentation files (the "Software"),
 * to deal in the Software without restriction, including without limitation
 * the rights to use, copy, modify, merge, publish, distribute, sublicense,
 * and/or sell copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
 * THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
 * DEALINGS IN THE SOFTWARE.
 */

package crypto

import (
	"crypto/cipher"
	"encoding/binary"
)

type ideaCipher struct {
	ek [52]uint16
	dk [52]uint16
}

func mul(x uint16, y uint16) uint16 {
	p := uint32(x) * uint32(y)
	if p == 0 {
		x = 1 - x - y
	} else {
		x = uint16(p >> 16)
		t16 := uint16(p & 0xffff)
		x = t16 - x
		if t16 < x {
			x++
		}
	}
	return x
}

func mulInv(x uint16) uint16 {
	if x <= 1 {
		return x
	}
	t1 := uint16(0x10001 / uint32(x))
	y := uint16(0x10001 % uint32(x))
	if y == 1 {
		return 1 - t1
	}
	t0 := uint16(1)
	q := uint16(0)
	for {
		q = x / y
		x %= y
		t0 += q * t1
		if x == 1 {
			return t0
		}
		q = y / x
		y %= x
		t1 += q * t0
		if y == 1 {
			return 1 - t1
		}
	}
}

func (c *ideaCipher) BlockSize() int {
	return 8
}

func NewIdeaCipher(key []byte) (cipher.Block, error) {
	if len(key) != 16 {
		panic("invalid key size")
	}

	var ek [52]uint16
	for i := range 8 {
		ek[i] = binary.BigEndian.Uint16(key[i*2 : i*2+2])
	}
	for i := range 5 {
		ek[(i+1)*8] = ek[i*8+1]<<9 | ek[i*8+2]>>7
		ek[(i+1)*8+1] = ek[i*8+2]<<9 | ek[i*8+3]>>7
		ek[(i+1)*8+2] = ek[i*8+3]<<9 | ek[i*8+4]>>7
		ek[(i+1)*8+3] = ek[i*8+4]<<9 | ek[i*8+5]>>7
		ek[(i+1)*8+4] = ek[i*8+5]<<9 | ek[i*8+6]>>7
		ek[(i+1)*8+5] = ek[i*8+6]<<9 | ek[i*8+7]>>7
		ek[(i+1)*8+6] = ek[i*8+7]<<9 | ek[i*8]>>7
		ek[(i+1)*8+7] = ek[i*8]<<9 | ek[i*8+1]>>7
	}
	ek[48] = ek[41]<<9 | ek[42]>>7
	ek[49] = ek[42]<<9 | ek[43]>>7
	ek[50] = ek[43]<<9 | ek[44]>>7
	ek[51] = ek[44]<<9 | ek[45]>>7

	var dk [52]uint16
	dk[51] = mulInv(ek[3])
	dk[50] = -ek[2]
	dk[49] = -ek[1]
	dk[48] = mulInv(ek[0])
	for i := range 7 {
		dk[47-i*6] = ek[i*6+5]
		dk[46-i*6] = ek[i*6+4]
		dk[45-i*6] = mulInv(ek[i*6+9])
		dk[44-i*6] = -ek[i*6+7]
		dk[43-i*6] = -ek[i*6+8]
		dk[42-i*6] = mulInv(ek[i*6+6])
	}
	dk[5] = ek[47]
	dk[4] = ek[46]
	dk[3] = mulInv(ek[51])
	dk[2] = -ek[50]
	dk[1] = -ek[49]
	dk[0] = mulInv(ek[48])

	return &ideaCipher{
		ek: ek,
		dk: dk,
	}, nil
}

func (c *ideaCipher) Encrypt(dst, src []byte) {
	crypt(c.ek, dst, src)
}

func (c *ideaCipher) Decrypt(dst, src []byte) {
	crypt(c.dk, dst, src)
}

func crypt(key [52]uint16, dst, src []byte) {
	x1 := binary.BigEndian.Uint16(src[0:2])
	x2 := binary.BigEndian.Uint16(src[2:4])
	x3 := binary.BigEndian.Uint16(src[4:6])
	x4 := binary.BigEndian.Uint16(src[6:8])

	var s3, s2 uint16
	for i := range 8 {
		x1 = mul(x1, key[i*6])
		x2 += key[i*6+1]
		x3 += key[i*6+2]
		x4 = mul(x4, key[i*6+3])
		s3 = x3
		x3 ^= x1
		x3 = mul(x3, key[i*6+4])
		s2 = x2
		x2 ^= x4
		x2 += x3
		x2 = mul(x2, key[i*6+5])
		x3 += x2
		x1 ^= x2
		x4 ^= x3
		x2 ^= s3
		x3 ^= s2
	}
	x1 = mul(x1, key[48])
	x3 += key[49]
	x2 += key[50]
	x4 = mul(x4, key[51])
	x3, x2 = x2, x3

	binary.BigEndian.PutUint16(dst[0:2], x1)
	binary.BigEndian.PutUint16(dst[2:4], x2)
	binary.BigEndian.PutUint16(dst[4:6], x3)
	binary.BigEndian.PutUint16(dst[6:8], x4)
}
