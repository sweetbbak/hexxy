package main

import (
	"os"
	"strconv"
)

const GREY = "\x1b[38;2;111;111;111m"
const CLR = "\x1b[0m"

var ESC = []byte{0x5c, 0x78, 0x31, 0x62, 0x5b}
var CLEAR = []byte("\x1b[0m")

// var CLEAR = []byte{0x5c, 0x78, 0x31, 0x62, 0x5b, 0x30, 0x6d}

type Color struct {
	disable bool
	values  [256]string
	cvalues [256][]byte
}

// check for NO_COLOR env var and block color
func HasNoColorEnvVar() bool {
	_, hasEnv := os.LookupEnv("NO_COLOR")
	return hasEnv
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
		c.cvalues[i] = []byte(bg + fg)
	}
}

func (c *Color) Colorize(s string, clr byte) string {
	const NOCOLOR = "\x1b[0m"
	return c.values[clr] + s + NOCOLOR
}

// function to colorize bytes - avoiding string conversions
func (c *Color) Colorize2(clr byte) ([]byte, []byte) {
	return c.cvalues[clr], CLEAR
}
