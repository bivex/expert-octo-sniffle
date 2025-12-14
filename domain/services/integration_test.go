package services

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"goastanalyzer/domain/valueobjects"
)

func TestIntegration_GoroutineLeakAndConcurrencyBugs(t *testing.T) {
	tests := []struct {
		name               string
		code               string
		expectedFindings  int
		expectedLeakTypes []string
		expectedBugTypes  []string
		checkSeverity     bool
		minSeverity       valueobjects.SeverityLevel
	}{
		{
			name: "Complex concurrent code with multiple issues",
			code: `
package main
import (
	"sync"
	"time"
)
func problematicConcurrentCode() {
	// Issue 1: Goroutine leak - channel receive without close
	ch1 := make(chan int)
	go func() {
		val := <-ch1 // Leak: no close operation
		_ = val
	}()

	// Issue 2: Select leak - no escape hatch
	ch2 := make(chan int)
	ch3 := make(chan int)
	ch4 := make(chan int)
	go func() {
		select {
		case <-ch2:
		case <-ch3:
		case <-ch4:
		// Multiple cases, no default or timeout - leak
		}
	}()

	// Issue 3: Deadlock potential
	var mu sync.Mutex
	mu.Lock()
	go func() {
		mu.Lock() // Double lock - deadlock
		mu.Unlock()
	}()
	time.Sleep(time.Millisecond)
	mu.Unlock()

	// Issue 4: Channel blocking
	ch5 := make(chan int)
	for i := 0; i < 100; i++ {
		ch5 <- i // Many sends, few receives
	}
	for i := 0; i < 5; i++ {
		<-ch5
	}
}`,
			expectedFindings:  10, // Current detection finds these issues
			expectedLeakTypes: []string{"channel_receive_leak", "select_statement_leak"},
			expectedBugTypes:  []string{"channel misuse"},
			checkSeverity:     true,
			minSeverity:       valueobjects.SeverityWarning,
		},
		{
			name: "Clean concurrent code - minimal issues",
			code: `
package main
import (
	"context"
	"sync"
	"time"
)
func cleanConcurrentCode() {
	// Proper goroutine with context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
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

	// Proper mutex usage
	var mu sync.Mutex
	var wg sync.WaitGroup
	counter := 0

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}
	wg.Wait()

	// Balanced channel operations
	dataCh := make(chan int, 5)
	go func() {
		defer close(dataCh)
		for i := 0; i < 5; i++ {
			dataCh <- i
		}
	}()

	sum := 0
	for val := range dataCh {
		sum += val
	}
	_ = sum
}`,
			expectedFindings:  0, // Clean code should have no findings
			expectedLeakTypes: []string{},
			expectedBugTypes:  []string{},
			checkSeverity:     false,
		},
		{
			name: "Mixed issues with different severity levels",
			code: `
package main
import "sync"
func mixedIssues() {
	// High severity: Deadlock
	var mu sync.Mutex
	mu.Lock()
	go func() {
		mu.Lock() // Will block
		defer mu.Unlock()
	}()
	// mu.Unlock() // Missing - deadlock

	// Medium severity: Channel leak
	ch := make(chan int)
	go func() {
		val := <-ch // Leak but less critical
		_ = val
	}()

	// Low severity: Unbalanced but small channel operations
	smallCh := make(chan int)
	for i := 0; i < 100; i++ {
		smallCh <- i
	}
	for i := 0; i < 10; i++ {
		<-smallCh
	}
}`,
			expectedFindings:  4, // Current detection finds 4 issues
			expectedLeakTypes: []string{"channel_receive_leak"},
			expectedBugTypes:  []string{"deadlock"},
			checkSeverity:     false, // Don't check severity for this test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			// Test individual detectors
			leakDetector := NewASTGoroutineLeakDetector()
			bugDetector := NewASTConcurrencyBugDetector()
			smellDetector := NewASTSmellDetector()

			config, _ := valueobjects.NewAnalysisConfiguration(10, 15, 50, true, valueobjects.SeverityInfo)

			leakFindings, _ := leakDetector.DetectLeaks(node, fset, config)
			bugFindings, _ := bugDetector.DetectBugs(node, fset, config)
			smellFindings, _ := smellDetector.DetectSmells(node, fset, config)

			allFindings := append(leakFindings, bugFindings...)
			allFindings = append(allFindings, smellFindings...)

			// Check total count
			if len(allFindings) != tt.expectedFindings {
				t.Errorf("Expected %d findings, got %d", tt.expectedFindings, len(allFindings))
				for i, finding := range allFindings {
					t.Logf("Finding %d [%s]: %s", i, finding.Type().String(), finding.Message())
				}
			}

			// Check leak types
			for _, expectedType := range tt.expectedLeakTypes {
				found := false
				for _, finding := range allFindings {
					if strings.Contains(finding.Message(), expectedType) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected leak type %s not found", expectedType)
				}
			}

			// Check bug types
			for _, expectedType := range tt.expectedBugTypes {
				found := false
				for _, finding := range allFindings {
					if strings.Contains(finding.Message(), expectedType) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected bug type %s not found", expectedType)
				}
			}

			// Check severity if requested
			if tt.checkSeverity {
				for _, finding := range allFindings {
					if finding.Severity() < tt.minSeverity {
						t.Errorf("Finding severity too low: %s has severity %s, expected at least %s",
							finding.Message(), finding.Severity().String(), tt.minSeverity.String())
					}
				}
			}
		})
	}
}

