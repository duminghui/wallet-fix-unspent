// +build !appengine,!js

package main

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

func checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}
