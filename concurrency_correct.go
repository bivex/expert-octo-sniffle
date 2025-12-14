package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Correct goroutine patterns - should not trigger warnings

// Example 1: Proper channel usage with close
func properChannelUsage() {
	ch := make(chan int, 10)

	// Producer goroutine
	go func() {
		defer close(ch) // Proper close
		for i := 0; i < 5; i++ {
			ch <- i
		}
	}()

	// Consumer
	for val := range ch {
		fmt.Println("Received:", val)
	}
}

// Example 2: Select with context cancellation
func properSelectWithContext() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch := make(chan int)

	go func() {
		defer close(ch)
		select {
		case ch <- 42:
		case <-ctx.Done():
			return
		}
	}()

	select {
	case val := <-ch:
		fmt.Println("Got value:", val)
	case <-ctx.Done():
		fmt.Println("Timeout")
	}
}

// Example 3: Select with default case (non-blocking)
func properSelectWithDefault() {
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)

	ch1 <- 1
	ch2 <- 2

	select {
	case val := <-ch1:
		fmt.Println("From ch1:", val)
	case val := <-ch2:
		fmt.Println("From ch2:", val)
	default:
		fmt.Println("No data available")
	}
}

// Example 4: Proper mutex usage
func properMutexUsage() {
	var mu sync.Mutex
	counter := 0

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	wg.Wait()
	fmt.Println("Final counter:", counter)
}

// Example 5: Proper RWMutex usage
func properRWMutexUsage() {
	var mu sync.RWMutex
	data := make(map[string]int)

	var wg sync.WaitGroup

	// Multiple readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mu.RLock()
			if val, ok := data["key"]; ok {
				fmt.Printf("Reader %d: %d\n", id, val)
			}
			mu.RUnlock()
		}(i)
	}

	// Single writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		data["key"] = 42
		mu.Unlock()
	}()

	wg.Wait()
}

// Example 6: Worker pool pattern
func properWorkerPool() {
	const numWorkers = 3
	const numJobs = 10

	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	// Start workers
	for w := 1; w <= numWorkers; w++ {
		go func(workerID int) {
			for job := range jobs {
				fmt.Printf("Worker %d processing job %d\n", workerID, job)
				time.Sleep(100 * time.Millisecond) // Simulate work
				results <- job * 2
			}
		}(w)
	}

	// Send jobs
	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)

	// Collect results
	for r := 1; r <= numJobs; r++ {
		<-results
	}
	close(results)
}

// Example 7: Proper goroutine cleanup with done channel
func properGoroutineCleanup() {
	done := make(chan struct{})
	results := make(chan int)

	go func() {
		defer close(results)
		for i := 0; i < 5; i++ {
			select {
			case results <- i:
			case <-done:
				return
			}
		}
	}()

	// Read some results
	for i := 0; i < 3; i++ {
		if val, ok := <-results; ok {
			fmt.Println("Result:", val)
		}
	}

	// Signal done
	close(done)

	// Wait for cleanup
	for range results {
		// Drain remaining results
	}
}

// Example 8: Proper use of sync.WaitGroup
func properWaitGroup() {
	var wg sync.WaitGroup
	results := make(chan int, 10)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results <- id * id
		}(i)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	sum := 0
	for result := range results {
		sum += result
	}

	fmt.Println("Sum of squares:", sum)
}

// Example 9: Proper use of sync.Once
func properSyncOnce() {
	var once sync.Once
	var mu sync.Mutex
	counter := 0

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			once.Do(func() {
				fmt.Println("Initialization done once")
			})

			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	wg.Wait()
	fmt.Println("Counter:", counter)
}

// Example 10: Proper channel direction usage
func properChannelDirections() {
	// Bidirectional channel
	ch := make(chan int, 5)

	go func() {
		defer close(ch)
		for i := 0; i < 5; i++ {
			ch <- i
		}
	}()

	// Use as receive-only in this scope
	recvCh := (<-chan int)(ch)

	// Read from receive-only channel
	for val := range recvCh {
		fmt.Println("Received:", val)
	}
}