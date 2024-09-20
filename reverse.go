package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

func HexxyReverse(r io.Reader, w io.Writer) error {
	var (
		cols int
		octs int
		char = make([]byte, 1)
	)

	if opts.Columns != -1 {
		cols = opts.Columns
	}

	switch dumpType {
	case dumpBinary:
		octs = 8
	case dumpCformat:
		octs = 4
	default:
		octs = 2
	}

	if opts.Len != -1 {
		if opts.Len < int64(cols) {
			cols = int(opts.Len)
		}
	}

	if octs < 1 {
		octs = cols
	}

	// character count
	c := int64(0)
	rd := bufio.NewReader(r)
	for {
		// TODO this is causing issues with plain
		line, err := rd.ReadBytes('\n')
		n := len(line)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("hexxy: %v", err)
		}

		if n == 0 {
			return nil
		}

		if dumpType == dumpHex {
			// line = line[9:48]
			line = line[9 : len(line)-19]
			n := len(line)

			for i := 0; i <= n; {
				// print(string(line[i : i+4]))

				if rv, _ := hexDecode(char, line[i:i+octs]); rv != 0 {
					w.Write(char)
				}

				i += 5
				// time.Sleep(time.Millisecond * 500)
			}

			// for i := 0; n >= octs+1; {
			// 	print(string(line[i : i+octs]))
			// 	time.Sleep(time.Millisecond * 500)
			// 	if rv, _ := hexDecode(char, line[i:i+octs]); rv != 0 {
			// 		w.Write(char)
			// 		i += 2
			// 		n -= 2
			// 		c++
			// 	} else if rv == -1 {
			// 		i++
			// 		n--
			// 	} else {
			// 		// rv == -2
			// 		i += 2
			// 		n -= 2
			// 	}
			// }
		} else if dumpType == dumpBinary {
			for i := 0; n >= octs; {
				if binaryDecode(char, line[i:i+octs]) != -1 {
					i++
					n--
					continue
				} else {
					w.Write(char)
					i += 8
					n -= 8
					c++
				}
			}
		} else if dumpType == dumpPlain {
			for i := 0; n >= octs; i++ {
				if rv, _ := hexDecode(char, line[i:i+octs]); rv != 0 {
					w.Write(char)
					c++
				}
				n--
			}
		} else if dumpType == dumpCformat {
			for i := 0; n >= octs; {
				if rv, _ := hexDecode(char, line[i:i+octs]); rv == 0 {
					w.Write(char)
					i += 4
					n -= 4
					c++
				} else if rv == -1 {
					i++
					n--
				} else { // rv == -2
					i += 2
					n -= 2
				}
			}
		}

		if c == int64(cols) && cols > 0 {
			return nil
		}
	}
}
