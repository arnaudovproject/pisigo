package main

import (
	"fmt"
	"os"
)

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "pisigo: "+format+"\n", args...)
	os.Exit(1)
}
