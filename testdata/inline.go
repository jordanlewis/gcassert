package gcassert

import "fmt"

func inlinable(a int) int {
	return a + 2
}

func notInlinable(a int) int {
	for i := 0; i < a; i++ {
		fmt.Println(i)
	}
	return 0
}

func caller() {
	sum := 0
	for i := 0; i < 10; i++ {
		// gcassert:inline
		sum += inlinable(i)
		// gcassert:inline
		sum += notInlinable(i)
	}
}
