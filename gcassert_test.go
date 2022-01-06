package gcassert

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/packages"
)

func TestParseDirectives(t *testing.T) {
	fileSet := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedCompiledGoFiles |
			packages.NeedTypes | packages.NeedTypesInfo,
		Fset: fileSet,
	}, "./testdata")
	if err != nil {
		t.Fatal(err)
	}
	absMap, err := parseDirectives(pkgs, fileSet)
	if err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Convert the map into relative paths for ease of testing, and remove
	// the syntax node so we don't have to test that as well.
	relMap := make(directiveMap, len(absMap))
	for absPath, m := range absMap {
		for k, info := range m {
			info.n = nil
			m[k] = info
		}
		relPath, err := filepath.Rel(cwd, absPath)
		if err != nil {
			t.Fatal(err)
		}
		relMap[relPath] = m
	}

	expectedMap := directiveMap{
		"testdata/bce.go": {
			8:  {directives: []assertDirective{bce}},
			11: {directives: []assertDirective{bce, inline}},
			13: {directives: []assertDirective{bce, inline}},
			17: {directives: []assertDirective{bce, inline}},
			19: {directives: []assertDirective{bce, inline}},
		},
		"testdata/inline.go": {
			46: {directives: []assertDirective{inline}},
			50: {directives: []assertDirective{inline}},
			52: {directives: []assertDirective{inline}},
			56: {directives: []assertDirective{inline}},
			59: {directives: []assertDirective{inline}},
			61: {directives: []assertDirective{inline}},
			63: {directives: []assertDirective{inline}},
		},
		"testdata/noescape.go": {
			21: {directives: []assertDirective{noescape}},
			28: {directives: []assertDirective{noescape}},
			34: {directives: []assertDirective{noescape}},
			41: {directives: []assertDirective{noescape}},
			44: {directives: []assertDirective{noescape}},
		},
	}
	assert.Equal(t, expectedMap, relMap)
}

func TestGCAssert(t *testing.T) {
	var w strings.Builder
	err := GCAssert(&w, "./testdata", "./testdata/otherpkg")
	if err != nil {
		t.Fatal(err)
	}

	expectedOutput := `testdata/noescape.go:21:	foo := foo{a: 1, b: 2}: foo escapes to heap:
testdata/noescape.go:34:	// This annotation should fail, because f will escape to the heap.
//gcassert:noescape
func (f foo) setA(a int) *foo {
	f.a = a
	return &f
}: f escapes to heap:
testdata/noescape.go:44:	: a escapes to heap:
testdata/bce.go:8:	fmt.Println(ints[5]): Found IsInBounds
testdata/bce.go:17:	sum += notInlinable(ints[i]): call was not inlined
testdata/bce.go:19:	sum += notInlinable(ints[i]): call was not inlined
testdata/inline.go:46:	alwaysInlined(3): call was not inlined
testdata/inline.go:52:	sum += notInlinable(i): call was not inlined
testdata/inline.go:56:	sum += 1: call was not inlined
testdata/inline.go:61:	test(0).alwaysInlinedMethod(): call was not inlined
testdata/inline.go:63:	test(0).neverInlinedMethod(10): call was not inlined
testdata/inline.go:65:	otherpkg.A{}.NeverInlined(sum): call was not inlined
`
	assert.Equal(t, expectedOutput, w.String())
}
