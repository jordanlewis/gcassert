package gcassert

import (
	"bufio"
	"errors"
	"fmt"
	"go/types"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// getBazelBuildCmd expects packages relative path from the workspace root
// and returns the command to build the given packages using Bazel with
// the right flags to get the compiler to print its optimization decisions.
func getBazelBuildCmd(pkgs []string) *exec.Cmd {
	args := make([]string, 0, len(pkgs)+1)
	args = append(args, "build")
	for i := range pkgs {
		if pkg := strings.TrimSpace(pkgs[i]); pkg != "" {
			args = append(args, pkg)
		}
	}
	args = append(
		args,
		"--@io_bazel_rules_go//go/config:gc_goopts=-m=2,-d=ssa/check_bce/debug=1",
	)
	return exec.Command("bazel", args...)
}

func getBazelInfo(info string) (string, error) {
	args := []string{"info", info, "--color=no"}
	cmd := exec.Command("bazel", args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s - %s", err, errBuf.String())
	}
	return strings.TrimSpace(outBuf.String()), nil
}

func getFnReceiver(obj types.Object) string {
	var receiver string
	sig := obj.Type().(*types.Signature)
	if recv := sig.Recv(); recv != nil {
		receiver = recv.Type().String()
	}
	// Example receiver: `*github.com/cockroachdb/cockroach/pkg/sql/colfetcher.cFetcher`.
	// Truncate receiver by removing package path.
	splitAt := strings.LastIndex(receiver, ".")
	truncatedReceiver := receiver[splitAt+1:]
	if strings.HasPrefix(receiver, "*") {
		// Re-add '*' for pointer receivers such as the example above.
		return "*" + truncatedReceiver
	}
	return truncatedReceiver
}

func getFnCallIdentifierInObjectFile(obj types.Object) string {
	fnCallIdentifierInObjectFile := obj.Pkg().Path()
	receiver := getFnReceiver(obj)
	if receiver != "" {
		return fmt.Sprintf("%s.(%s).%s", fnCallIdentifierInObjectFile, receiver, obj.Name())
	}
	return fmt.Sprintf("%s.%s", fnCallIdentifierInObjectFile, obj.Name())
}

// assertInlineByInspectingObjectFiles disassembles machine code and searches
// through the text symbols for calls to functions that must be inlined.
func assertInlineByInspectingObjectFiles(w io.Writer, paths []string) error {
	var bazelBin string
	var err error
	if bazelBin, err = getBazelInfo("bazel-bin"); err != nil {
		return err
	}
	var workspacePath string
	if workspacePath, err = getBazelInfo("workspace"); err != nil {
		return err
	}

	pr, pw := io.Pipe()
	var f *os.File
	var mw io.Writer
	if debugMode {
		// Create a temp file to log all diagnostic output.
		var err error
		f, err = os.CreateTemp("", "gcassert-object-code-inspection*.log")
		if err != nil {
			return err
		}
		mw = io.MultiWriter(pw, f)
		defer fmt.Printf("See %s for full output.\n", f.Name())
	}

	disassembleCmdError := make(chan error, 1)
	go func() {
		defer func() {
			_ = pw.Close()
			if f != nil {
				_ = f.Close()
			}
		}()
		for _, path := range paths {
			getPackageName := func(path string) string {
				return path[strings.LastIndex(path, "/")+1:]
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				disassembleCmdError <- err
				return
			}
			relPkgPath, err := filepath.Rel(workspacePath, absPath)
			if err != nil {
				disassembleCmdError <- err
				return
			}
			objectFilePath := filepath.Join(bazelBin, relPkgPath, getPackageName(path)+".a")
			args := []string{"tool", "objdump", objectFilePath}
			objectFileDisassembleCmd := exec.Command("go", args...)
			if debugMode {
				objectFileDisassembleCmd.Stdout = mw
				objectFileDisassembleCmd.Stderr = mw
				// Log full 'objdump' command.
				fmt.Fprintln(f, objectFileDisassembleCmd)
			} else {
				objectFileDisassembleCmd.Stdout = pw
				objectFileDisassembleCmd.Stderr = pw
			}

			if err := objectFileDisassembleCmd.Run(); err != nil {
				disassembleCmdError <- err
				return
			}
		}
		disassembleCmdError <- nil
	}()

	scanner := bufio.NewScanner(pr)

	// callRegex is used to read a function identifier from a CALL
	// line in the object file. Example call line from the object file:
	/*
	  mvcc.go:1135	0x66928a	94000000	CALL 0(PC)	[0:4]R_CALLARM64:github.com/cockroachdb/cockroach/pkg/storage.(*MVCCGetOptions).validate

	*/
	callRegex, err := regexp.Compile(`\s*(\S+)\s*\S*\s*\S*\s*CALL\s*\S*\s*(\S+)`)
	if err != nil {
		return err
	}
	// atleastMatchedOnce ensures that we are alerted if the regex is broken due to
	// architecture / go version changes that change the structure of the
	// disassembled object file.
	var atleastMatchedOnce bool
	for scanner.Scan() {
		line := scanner.Text()
		fnIdentifierMatch := callRegex.FindStringSubmatch(line)
		if fnIdentifierMatch != nil {
			atleastMatchedOnce = true
			// See example call above for details on why this manipulation works.
			currentFn := fnIdentifierMatch[2][strings.LastIndex(fnIdentifierMatch[2], ":")+1:]
			if _, ok := bazelInlineSites[currentFn]; ok {
				fileNameAndLineNum := fnIdentifierMatch[1]
				message := "call was not inlined"
				fmt.Fprintf(w, "%s: %s\t%s\n", message, fileNameAndLineNum, currentFn)
			}
		}
	}

	if err := <-disassembleCmdError; err != nil {
		// If 'objdump' failed, return the error.
		return err
	}

	if !atleastMatchedOnce {
		errorMsg := `callRegex did not match any line in the disassembled object code.
This means it did not find any non-inlined function calls in
the inspected object files. Ensure that the regex is still correct if this
is not expected`
		if strings.Contains(workspacePath, "github.com/cockroachdb/cockroach") {
			// Only fail under cockroach workspace, otherwise print a warning.
			return errors.New(errorMsg)
		} else {
			log.Printf("WARNING: %s\n", errorMsg)
		}
	}
	return nil
}
