package main

import "fmt"

func twoSum(nums []int, target int) []int {
	m := map[int]int{}

	for i, v := range nums {
		ind, ok := m[target-v]
		if ok {
			return []int{ind, i}
		}

		m[v] = i
	}

	panic("no solution")
}

func main() {
	result := twoSum([]int{2, 7, 11, 15}, 9)
	fmt.Println(result)
}
