package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	OffsetFormat string `short:"t" long:"radix" default:"x" choice:"d" choice:"o" choice:"x" description:"Print offset in [d|o|x] format"`
	Binary       bool   `short:"b" long:"binary" description:"output in binary format (01010101) incompatible with plain, reverse and include"`
	Reverse      bool   `short:"r" long:"reverse" description:"re-assemble hexdump output back into binary"`
	Autoskip     bool   `short:"a" long:"autoskip" description:"toggle autoskip (replaces blank lines with a *)"`
	Bars         bool   `short:"B" long:"bars" description:"delimiter bars in ascii table"`
	Seek         int64  `short:"s" long:"seek" description:"start at <seek> bytes"`
	Len          int64  `short:"l" long:"len" description:"stop after <len> octets"`
	Columns      int    `short:"c" long:"columns" description:"column count"`
	GroupSize    int    `short:"g" long:"groups" description:"group count"`
	Plain        bool   `short:"p" long:"plain" description:"plain output without ascii table and offset row [often used with hexxy -r]"`
	Upper        bool   `short:"u" long:"upper" description:"output hex in UPPERCASE format"`
	CInclude     bool   `short:"i" long:"include" description:"output in C include format"`
	OutputFile   string `short:"o" long:"output" description:"automatically output to file instead of STDOUT"`
	Separator    string `long:"separator" default:"|" description:"separator character for the ascii character table"`
	ForceColor   bool   `short:"F" long:"force-color" description:"color is automatically disabled if output is a pipe, this option forces color output"`
	NoColor      bool   `short:"N" long:"no-color" description:"do not print output with color"`
	Verbose      bool   `short:"v" long:"verbose" description:"print debugging information and verbose output"`
}

var Debug = func(string, ...interface{}) {}

const (
	dumpHex = iota
	dumpBinary
	dumpCformat
	dumpPlain
)

const (
	ldigits = "0123456789abcdef"
	udigits = "0123456789ABCDEF"
)

var (
	dumpType     int
	space        = []byte(" ")
	doubleSpace  = []byte("  ")
	dot          = []byte(".")
	newLine      = []byte("\n")
	zeroHeader   = []byte("0000000: ")
	unsignedChar = []byte("unsigned char ")
	unsignedInt  = []byte("};\nunsigned int ")
	lenEquals    = []byte("_len = ")
	brackets     = []byte("[] = {")
	asterisk     = []byte("*")
	commaSpace   = []byte(", ")
	comma        = []byte(",")
	semiColonNl  = []byte(";\n")
	bar          = []byte("|")
)

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

const GREY = "\x1b[38;2;111;111;111m"
const CLR = "\x1b[0m"

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

func inputIsPipe() bool {
	stat, _ := os.Stdin.Stat()
	return stat.Mode()&os.ModeCharDevice != os.ModeCharDevice
}

func outputIsPipe() bool {
	stat, _ := os.Stdout.Stat()
	return stat.Mode()&os.ModeCharDevice != os.ModeCharDevice
}

func HexdumpPlain(file *os.File) error {
	var i uint64
	reader := bufio.NewReaderSize(file, 10*1024)

	for {
		b, err := reader.ReadByte()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("Failed to read %v: %w\n", file.Name(), err)
		}

		if i%30 == 0 {
			println()
		}

		fmt.Printf("%02x", b)
		i++
	}

	return nil
}

// func plain2Binary(file *os.File) error {
// 	return reverse(os.Stdout, os.Stdin)
// }

// func getOffsetFormat() error {
// 	var prefix string
// 	var suffix string
// 	var format string

// 	switch opts.OffsetFormat {
// 	case "d":
// 		format = prefix + "%08d  " + suffix
// 	case "o":
// 		format = prefix + "%08o  " + suffix
// 	case "x":
// 		format = prefix + "%08x  " + suffix
// 	default:
// 		return fmt.Errorf("Offset format must be [d|o|x]")
// 	}
// 	return nil
// }

