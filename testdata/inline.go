package gcassert

import (
	"fmt"
	"math/bits"

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

	if bits.UintSize == 64 {
		sum += test(0).alwaysInlinedMethod()
	} else {
		// placeholder
	}
	sum += test(0).neverInlinedMethod(10)

	otherpkg.A{}.NeverInlined(sum)
}
