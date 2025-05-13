package main

import "fmt"

func main() {
	s1 := []int{1, 2, 3, 4, 5}
	s2 := append(s1[:0:0], s1...)

	s1[3] = 99

	fmt.Println(s1)
	fmt.Println(s2)
}
