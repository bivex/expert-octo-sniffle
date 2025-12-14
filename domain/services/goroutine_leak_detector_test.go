package services

import (
	"go/parser"
	"go/token"
	"testing"

	"goastanalyzer/domain/valueobjects"
)

func TestGoroutineLeakDetector_DetectLeaks(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedLeaks  int
		expectedTypes  []string
	}{
		{
			name: "Channel receive leak - no close",
			code: `
package main
func test() {
	ch := make(chan int)
	go func() {
		<-ch // Receive without close - leak
	}()
}`,
			expectedLeaks: 1,
			expectedTypes: []string{"channel_receive_leak"},
		},
		{
			name: "Channel receive leak - with close (no leak)",
			code: `
package main
func test() {
	ch := make(chan int)
	go func() {
		defer close(ch)
		val := <-ch
		_ = val
	}()
	ch <- 1
}`,
			expectedLeaks: 0,
			expectedTypes: []string{},
		},
		{
			name: "Select statement leak - no escape hatch",
			code: `
package main
func test() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	go func() {
		select {
		case <-ch1:
		case <-ch2:
		// No default or timeout - leak
		}
	}()
}`,
			expectedLeaks: 2,
			expectedTypes: []string{"channel_receive_leak", "select_statement_leak"},
		},
		{
			name: "Select with context cancel - no leak",
			code: `
package main
import "context"
func test() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan int)
	go func() {
		select {
		case <-ch:
		case <-ctx.Done():
			return
		}
	}()
}`,
			expectedLeaks: 0,
			expectedTypes: []string{},
		},
		{
			name: "Channel send leak - premature return",
			code: `
package main
func test() {
	ch := make(chan int)
	go func() {
		ch <- 1
		return // Premature return - potential leak
	}()
}`,
			expectedLeaks: 1,
			expectedTypes: []string{"channel_send_leak"},
		},
		{
			name: "Multiple leaks in one function",
			code: `
package main
func test() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	go func() {
		<-ch1 // Leak 1
		select {
		case <-ch2:
		// Leak 2 - no escape hatch
		}
	}()
}`,
			expectedLeaks: 2,
			expectedTypes: []string{"channel_receive_leak", "select_statement_leak"},
		},
		{
			name: "Proper goroutine - no leaks",
			code: `
package main
func test() {
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case done <- struct{}{}:
		default:
		}
	}()
	<-done
}`,
			expectedLeaks: 0,
			expectedTypes: []string{},
		},
		{
			name: "Named function goroutine - no body analysis",
			code: `
package main
func worker() {}
func test() {
	go worker() // Named function - no leak detection
}`,
			expectedLeaks: 0,
			expectedTypes: []string{},
		},
		{
			name: "Complex select with timeout - no leak",
			code: `
package main
import "time"
func test() {
	ch := make(chan int)
	go func() {
		select {
		case <-ch:
		case <-time.After(time.Second):
			return
		}
	}()
}`,
			expectedLeaks: 0,
			expectedTypes: []string{},
		},
		{
			name: "Channel send with proper cleanup - no leak",
			code: `
package main
func test() {
	ch := make(chan int)
	done := make(chan struct{})
	go func() {
		defer close(ch)
		select {
		case ch <- 1:
		case <-done:
			return
		}
	}()
	close(done)
}`,
			expectedLeaks: 0,
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

			detector := NewASTGoroutineLeakDetector()
			config, _ := valueobjects.NewAnalysisConfiguration(10, 15, 50, true, valueobjects.SeverityWarning)
			findings, err := detector.DetectLeaks(node, fset, config)
			if err != nil {
				t.Fatalf("DetectLeaks failed: %v", err)
			}

			if len(findings) != tt.expectedLeaks {
				t.Errorf("Expected %d leaks, got %d", tt.expectedLeaks, len(findings))
				for i, finding := range findings {
					t.Logf("Finding %d: %s", i, finding.Message())
				}
			}

			// Check that expected leak types are found
			for _, expectedType := range tt.expectedTypes {
				found := false
				for _, finding := range findings {
					if finding.Message() != "" && stringContains(finding.Message(), expectedType) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find leak type %s, but didn't", expectedType)
				}
			}
		})
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}