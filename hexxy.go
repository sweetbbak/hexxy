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
	OffsetFormat string `short:"t" long:"radix" default:"x" choice:"d" choice:"o" choice:"x" description:"Print offset in [d|o|x] format"`
	Binary       bool   `short:"b" long:"binary" description:"output in binary format (01010101) incompatible with plain, reverse and include"`
	Reverse      bool   `short:"r" long:"reverse" description:"re-assemble hexdump output back into binary"`
	Autoskip     bool   `short:"a" long:"autoskip" description:"toggle autoskip (replaces blank lines with a *)"`
	Bars         bool   `short:"B" long:"bars" description:"delimiter bars in ascii table"`
	Seek         int64  `short:"s" long:"seek" description:"start at <seek> bytes"`
	Len          int64  `short:"l" long:"len" description:"stop after <len> octets"`
	Columns      int    `short:"c" long:"columns" description:"column count"`
	GroupSize    int    `short:"g" long:"groups" description:"group size of bytes"`
	Plain        bool   `short:"p" long:"plain" description:"plain output without ascii table and offset row [often used with hexxy -r]"`
	Upper        bool   `short:"u" long:"upper" description:"output hex in UPPERCASE format"`
	CInclude     bool   `short:"i" long:"include" description:"output in C include format"`
	OutputFile   string `short:"o" long:"output" description:"automatically output to file instead of STDOUT"`
	Separator    string `          long:"separator" default:"|" description:"separator character for the ascii character table"`
	HasColor     string `short:"C" long:"color" default:"auto" choice:"always" choice:"auto" choice:"never" description:"this option forces color output [always|auto|never]"`
	NoColor      bool   `short:"n" long:"no-color" description:"do not print output with color"`
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
	udigits = "0123456789ABCDEF"
	ldigits = "0123456789abcdef"
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

var (
	USE_COLOR bool
)

func inputIsPipe() bool {
	stat, _ := os.Stdin.Stat()
	return stat.Mode()&os.ModeCharDevice != os.ModeCharDevice
}

func outputIsPipe() bool {
	stat, _ := os.Stdout.Stat()
	return stat.Mode()&os.ModeCharDevice != os.ModeCharDevice
}

