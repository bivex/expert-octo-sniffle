package main

func simpleFunction(x int) int {
	if x > 0 {
		return x * 2
	}
	return 0
}

func complexFunction(n int) int {
	result := 0
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			if i%3 == 0 {
				result += i
			}
		} else {
			result -= i
		}
	}
	return result
}
