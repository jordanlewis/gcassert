// Copyright 2020 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package gcassert

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type assertDirective int

const (
	noDirective assertDirective = iota
	inline
	bce
)

func stringToDirective(s string) (assertDirective, error) {
	switch s {
	case "inline":
		return inline, nil
	case "bce":
		return bce, nil
	}
	return noDirective, errors.New(fmt.Sprintf("no such directive %s", s))
}

type lineInfo struct {
	n          ast.Node
	directives []assertDirective
	// passedDirective is a map from index into the directives slice to a
	// boolean that says whether or not the directive succeeded, in the case
	// of directives like inlining that have compiler output if they passed.
	// For directives like bce that have compiler output if they failed, there's
	// no entry in this map.
	passedDirective map[int]bool
}

var gcAssertRegex = regexp.MustCompile(`// gcassert:(\w+)`)

type assertVisitor struct {
	commentMap ast.CommentMap

	directiveMap map[int]lineInfo
	fileSet      *token.FileSet
}

func newAssertVisitor(commentMap ast.CommentMap, fileSet *token.FileSet) assertVisitor {
	return assertVisitor{
		commentMap:   commentMap,
		fileSet:      fileSet,
		directiveMap: make(map[int]lineInfo),
	}
}

func (v assertVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil {
		return w
	}
	m := v.commentMap[node]
COMMENTLOOP:
	for _, g := range m {
		for _, c := range g.List {
			matches := gcAssertRegex.FindStringSubmatch(c.Text)
			if len(matches) == 0 {
				continue COMMENTLOOP
			}
			// The 0th match is the whole string, and the 1st match is the
			// gcassert directive.

			directive, err := stringToDirective(matches[1])
			if err != nil {
				continue COMMENTLOOP
			}
			pos := node.Pos()
			lineNumber := v.fileSet.Position(pos).Line
			lineInfo := v.directiveMap[lineNumber]
			lineInfo.directives = append(lineInfo.directives, directive)
			lineInfo.n = node
			v.directiveMap[lineNumber] = lineInfo
		}
	}
	return v
}

func GCAssert(path string, w io.Writer) {
	fileSet := token.NewFileSet()
	packageMap, err := parser.ParseDir(fileSet, path, nil /* filter */, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	directiveMap := parseDirectives(packageMap, fileSet)

	// Next: invoke Go compiler with -m flags to get the compiler to print
	// its optimization decisions.

	args := append([]string{"build", "-gcflags=all=-m -m -d=ssa/check_bce/debug=1"}, "./"+path)
	cmd := exec.Command("go", args...)
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	cmd.Dir = cwd
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	cmdErr := make(chan error, 1)
	go func() {
		cmdErr <- cmd.Run()
		pw.Close()
	}()

	scanner := bufio.NewScanner(pr)
	optInfo := regexp.MustCompile(`([\.\/\w]+):(\d+):\d+: (.*)`)
	boundsCheck := "Found IsInBounds"
	sliceBoundsCheck := "Found SliceIsInBounds"

	for scanner.Scan() {
		line := scanner.Text()
		matches := optInfo.FindStringSubmatch(line)
		if len(matches) != 0 {
			filepath := matches[1]
			lineNo, err := strconv.Atoi(matches[2])
			if err != nil {
				panic(err)
			}
			message := matches[3]

			if lineToDirectives := directiveMap[filepath]; lineToDirectives != nil {
				info := lineToDirectives[lineNo]
				if info.passedDirective == nil {
					info.passedDirective = make(map[int]bool)
					lineToDirectives[lineNo] = info
				}
				for i, d := range info.directives {
					switch d {
					case bce:
						if message == boundsCheck || message == sliceBoundsCheck {
							// Error! We found a bounds check where the user expected
							// there to be none.
							// Print out the user's code lineNo that failed the assertion,
							// the assertion itself, and the compiler output that
							// proved that the assertion failed.
							printAssertionFailure(fileSet, info, w, message)
						}
					case inline:
						if strings.HasPrefix(message, "inlining call to") {
							info.passedDirective[i] = true
						}
					}
				}
			}
		}
	}

	for _, lineToDirectives := range directiveMap {
		for _, info := range lineToDirectives {
			for i, d := range info.directives {
				// An inlining directive passes if it has compiler output. For
				// each inlining directive, check if there was matching compiler
				// output and fail if not.
				if d == inline {
					if !info.passedDirective[i] {
						printAssertionFailure(fileSet, info, w, "call was not inlined")
					}
				}
			}
		}
	}

	// Finally: for every optimization decision that matches a line of
	// code that we found earlier, decide whether the assertion for that
	// line passes or fails.
}

func printAssertionFailure(fileSet *token.FileSet, info lineInfo, w io.Writer, message string) {
	var buf strings.Builder
	_ = printer.Fprint(&buf, fileSet, info.n)
	pos := fileSet.Position(info.n.Pos())
	fmt.Fprintf(w, "%s:%d:\t%s: %s\n", pos.Filename, pos.Line, buf.String(), message)
}

type directiveMap map[string]map[int]lineInfo

func parseDirectives(packageMap map[string]*ast.Package, fileSet *token.FileSet) directiveMap {
	fileDirectiveMap := make(directiveMap)
	for _, pkg := range packageMap {
		for absPath, file := range pkg.Files {
			commentMap := ast.NewCommentMap(fileSet, file, file.Comments)

			v := newAssertVisitor(commentMap, fileSet)
			// First: find all lines of code annotated with our gcassert directives.
			ast.Walk(v, file)

			fileDirectiveMap[absPath] = v.directiveMap
		}
	}
	return fileDirectiveMap
}
