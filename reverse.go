package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// func reverse(w io.Writer, path string) error {
func reverse(w io.Writer, f *os.File) error {
	// f, err := os.Open(path)
	// if err != nil {
	// 	return err
	// }
	// defer f.Close()
	s := bufio.NewScanner(f)

	star := false
	var prev uint64
	var data []byte
	var zero [16]byte

	for s.Scan() {
		line := s.Text()
		if line == "*" {
			star = true
			continue
		}

		if len(line) < len("00000000") {
			return fmt.Errorf("invalid line %q, missing address prefix", line)
		}

		part := line[:len("00000000")]
		line = line[len("00000000"):]

		addr, err := strconv.ParseUint(part, 16, 32)
		if err != nil {
			return err
		}

		if star {
			for i := prev + 16; i < addr; i += 16 {
				data = append(data, zero[:]...)
			}
			star = false
		}

		prev = addr
		pos := strings.IndexByte(line, '|')

		if pos != -1 {
			line = line[:pos]
		}

		for len(line) > 0 {
			line = strings.TrimSpace(line)
			pos := strings.IndexByte(line, ' ')
			if pos == -1 {
				pos = len(line)
			}

			part := line[:pos]
			line = line[pos:]

			b, err := strconv.ParseUint(part, 16, 8)
			if err != nil {
				return err
			}

			data = append(data, byte(b))
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	if _, err := w.Write(data); err != nil {
		return err
	}
	return nil
}
