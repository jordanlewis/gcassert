package gcassert

func bar() {
	Gen().Layout()
}

func Gen() S {
	return S{}
}

// This assertion should fail, because it's not an inlineable function. This is
// a regression test to assert that it does fail even though the line
// Gen().Layout() has another inlined function in it, Gen().

//gcassert:inline
func (s S) Layout() {
	select {}
}

type S struct{}
