package main

import (
	"fmt"
	"sync"
	"time"
)

// Test case 1: Channel receive leak - goroutine that receives but never closes
func testChannelReceiveLeak() {
	ch := make(chan int)

	go func() {
		for {
			<-ch // Receive without close - potential leak
		}
	}()

	ch <- 1
	time.Sleep(time.Millisecond * 100)
}

// Test case 2: Select statement leak - select without context cancel or timeout
func testSelectStatementLeak() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		select {
		case <-ch1:
			fmt.Println("Received from ch1")
		case <-ch2:
			fmt.Println("Received from ch2")
		// No default case or timeout - potential leak
		}
	}()

	ch1 <- 1
	time.Sleep(time.Millisecond * 100)
}

// Test case 3: Channel send leak - premature return with send
func testChannelSendLeak() {
	ch := make(chan int)

	go func() {
		defer func() {
			ch <- 1 // Send after potential panic
		}()

		if true {
			return // Premature return before close
		}
	}()

	val := <-ch
	fmt.Println("Received:", val)
}

// Test case 4: Blocking bug - channel misuse
func testBlockingBug() {
	ch := make(chan int, 1)

	go func() {
		ch <- 1
		ch <- 2 // Second send may block if buffer is full
		ch <- 3 // Third send will definitely block
	}()

	time.Sleep(time.Millisecond * 100)
}

// Test case 5: Race condition - unprotected shared variable access
var sharedCounter int

func testRaceCondition() {
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sharedCounter++ // Race condition - no mutex protection
		}()
	}

	wg.Wait()
	fmt.Println("Final counter:", sharedCounter)
}

// Test case 6: Deadlock potential - incorrect mutex usage
func testDeadlockPotential() {
	var mu sync.Mutex

	go func() {
		mu.Lock()
		// Do something
		// mu.Unlock() // Commented out - potential deadlock
	}()

	mu.Lock()
	fmt.Println("This may deadlock")
	mu.Unlock()
}

// Test case 7: Goroutine with proper cleanup (should not trigger warnings)
func testProperGoroutine() {
	ch := make(chan int)
	done := make(chan struct{})

	go func() {
		defer close(ch)
		for {
			select {
			case <-done:
				return
			default:
				time.Sleep(time.Millisecond)
			}
		}
	}()

	close(done)
	<-ch // Wait for cleanup
}