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
	return sum
}
