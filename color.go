package main

import (
	"bytes"
	"strconv"
	"unsafe"
)

const GREY = "\x1b[38;2;111;111;111m"
const CLR = "\x1b[0m"

var ESC = []byte{0x5c, 0x78, 0x31, 0x62, 0x5b}
var CLEAR = []byte{0x5c, 0x78, 0x31, 0x62, 0x5b, 0x30, 0x6d}

type Color struct {
	disable bool
	values  [256]string
	bvalues [256][]byte
	cvalues map[byte][]byte
}

func (c *Color) Compute() {
	const WHITEB = "\x1b[1;37m"
	for i := 0; i < 256; i++ {
		var fg, bg string

		lowVis := i == 0 || (i >= 16 && i <= 20) || (i >= 232 && i <= 242)

		if lowVis {
			fg = WHITEB + "\x1b[38;5;" + "255" + "m"
			bg = "\x1b[48;5;" + strconv.Itoa(int(i)) + "m"
		} else {
			fg = "\x1b[38;5;" + strconv.Itoa(int(i)) + "m"
			bg = ""
		}
		c.values[i] = bg + fg
	}
}

func (c *Color) Colorize(s string, clr byte) string {
	const NOCOLOR = "\x1b[0m"
	return c.values[clr] + s + NOCOLOR
}

func (c *Color) ColorizeBytes(s string, byteColor []byte) []byte {
	const NOCOLOR = "\x1b[0m"
	b := ByteArrayToInt(byteColor)
	return []byte(c.values[b] + s + NOCOLOR)
}

func (c *Color) ComputeBytes() {
	const WHITEB = "\x1b[1;37m"
	for i := 0; i < 256; i++ {
		var fg, bg string
		b := byte(i)

		lowVis := i == 0 || (i >= 16 && i <= 20) || (i >= 232 && i <= 242)

		if lowVis {
			fg = WHITEB + "\x1b[38;5;" + "255" + "m"
			bg = "\x1b[48;5;" + strconv.Itoa(int(i)) + "m"
		} else {
			fg = "\x1b[38;5;" + strconv.Itoa(int(i)) + "m"
			bg = ""
		}

		c.values[i] = bg + fg
		c.cvalues[b] = []byte(bg + fg)
	}
}

func (c *Color) xComputeBytes() {
	const Marker = '\x1b'
	var b bytes.Buffer

	for i := 0; i < 256; i++ {
		// var fg, bg []byte
		b.Write(ESC)

		// x := string(i)
		// y := []byte(x)

		lowVis := i == 0 || (i >= 16 && i <= 20) || (i >= 232 && i <= 242)
		if lowVis {
			b.Write([]byte{'[', '1', ';', '3', '7', 'm'})
			b.Write(ESC)
			b.Write([]byte{'[', '4', '8', ';', '5'})
			bg := make([]byte, 3)
			bg = IntToByteArray(i)
			b.Write(bg)
			b.WriteByte('m')
		} else {
			b.Write([]byte{'[', '3', '8', ';', '5'})
			fg := make([]byte, 3)
			fg = IntToByteArray(i)
			b.Write(fg)
			b.WriteByte('m')
		}
		// c.values[i] = bg + fg
		// c.bvalues[i] = bytes.Join([]byte(bg), []byte(fg))
		c.bvalues[i] = b.Bytes()
	}
}

func IntToByteArray(num int) []byte {
	size := int(unsafe.Sizeof(num))
	arr := make([]byte, size)
	for i := 0; i < size; i++ {
		byt := *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&num)) + uintptr(i)))
		arr[i] = byt
	}
	return arr
}

func ByteArrayToInt(arr []byte) int64 {
	val := int64(0)
	size := len(arr)
	for i := 0; i < size; i++ {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}
	return val
}
