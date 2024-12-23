package gcassert

//gcassert:foo
func badDirective1() {}

func badDirective2() {
	//gcassert:bce,bar,inline
	badDirective1()
}

//gcassert:inline,afterinline
func badDirective3() {
	badDirective2()
}
