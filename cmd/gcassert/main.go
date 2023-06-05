package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jordanlewis/gcassert"
)

var useBazelFlag = flag.Bool("use-bazel", false, "use bazel to build")
var debugFlag = flag.Bool("debug", false, "write logs to files")

func main() {
	flag.Parse()
	var buf strings.Builder
	err := gcassert.GCAssert(&buf, *useBazelFlag, *debugFlag, flag.Args()...)
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
