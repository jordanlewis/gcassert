package gcassert

import "fmt"

func aLoop(ints []int) int {
	sum := 0
	// gcassert:bce
	fmt.Println(ints[5])
	for i := range ints {
		// gcassert:bce
		// gcassert:inline
		sum += inlinable(ints[i])

		// gcassert:bce
		// gcassert:inline
		sum += notInlinable(ints[i])
	}
	return sum
}
