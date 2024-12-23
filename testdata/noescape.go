package gcassert

type foo struct {
	a int
	b int
}

func returnsStackVarPtr() *foo {
	// this should fail
	//gcassert:noescape
	foo := foo{a: 1, b: 2}
	return &foo
}

func returnsStackVar() foo {
	// this should succeed
	//gcassert:noescape
	foo := foo{a: 1, b: 2}
	return foo
}

// This annotation should fail, because f will escape to the heap.
//
//gcassert:noescape
func (f foo) setA(a int) *foo {
	f.a = a
	return &f
}

// This annotation should pass, because f does not escape.
//
//gcassert:noescape
func (f foo) returnA(
	// This annotation should fail, because a will escape to the heap.
	//gcassert:noescape
	a int,
	b int,
) *int {
	return &a
}
