package gcassert

import (
	"fmt"

	"github.com/jordanlewis/gcassert/testdata/otherpkg"
)

func inlinable(a int) int {
	return a + 2
}

func notInlinable(a int) int {
	for i := 0; i < a; i++ {
		fmt.Println(i)
	}
	return 0
}

type test int

//gcassert:inline
func (t test) alwaysInlinedMethod() int {
	return 0
}

//gcassert:inline
func (t test) neverInlinedMethod(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		fmt.Println(i)
	}
	return sum
}

// This assertion makes sure that every callsite to alwaysInlined is in fact
// inlined.
//gcassert:inline
//go:noinline
func alwaysInlined(a int) int {
	return a + a
}

func caller() {
	alwaysInlined(3)
	sum := 0
	for i := 0; i < 10; i++ {
		//gcassert:inline
		sum += inlinable(i)
		//gcassert:inline
		sum += notInlinable(i)
	}

	// This assertion should fail as there's nothing to inline.
	sum += 1 //gcassert:inline

	sum += test(0).alwaysInlinedMethod()
	sum += test(0).neverInlinedMethod(10)

	otherpkg.A{}.NeverInlined(sum)

	otherpkg.NeverInlinedFunc(sum)

	//gcassert:inline
	anonInlined := func(a int) int {
		return a + 10
	}

	anonInlinedNoAssert := func(a int) int {
		return a + 10
	}

	//gcassert:inline
	anonNeverInlined := func(a int) int {
		fmt.Println(a)
		return a + 10
	}

	anonNeverInlinedNoAssert := func(a int) int {
		fmt.Println(a)
		return a + 10
	}

	for i := 0; i < 10; i++ {
		sum += anonInlined(i)
		sum += anonNeverInlined(i)
		//gcassert:inline
		sum += anonInlinedNoAssert(i)
		//gcassert:inline
		sum += anonNeverInlinedNoAssert(i)
	}

	// The assertion for the anonymous function assigned to baz below does not
	// apply to this call of the named function baz, which cannot be inlined and
	// has no inline assertion.
	sum += baz(1)

	//gcassert:inline
	baz := func(a int) int {
		return a + 10
	}
	sum += baz(1)

	{
		// The assertion for the baz in the parent scope does not apply to this
		// baz, which cannot be inlined and has no assertion.
		baz := func(a int) int {
			fmt.Println(a)
			return a + 10
		}
		sum += baz(1)
	}
}

//go:noinline
func baz(a int) int {
	return a + 10
}
