package main
func test() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	select {
	case <-ch1:
	case <-ch2:
	}
}
