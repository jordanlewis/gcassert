package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jordanlewis/gcassert"
)

func main() {
	flag.Parse()
	packagePath := flag.Arg(0)
	var buf strings.Builder
	err := gcassert.GCAssert(packagePath, &buf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	output := buf.String()
	if len(output) != 0 {
		fmt.Fprint(os.Stderr, output)
		os.Exit(1)
	}
}