func XXD(r io.Reader, w io.Writer, filename string) error {
	var (
		lineOffset  int64
		hexOffset   = make([]byte, 6)
		groupSize   int
		cols        int
		octs        int
		caps        = ldigits
		doCheader   = true
		doCEnd      bool
		varDeclChar = make([]byte, 14+len(filename)+6) // for "unsigned char NAME_FORMAT[] = {"
		varDeclInt  = make([]byte, 16+len(filename)+7) // enough room for "unsigned int NAME_FORMAT = "
		nulLine     int64
		totalOcts   int64
	)

	if dumpType == dumpCformat {
		_ = copy(varDeclChar[0:14], unsignedChar[:])
		_ = copy(varDeclInt[0:14], lenEquals[:])

		for i := 0; i < len(filename); i++ {
			if filename[i] != '.' {
				varDeclChar[14+i] = filename[i]
				varDeclInt[16+i] = filename[i]
			} else {
				varDeclChar[14+i] = '_'
				varDeclInt[16+i] = '_'
			}
		}
		// copy "[] = {" and "_len = "
		_ = copy(varDeclChar[14+len(filename):], brackets[:])
		_ = copy(varDeclInt[16+len(filename):], lenEquals[:])
	}

	if opts.Upper {
		caps = udigits
	}

	if opts.Columns == -1 {
		switch dumpType {
		case dumpPlain:
			cols = 30
		case dumpCformat:
			cols = 12
		case dumpBinary:
			cols = 6
		default:
			cols = 16
		}
	} else {
		cols = opts.Columns
	}

	switch dumpType {
	case dumpBinary:
		octs = 8
		groupSize = 1
	case dumpPlain:
		octs = 0
	case dumpCformat:
		octs = 4
	default:
		octs = 2
		groupSize = 2
	}

	if opts.GroupSize != -1 {
		groupSize = opts.GroupSize
	}

	if opts.Len != -1 {
		if opts.Len < int64(cols) {
			cols = int(opts.Len)
		}
	}

	if octs < 1 {
		octs = cols
	}

	// allocate their size based on the users specs, hence why its declared here
	var (
		line = make([]byte, cols)
		char = make([]byte, octs)
	)

	c := int64(0)
	nl := int64(0)
	r = bufio.NewReader(r)

	var (
		n   int
		err error
	)

	for {
		n, err = io.ReadFull(r, line)
		if err != nil && errors.Is(err, io.EOF) && errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("hexxy: %v", err)
		}

		if dumpType == dumpPlain && n != 0 {
			for i := 0; i < n; i++ {
				hexEncode(char, line[i:i+1], caps)
				w.Write(char)
				c++
			}
			continue
		}

		if n == 0 {
			if dumpType == dumpPlain {
				w.Write(newLine)
			}

			if dumpType == dumpCformat {
				doCEnd = true
			} else {
				return nil
			}
		}

		if opts.Len != -1 {
			if totalOcts == opts.Len {
				break
			}
			totalOcts += opts.Len
		}

		if opts.Autoskip && empty(&line) {
			if nulLine == 1 {
				w.Write(asterisk)
				w.Write(newLine)
			}

			nulLine++

			if nulLine > 1 {
				lineOffset++
				continue
			}
		}

		// hex or binary formats only
		if dumpType <= dumpBinary {
			// line offset
			hexOffset = strconv.AppendInt(hexOffset[0:0], lineOffset, 16)
			w.Write(zeroHeader[0:(6 - len(hexOffset))])
			w.Write(hexOffset)
			w.Write(zeroHeader[6:])
			lineOffset++
		} else if doCheader {
			w.Write(varDeclChar)
			w.Write(newLine)
			doCheader = false
		}

		if dumpType == dumpBinary {
			// binary values
			for i, k := 0, octs; i < n; i, k = i+1, k+octs {
				binaryEncode(char, line[i:i+1])
				w.Write(char)
				c++

				if k == octs*groupSize {
					k = 0
					w.Write(space)
				}
			}
		} else if dumpType == dumpCformat {
			if !doCEnd {
				w.Write(doubleSpace)
			}
			for i := 0; i < n; i++ {
				cfmtEncode(char, line[i:i+1], caps)
				w.Write(char)
				c++
				// no space at EOL
				if i != n-1 {
					w.Write(commaSpace)
				} else if n == cols {
					w.Write(comma)
				}
			}
		} else {
			// hex values -- default
			for i, k := 0, octs; i < n; i, k = i+1, k+octs {
				hexEncode(char, line[i:i+1], caps)
				w.Write(char)
				c++

				if k == octs*groupSize {
					k = 0
					w.Write(space)
				}
			}
		}

		if doCEnd {
			w.Write(varDeclInt)
			w.Write([]byte(strconv.FormatInt(c, 10)))
			w.Write(semiColonNl)
			return nil
		}

		if n < len(line) && dumpType <= dumpBinary {
			for i := n * octs; i < len(line)*octs; i++ {
				w.Write(space)

				if i%octs == 1 {
					w.Write(space)
				}
			}
		}

		if dumpType != dumpCformat {
			w.Write(space)
		}

		if dumpType <= dumpBinary {
			// character values
			b := line[:n]
			// |hello,.world!|
			if opts.Bars {
				w.Write(bar)
			}

			var v byte
			for i := 0; i < len(b); i++ {
				v = b[i]
				if v > 0x1f && v < 0x7f {
					w.Write(line[i : i+1])
				} else {
					w.Write(dot)
				}
			}

			if opts.Bars {
				w.Write(bar)
			}
		}
		w.Write(newLine)
		nl++
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

	var infile, outfile *os.File
	var err error

	if len(args) < 1 && inputIsPipe() {
		infile = os.Stdin
	} else {
		infile, err = os.Open(args[0])
		if err != nil {
			return fmt.Errorf("hexxy: %v", err.Error())
		}
	}
	defer infile.Close()

	if opts.Seek != -1 {
		_, err = infile.Seek(opts.Seek, io.SeekStart)
		if err != nil {
			return fmt.Errorf("hexxy: %v", err.Error())
		}
	}

	if opts.OutputFile != "" {
		outfile, err = os.Open(opts.OutputFile)
		if err != nil {
			return fmt.Errorf("hexxy: %v", err.Error())
		}
	} else {
		outfile = os.Stdout
	}
	defer outfile.Close()

	switch {
	case opts.Binary:
		dumpType = dumpBinary
	case opts.CInclude:
		dumpType = dumpCformat
	case opts.Plain:
		dumpType = dumpPlain
	default:
		dumpType = dumpHex
	}

	out := bufio.NewWriter(outfile)
	defer out.Flush()

	if opts.Reverse {

	}

	if err := XXD(infile, out, infile.Name()); err != nil {
		return fmt.Errorf("hexxy: %v", err.Error())
	}

	return nil
}

const usage_msg = `
hexxy is a command line hex dumping tool

Examples:
	hexxy [OPTIONS] input-file
`

// extra usage examples
func usage() {
	fmt.Fprint(os.Stderr, usage_msg)
}

func init() {
	opts.Seek = -1 // default no-op value
	opts.Columns = -1
	opts.GroupSize = -1
	opts.Len = -1
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	args, err := parser.Parse()
	if flags.WroteHelp(err) {
		os.Exit(0)
	}
	if err != nil {
		log.Fatal(err)
	}

	if !inputIsPipe() && len(args) == 0 {
		parser.WriteHelp(os.Stderr)
		os.Exit(0)
	}

	if opts.Verbose {
		Debug = log.Printf
	}
	if err := Hexxy(args); err != nil {
		log.Fatal(err)
	}
}
