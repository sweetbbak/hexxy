package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	NoColor bool `short:"N" long:"no-color" description:"do not print output with color"`
	Verbose bool `short:"v" long:"verbose" description:"print debugging information and verbose output"`
}

var Debug = func(string, ...interface{}) {}

type Color struct {
	disable bool
	values  [256]string
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

func stdinOpen() bool {
	stat, _ := os.Stdin.Stat()
	if stat.Mode()&os.ModeCharDevice == os.ModeCharDevice {
		return false
	} else {
		return true
	}
}

func asciiRow(ascii []byte, clr *Color, stdout io.Writer) {
	var s string
	for _, b := range ascii {
		if b >= 33 && b <= 126 {
			s = clr.Colorize(string(b), b)
		} else {
			s = clr.Colorize(".", b)
		}

		fmt.Fprint(stdout, s)
	}
}

func Hexdump(file *os.File, color *Color) error {
	stdout := bufio.NewWriter(os.Stdout)
	stderr := os.Stderr
	ascii := [16]byte{}

	var i uint64 = 0
	reader := bufio.NewReaderSize(file, 10*1024*1024)

	for {
		b, err := reader.ReadByte()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fmt.Fprintf(stderr, "Failed to read %v: %v\n", file.Name(), err)
			return err
		}

		ascii[i%16] = b

		// offset
		if i%16 == 0 {
			fmt.Fprintf(stdout, "%08x   ", i)
		}

		// byte
		fmt.Fprintf(stdout, color.Colorize("%02x", b)+" ", b)

		// extra space every 4 bytes
		if (i+1)%4 == 0 {
			fmt.Fprint(stdout, " ")
		}

		// print ascii row and newline │ | ┆
		if (i+1)%16 == 0 {
			fmt.Fprint(stdout, "│")
			asciiRow(ascii[:i%16], color, stdout)
			fmt.Fprintln(stdout, "│")
			ascii = [16]byte{} // reset
		}

		i++
	}

	if i%16 != 0 {
		left := int(16 - i%16)
		spaces := 3*left + (left-1)/4 + 1
		fmt.Fprint(stdout, strings.Repeat(" ", spaces))
		fmt.Fprint(stdout, "│")
		asciiRow(ascii[:i%16], color, stdout)
		fmt.Fprintln(stdout, "│")
		fmt.Fprintf(stdout, "%08x\n", i)
	}

	return nil
}

func Hexxy(args []string) error {
	color := &Color{}

	if opts.NoColor {
		color.disable = true
	}

	if !color.disable {
		color.Compute()
	}

	if len(args) < 1 && stdinOpen() {
		return Hexdump(os.Stdin, color)
	}

	for _, f := range args {
		file, err := os.Open(f)
		if err != nil {
			return err
		}

		Hexdump(file, color)
	}

	return nil
}

func main() {
	args, err := flags.Parse(&opts)
	if flags.WroteHelp(err) {
		os.Exit(0)
	}
	if err != nil {
		log.Fatal(err)
	}

	if opts.Verbose {
		Debug = log.Printf
	}

	if err := Hexxy(args); err != nil {
		log.Fatal(err)
	}
}
