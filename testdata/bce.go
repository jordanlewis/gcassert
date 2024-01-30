package gcassert

import "fmt"

func aLoop(ints []int) int {
	sum := 0
	//gcassert:bce
	fmt.Println(ints[5])
	for i := range ints {
		//gcassert:bce,inline
		sum += inlinable(ints[i])

		sum += inlinable(ints[i]) //gcassert:bce,inline

		//gcassert:bce
		//gcassert:inline
		sum += notInlinable(ints[i])

		sum += notInlinable(ints[i]) //gcassert:bce,inline
	}
	// N.B. The statement on line 8 yields 'IsInBounds' check since we can't prove the slice has at least 6 elements.
        // Thus, the statement below yields 'IsSliceInBounds' check since we also can't prove it has at least 7 elements.
        fmt.Println(ints[1:7]) //gcassert:bce
	return sum
}
