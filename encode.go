package main

import ()

func binaryEncode(dst, src []byte) {
	d := uint(0)
	_, _ = src[0], dst[7]
	for i := 7; i >= 0; i-- {
		if src[0]&(1<<d) == 0 {
			dst[i] = 0
		} else {
			dst[i] = 1
		}
		d++
	}
}

// returns -1 on success
// returns k > -1 if space found where k is index of space byte
func binaryDecode(dst, src []byte) int {
	var v, d byte

	for i := 0; i < len(src); i++ {
		v, d = src[i], d<<1
		if isSpace(v) { // found a space, so between groups
			if i == 0 {
				return 1
			}
			return i
		}
		if v == '1' {
			d ^= 1
		} else if v != '0' {
			return i // will catch issues like "000000: "
		}
	}

	dst[0] = d
	return -1
}

func cfmtEncode(dst, src []byte, hextable string) {
	b := src[0]
	dst[3] = hextable[b&0x0f]
	dst[2] = hextable[b>>4]
	dst[1] = 'x'
	dst[0] = '0'
}

// copied from encoding/hex package in order to add support for uppercase hex
func hexEncode(dst, src []byte, hextable string) {
	b := src[0]
	dst[1] = hextable[b&0x0f]
	dst[0] = hextable[b>>4]
}

// copied from encoding/hex package
// returns -1 on bad byte or space (\t \s \n)
// returns -2 on two consecutive spaces
// returns 0 on success
func hexDecode(dst, src []byte) int {
	_, _ = src[2], dst[0]

	if isSpace(src[0]) {
		if isSpace(src[1]) {
			return -2
		}
		return -1
	}

	if isPrefix(src[0:2]) {
		src = src[2:]
	}

	for i := 0; i < len(src)/2; i++ {
		a, ok := fromHexChar(src[i*2])
		if !ok {
			return -1
		}
		b, ok := fromHexChar(src[i*2+1])
		if !ok {
			return -1
		}

		dst[0] = (a << 4) | b
	}
	return 0
}

// copied from encoding/hex package
func fromHexChar(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}

	return 0, false
}

// check if entire line is full of empty []byte{0} bytes (nul in C)
func empty(b *[]byte) bool {
	for i := 0; i < len(*b); i++ {
		if (*b)[i] != 0 {
			return false
		}
	}
	return true
}

// check if filename character contains problematic characters
func isSpecial(b byte) bool {
	switch b {
	case '/', '!', '#', '$', '%', '^', '&', '*', '(', ')', ';', ':', '|', '{', '}', '\\', '~', '`':
		return true
	default:
		return false
	}
}

// quick binary tree check
// probably horribly written idk it's late at night
func parseSpecifier(b string) float64 {
	lb := len(b)
	if lb == 0 {
		return 0
	}

	var b0, b1 byte
	if lb < 2 {
		b0 = b[0]
		b1 = '0'
	} else {
		b1 = b[1]
		b0 = b[0]
	}

	if b1 != '0' {
		if b1 == 'b' { // bits, so convert bytes to bits for os.Seek()
			if b0 == 'k' || b0 == 'K' {
				return 0.0078125
			}

			if b0 == 'm' || b0 == 'M' {
				return 7.62939453125e-06
			}

			if b0 == 'g' || b0 == 'G' {
				return 7.45058059692383e-09
			}
		}

		if b1 == 'B' { // kilo/mega/giga- bytes are assumed
			if b0 == 'k' || b0 == 'K' {
				return 1024
			}

			if b0 == 'm' || b0 == 'M' {
				return 1048576
			}

			if b0 == 'g' || b0 == 'G' {
				return 1073741824
			}
		}
	} else { // kilo/mega/giga- bytes are assumed for single b, k, m, g
		if b0 == 'k' || b0 == 'K' {
			return 1024
		}

		if b0 == 'm' || b0 == 'M' {
			return 1048576
		}

		if b0 == 'g' || b0 == 'G' {
			return 1073741824
		}
	}

	return 1 // assumes bytes as fallback
}

// is byte a space? (\t, \n, \s)
func isSpace(b byte) bool {
	switch b {
	case 32, 12, 9:
		return true
	default:
		return false
	}
}

// are the two bytes hex prefixes? (0x or 0X)
func isPrefix(b []byte) bool {
	return b[0] == '0' && (b[1] == 'x' || b[1] == 'X')
}
