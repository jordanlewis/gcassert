package otherpkg

import "fmt"

type A struct{}

//gcassert:inline
func (a A) NeverInlined(n int) {
	for i := 0; i < n; i++ {
		fmt.Println(i)
	}
}

//gcassert:inline
func NeverInlinedFunc(n int) {
	for i := 0; i < n; i++ {
		fmt.Println(i)
	}
}
