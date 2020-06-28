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
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/packages"
)

func TestParseDirectives(t *testing.T) {
	fileSet := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedCompiledGoFiles,
		Fset: fileSet,
	}, "./testdata")
	if err != nil {
		t.Fatal(err)
	}
	actualMap, err := parseDirectives(pkgs, fileSet)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range actualMap {
		for k, info := range m {
			info.n = nil
			m[k] = info
		}
	}

	expectedMap := directiveMap{
		"testdata/bce.go": {
			8:  {directives: []assertDirective{bce}},
			12: {directives: []assertDirective{bce, inline}},
			16: {directives: []assertDirective{bce, inline}},
		},
		"testdata/inline.go": {
			20: {directives: []assertDirective{inline}},
			22: {directives: []assertDirective{inline}},
		},
	}
	assert.Equal(t, expectedMap, actualMap)
}

func TestGCAssert(t *testing.T) {
	var w strings.Builder
	err := GCAssert("./testdata", &w)
	if err != nil {
		t.Fatal(err)
	}

	expectedOutput := `testdata/bce.go:8:	fmt.Println(ints[5]): Found IsInBounds
testdata/bce.go:16:	sum += notInlinable(ints[i]): call was not inlined
testdata/inline.go:22:	sum += notInlinable(i): call was not inlined
`
	assert.Equal(t, expectedOutput, w.String())
}