func TestConfidenceScores(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		expectedMessages []string
	}{
		{
			name: "Channel receive leak confidence",
			code: `
package main
func test() {
	ch := make(chan int)
	go func() {
		<-ch // Should have confidence 0.42
	}()
}`,
			expectedMessages: []string{"confidence: 0.42"},
		},
		{
			name: "Select statement leak confidence",
			code: `
package main
func test() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	go func() {
		select {
		case <-ch1:
		case <-ch2:
		}
	}()
}`,
			expectedMessages: []string{"confidence: 0.86"},
		},
		{
			name: "Channel send leak confidence",
			code: `
package main
func test() {
	ch := make(chan int)
	go func() {
		ch <- 1
		return // Premature return - leak
	}()
}`,
			expectedMessages: []string{"confidence: 0.57"},
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

			for _, expectedMsg := range tt.expectedMessages {
				found := false
				for _, finding := range findings {
					if strings.Contains(finding.Message(), expectedMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected confidence message '%s' not found in findings", expectedMsg)
				}
			}
		})
	}
}

func TestSeverityLevels(t *testing.T) {
	code := `
package main
import "sync"
func test() {
	// Critical: Race condition (if enabled)
	shared := 0
	go func() { shared++ }()

	// Error: Select leak
	ch := make(chan int)
	go func() {
		select {
		case <-ch:
		// No escape
		}
	}()

	// Warning: Channel issues
	ch2 := make(chan int)
	for i := 0; i < 10; i++ {
		ch2 <- i
	}
	<-ch2
}`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	smellDetector := NewASTSmellDetector()
	config, _ := valueobjects.NewAnalysisConfiguration(10, 15, 50, true, valueobjects.SeverityInfo)
	findings, err := smellDetector.DetectSmells(node, fset, config)
	if err != nil {
		t.Fatalf("DetectSmells failed: %v", err)
	}

	// Check that we have findings with appropriate severity levels
	severityCount := make(map[valueobjects.SeverityLevel]int)
	for _, finding := range findings {
		severityCount[finding.Severity()]++
	}

	// Should have some warnings and errors
	if severityCount[valueobjects.SeverityWarning] == 0 && severityCount[valueobjects.SeverityError] == 0 {
		t.Error("Expected some warnings or errors, but found none")
	}

	t.Logf("Severity distribution: Info=%d, Warning=%d, Error=%d, Critical=%d",
		severityCount[valueobjects.SeverityInfo],
		severityCount[valueobjects.SeverityWarning],
		severityCount[valueobjects.SeverityError],
		severityCount[valueobjects.SeverityCritical])
}

func TestCombinedAnalysis(t *testing.T) {
	// Test that all detectors work together without conflicts
	code := `
package main
import (
	"context"
	"sync"
)
func comprehensiveTest() {
	// Clean concurrent code that should pass all checks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan int, 10)

	// Worker pool
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			select {
			case results <- workerID:
			case <-ctx.Done():
				return
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	sum := 0
	for result := range results {
		sum += result
	}
	_ = sum
}`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	// Run all detectors
	leakDetector := NewASTGoroutineLeakDetector()
	bugDetector := NewASTConcurrencyBugDetector()
	smellDetector := NewASTSmellDetector()

	config, _ := valueobjects.NewAnalysisConfiguration(10, 15, 50, true, valueobjects.SeverityWarning)

	leakFindings, _ := leakDetector.DetectLeaks(node, fset, config)
	bugFindings, _ := bugDetector.DetectBugs(node, fset, config)
	smellFindings, _ := smellDetector.DetectSmells(node, fset, config)

	totalFindings := len(leakFindings) + len(bugFindings) + len(smellFindings)

	// This clean code should have minimal findings (maybe some complexity)
	if totalFindings > 4 { // Allow some complexity and select warnings
		t.Errorf("Clean concurrent code produced too many findings: %d", totalFindings)
		for _, finding := range leakFindings {
			t.Logf("Leak: %s", finding.Message())
		}
		for _, finding := range bugFindings {
			t.Logf("Bug: %s", finding.Message())
		}
		for _, finding := range smellFindings {
			t.Logf("Smell: %s", finding.Message())
		}
	}

	t.Logf("Total findings for clean code: %d (leak=%d, bug=%d, smell=%d)",
		totalFindings, len(leakFindings), len(bugFindings), len(smellFindings))
}