func HexxyDump(r io.Reader, w io.Writer, filename string, color *Color) error {
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
		colFmt      int
	)

	if dumpType == dumpCformat {
		_ = copy(varDeclChar[0:14], unsignedChar[:])
		_ = copy(varDeclInt[0:16], unsignedInt[:])

		for i := 0; i < len(filename); i++ {
			if !isSpecial(filename[i]) {
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

	switch opts.OffsetFormat {
	case "d":
		colFmt = 10
	case "o":
		colFmt = 8
	case "x":
		fallthrough
	default:
		colFmt = 16
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
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("hexxy: %v", err)
		}

		// we check early on for if the dump type is "plain" (no formatting, its just a stream of bytes)
		// and we don't have to do any hard work
		if dumpType == dumpPlain && n != 0 {
			for i := 0; i < n; i++ {
				hexEncode(char, line[i:i+1], caps)
				w.Write(char)
				c++
			}
			continue
		}

		// write the line ending based on the dump "mode"
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

		if opts.Autoskip && isEmpty(&line) {
			if nulLine == 1 {
				w.Write(asterisk)
				w.Write(newLine)
			}

			nulLine++

			if nulLine > 1 {
				lineOffset++ // still increment offset while printing crunched lines with '*'
				continue
			}
		}

		// hex or binary formats only
		// writing the 0000000: part
		if dumpType <= dumpBinary {
			// create line offset
			hexOffset = strconv.AppendInt(hexOffset[0:0], lineOffset, colFmt)

			// confusing looking but we are just "slicing" our zero padding buffer and our offset byte buffer together
			// ie zeroHeader = 0000000: and hexOffset = 10 -- we are just inserting that 10 in this position '0000010:'
			// based on the offsets length
			if USE_COLOR {
				w.Write([]byte(GREY))
				w.Write(zeroHeader[0:(6 - len(hexOffset))])
				w.Write(hexOffset)
				w.Write(zeroHeader[6:])
				w.Write([]byte(CLEAR))
			} else {
				w.Write(zeroHeader[0:(6 - len(hexOffset))])
				w.Write(hexOffset)
				w.Write(zeroHeader[6:])
			}

			lineOffset++
		} else if doCheader {
			w.Write(varDeclChar)
			w.Write(newLine)
			doCheader = false
		}

		if dumpType == dumpBinary {
			// dump binary values
			for i, k := 0, octs; i < n; i, k = i+1, k+octs {
				binaryEncode(char, line[i:i+1])

				if USE_COLOR {
					for _, b := range char {
						if b == '1' {
							w.Write([]byte("\x1b[32m"))
							w.Write([]byte{b})
							w.Write([]byte(CLEAR))
						} else {
							w.Write([]byte("\x1b[34m"))
							w.Write([]byte{b})
							w.Write([]byte(CLEAR))
						}
					}
					// w.Write(char)
					c++
				} else {
					w.Write(char)
					c++
				}

				if k == octs*groupSize {
					k = 0
					w.Write(space)
				}
			}
		} else if dumpType == dumpCformat {
			// dump C format
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

				if USE_COLOR {
					i := line[i : i+1][0]
					b, c := color.Colorize2(i)
					w.Write(b)
					w.Write(char)
					w.Write(c)
				} else {
					w.Write(char)
				}
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

	if opts.NoColor || !USE_COLOR {
		color.disable = true
	}

	if !color.disable {
		color.Compute()
	}

	var (
		infile  *os.File
		outfile *os.File
		err     error
	)

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
		if err := HexxyReverse(infile, outfile); err != nil {
			return fmt.Errorf("hexxy: %v", err.Error())
		}
		return nil
	}

	if err := HexxyDump(infile, out, infile.Name(), color); err != nil {
		return fmt.Errorf("hexxy: %v", err.Error())
	}

	return nil
}

const usage_msg = `
hexxy is a command line hex dumping tool

Examples:
	hexxy [OPTIONS] input-file

	# Include a binary as a C variable
	hexxy -i input-file > output.c

	# Use plain non-formatted output
	hexxy -p input-file

	# Reverse plain non-formatted output (reverse plain)
	hexxy -rp input-file

	# Show output with a space in between N groups of bytes
	hexxy -g1 input-file ... -> outputs: 00000000: 0f 1a ff ff 00 aa

	# Seek to N bytes in an input file
	hexxy -s 12546 input-file
`

// extra usage examples
func usage() {
	fmt.Fprint(os.Stderr, usage_msg)
}

// parses the color flag and decides whether color is appropriate or not
func useColor() bool {
	// NO_COLOR spec compliance
	if HasNoColorEnvVar() {
		return false
	}

	switch strings.ToLower(opts.HasColor) {
	case "always":
		return true
	case "auto":
		if opts.NoColor {
			return false
		}

		return !outputIsPipe()
	case "never":
		return false
	}
	return false
}

func init() {
	opts.Seek = -1 // default no-op values
	opts.Columns = -1
	opts.GroupSize = -1
	opts.Len = -1
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	args, err := parser.Parse()
	if flags.WroteHelp(err) {
		usage()
		os.Exit(0)
	}
	if err != nil {
		log.Fatal(err)
	}

	// set color based on flags or default to off
	USE_COLOR = useColor()

	if !inputIsPipe() && len(args) == 0 {
		parser.WriteHelp(os.Stderr)
		fmt.Print(usage_msg)
		os.Exit(0)
	}

	if opts.Verbose {
		Debug = log.Printf
	}

	if err := Hexxy(args); err != nil {
		log.Fatal(err)
	}
}
