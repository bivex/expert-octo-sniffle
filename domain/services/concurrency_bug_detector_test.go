package services

import (
	"go/parser"
	"go/token"
	"testing"

	"goastanalyzer/domain/valueobjects"
)

func TestConcurrencyBugDetector_DetectBugs(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedBugs  int
		expectedTypes []string
	}{
		{
			name: "Channel blocking bug - unbalanced operations",
			code: `
package main
func test() {
	ch := make(chan int)
	ch <- 1
	ch <- 2
	ch <- 3
	ch <- 4
	ch <- 5
	ch <- 6
	ch <- 7
	ch <- 8
	ch <- 9
	ch <- 10
	ch <- 11
	ch <- 12
	ch <- 13
	ch <- 14
	ch <- 15
	<-ch // Only one receive - potential blocking
}`,
			expectedBugs: 1,
			expectedTypes: []string{"channel misuse"},
		},
		{
			name: "Mutex deadlock - lock without unlock",
			code: `
package main
import "sync"
func test() {
	var mu sync.Mutex
	mu.Lock()
	// mu.Unlock() // Missing unlock - deadlock
}`,
			expectedBugs: 1,
			expectedTypes: []string{"deadlock"},
		},
		{
			name: "Select without escape hatch - blocking",
			code: `
package main
func test() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	select {
	case <-ch1:
	case <-ch2:
	// Two cases, no default - can block indefinitely
	}
}`,
			expectedBugs: 1,
			expectedTypes: []string{"channel misuse"},
		},
		{
			name: "Proper mutex usage - no bugs",
			code: `
package main
import "sync"
func test() {
	var mu sync.Mutex
	var counter int
	mu.Lock()
	counter++
	mu.Unlock()
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Select with default - no blocking",
			code: `
package main
func test() {
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	select {
	case val := <-ch1:
		_ = val
	case val := <-ch2:
		_ = val
	default:
		// Non-blocking
	}
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Balanced channel operations - no bugs",
			code: `
package main
func test() {
	ch := make(chan int, 5)
	for i := 0; i < 3; i++ {
		ch <- i
	}
	for i := 0; i < 3; i++ {
		<-ch
	}
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Multiple mutex operations - deadlock detection",
			code: `
package main
import "sync"
func test() {
	var mu sync.Mutex
	mu.Lock()
	mu.Lock() // Double lock - deadlock
	mu.Unlock()
}`,
			expectedBugs: 1,
			expectedTypes: []string{"deadlock"},
		},
		{
			name: "Proper WaitGroup usage - no bugs",
			code: `
package main
import "sync"
func test() {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// work
		}()
	}
	wg.Wait()
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Complex blocking scenario",
			code: `
package main
func test() {
	ch1 := make(chan int)
	ch2 := make(chan int, 1)
	ch3 := make(chan int, 1)

	ch2 <- 1
	ch3 <- 2

	// This select should not block
	select {
	case <-ch1:
	case val := <-ch2:
		_ = val
	case val := <-ch3:
		_ = val
	default:
	}
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Mutex in goroutine - no race condition detection (disabled)",
			code: `
package main
import "sync"
func test() {
	var mu sync.Mutex
	counter := 0
	go func() {
		mu.Lock()
		counter++
		mu.Unlock()
	}()
	mu.Lock()
	counter++
	mu.Unlock()
}`,
			expectedBugs: 0, // Race detection disabled for now
			expectedTypes: []string{},
		},
		{
			name: "Channel operations in loop - balanced",
			code: `
package main
func test() {
	ch := make(chan int, 10)
	go func() {
		for i := 0; i < 5; i++ {
			ch <- i
		}
		close(ch)
	}()
	sum := 0
	for val := range ch {
		sum += val
	}
	_ = sum
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Extreme channel imbalance",
			code: `
package main
func test() {
	ch := make(chan int)
	ch <- 1
	ch <- 2
	ch <- 3
	ch <- 4
	ch <- 5
	ch <- 6
	ch <- 7
	ch <- 8
	ch <- 9
	ch <- 10
	ch <- 11
	ch <- 12
	ch <- 13
	<-ch // Only one receive
}`,
			expectedBugs: 1,
			expectedTypes: []string{"channel misuse"},
		},
		{
			name: "Proper sync.Once usage",
			code: `
package main
import "sync"
func test() {
	var once sync.Once
	var counter int
	for i := 0; i < 10; i++ {
		go func() {
			once.Do(func() {
				counter++
			})
		}()
	}
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Select with timeout - no blocking",
			code: `
package main
import "time"
func test() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	select {
	case <-ch1:
	case <-ch2:
	case <-time.After(time.Millisecond):
		// Timeout prevents blocking
	}
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Worker pool pattern - correct",
			code: `
package main
func test() {
	jobs := make(chan int, 10)
	results := make(chan int, 10)

	for w := 1; w <= 3; w++ {
		go func(id int) {
			for job := range jobs {
				results <- job * 2
			}
		}(w)
	}

	for j := 1; j <= 5; j++ {
		jobs <- j
	}
	close(jobs)

	for r := 1; r <= 5; r++ {
		<-results
	}
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Context cancellation pattern",
			code: `
package main
import "context"
func test() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch1 := make(chan int)
	ch2 := make(chan int)
	go func() {
		select {
		case ch1 <- 42:
		case <-ctx.Done():
			return
		}
	}()

	select {
	case val := <-ch1:
		_ = val
	case val := <-ch2:
		_ = val
	case <-ctx.Done():
	}
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
		{
			name: "Complex deadlock scenario",
			code: `
package main
import "sync"
func test() {
	var mu sync.Mutex
	mu.Lock()
	go func() {
		mu.Lock() // Double lock - deadlock
		mu.Unlock()
	}()
	// mu.Unlock() // Missing unlock - deadlock
}`,
			expectedBugs: 1,
			expectedTypes: []string{"deadlock"},
		},
		{
			name: "Channel close after send",
			code: `
package main
func test() {
	ch := make(chan int)
	go func() {
		ch <- 1
		ch <- 2
		close(ch)
	}()
	for val := range ch {
		_ = val
	}
}`,
			expectedBugs: 2,
			expectedTypes: []string{"channel misuse"},
		},
		{
			name: "Empty function - no bugs",
			code: `
package main
func test() {
	// Empty function
}`,
			expectedBugs: 0,
			expectedTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			detector := NewASTConcurrencyBugDetector()
			config, _ := valueobjects.NewAnalysisConfiguration(10, 15, 50, true, valueobjects.SeverityWarning)
			findings, err := detector.DetectBugs(node, fset, config)
			if err != nil {
				t.Fatalf("DetectBugs failed: %v", err)
			}

			if len(findings) != tt.expectedBugs {
				t.Errorf("Expected %d bugs, got %d", tt.expectedBugs, len(findings))
				for i, finding := range findings {
					t.Logf("Finding %d: %s", i, finding.Message())
				}
			}

			// Check that expected bug types are found
			for _, expectedType := range tt.expectedTypes {
				found := false
				for _, finding := range findings {
					if finding.Message() != "" && stringContainsBug(finding.Message(), expectedType) {
						found = true
						break
					}
				}
				if !found && len(tt.expectedTypes) > 0 {
					t.Errorf("Expected to find bug type %s, but didn't", expectedType)
				}
			}
		})
	}
}

func stringContainsBug(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}