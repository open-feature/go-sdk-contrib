package vercel

import (
	"encoding/binary"
	"math/bits"
)

const (
	xxPrime32_1 uint32 = 2654435761
	xxPrime32_2 uint32 = 2246822519
	xxPrime32_3 uint32 = 3266489917
	xxPrime32_4 uint32 = 668265263
	xxPrime32_5 uint32 = 374761393
)

func xxhash32String(s string, seed uint32) uint32 {
	b := []byte(s)
	n := len(b)
	i := 0

	var h uint32
	if n >= 16 {
		v1 := seed + xxPrime32_1 + xxPrime32_2
		v2 := seed + xxPrime32_2
		v3 := seed
		v4 := seed - xxPrime32_1

		for i <= n-16 {
			v1 = xxRound32(v1, binary.LittleEndian.Uint32(b[i:]))
			i += 4
			v2 = xxRound32(v2, binary.LittleEndian.Uint32(b[i:]))
			i += 4
			v3 = xxRound32(v3, binary.LittleEndian.Uint32(b[i:]))
			i += 4
			v4 = xxRound32(v4, binary.LittleEndian.Uint32(b[i:]))
			i += 4
		}

		h = bits.RotateLeft32(v1, 1) +
			bits.RotateLeft32(v2, 7) +
			bits.RotateLeft32(v3, 12) +
			bits.RotateLeft32(v4, 18)
	} else {
		h = seed + xxPrime32_5
	}

	h += uint32(n)

	for i <= n-4 {
		h += binary.LittleEndian.Uint32(b[i:]) * xxPrime32_3
		h = bits.RotateLeft32(h, 17) * xxPrime32_4
		i += 4
	}

	for i < n {
		h += uint32(b[i]) * xxPrime32_5
		h = bits.RotateLeft32(h, 11) * xxPrime32_1
		i++
	}

	h ^= h >> 15
	h *= xxPrime32_2
	h ^= h >> 13
	h *= xxPrime32_3
	h ^= h >> 16
	return h
}

func xxRound32(acc, input uint32) uint32 {
	acc += input * xxPrime32_2
	acc = bits.RotateLeft32(acc, 13)
	acc *= xxPrime32_1
	return acc
}
