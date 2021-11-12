package crypto

import (
	"bytes"
	"crypto/cipher"
	"encoding/binary"
)

var _ cipher.Stream = (*TableCipher)(nil)

const tableSize = 256

type TableCipher []byte

func (t TableCipher) XORKeyStream(dst, src []byte) {
	for i := 0; i < len(src); i++ {
		dst[i] = t[src[i]]
	}
}

func NewTableCipher(key []byte) (enc, dec TableCipher) {
	enc = make([]byte, tableSize)
	dec = make([]byte, tableSize)
	table := make([]uint64, tableSize)

	var a uint64
	buf := bytes.NewBuffer(key)
	binary.Read(buf, binary.LittleEndian, &a)
	var i uint64
	for i = 0; i < tableSize; i++ {
		table[i] = i
	}
	for i = 1; i < 1024; i++ {
		table = sort(table, func(x, y uint64) int64 {
			return int64(a%(x+i) - a%(y+i))
		})
	}
	for i = 0; i < tableSize; i++ {
		enc[i] = byte(table[i])
	}
	for i = 0; i < tableSize; i++ {
		dec[enc[i]] = byte(i)
	}
	return enc, dec
}

func merge(left, right []uint64, comparison func(uint64, uint64) int64) []uint64 {
	result := make([]uint64, len(left)+len(right))
	l, r := 0, 0
	for (l < len(left)) && (r < len(right)) {
		if comparison(left[l], right[r]) <= 0 {
			result[l+r] = left[l]
			l++
		} else {
			result[l+r] = right[r]
			r++
		}
	}
	for l < len(left) {
		result[l+r] = left[l]
		l++
	}
	for r < len(right) {
		result[l+r] = right[r]
		r++
	}
	return result
}

func sort(arr []uint64, comparison func(uint64, uint64) int64) []uint64 {
	if len(arr) < 2 {
		return arr
	}
	middle := uint64(len(arr) / 2)
	return merge(sort(arr[0:middle], comparison), sort(arr[middle:], comparison), comparison)
}
