package services

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"goastanalyzer/domain/entities"
	"goastanalyzer/domain/valueobjects"
)

// ConcurrencyBugType represents different types of concurrency bugs
type ConcurrencyBugType int

const (
	ConcurrencyBugBlocking ConcurrencyBugType = iota
	ConcurrencyBugNonBlocking
	ConcurrencyBugRaceCondition
)

// String returns a string representation of the concurrency bug type
func (cbt ConcurrencyBugType) String() string {
	switch cbt {
	case ConcurrencyBugBlocking:
		return "blocking_bug"
	case ConcurrencyBugNonBlocking:
		return "non_blocking_bug"
	case ConcurrencyBugRaceCondition:
		return "race_condition"
	default:
		return "unknown"
	}
}

// BlockingCause represents the cause of a blocking bug
type BlockingCause int

const (
	BlockingCauseChannelMisuse BlockingCause = iota
	BlockingCauseSharedMemorySync
	BlockingCauseDeadlock
)

// String returns a string representation of the blocking cause
func (bc BlockingCause) String() string {
	switch bc {
	case BlockingCauseChannelMisuse:
		return "channel_misuse"
	case BlockingCauseSharedMemorySync:
		return "shared_memory_sync"
	case BlockingCauseDeadlock:
		return "deadlock"
	default:
		return "unknown"
	}
}

// ConcurrencyBugDetector detects concurrency bug patterns in Go code
type ConcurrencyBugDetector interface {
	DetectBugs(node ast.Node, fset *token.FileSet, config valueobjects.AnalysisConfiguration) ([]entities.AnalysisFinding, error)
}

// ASTConcurrencyBugDetector implements ConcurrencyBugDetector using AST analysis
type ASTConcurrencyBugDetector struct{}

// NewASTConcurrencyBugDetector creates a new AST-based concurrency bug detector
func NewASTConcurrencyBugDetector() *ASTConcurrencyBugDetector {
	return &ASTConcurrencyBugDetector{}
}

// DetectBugs analyzes code for concurrency bug patterns
func (cbd *ASTConcurrencyBugDetector) DetectBugs(node ast.Node, fset *token.FileSet, config valueobjects.AnalysisConfiguration) ([]entities.AnalysisFinding, error) {
	var findings []entities.AnalysisFinding

	ast.Inspect(node, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.FuncDecl:
			if bugFindings := cbd.analyzeFunctionConcurrency(stmt, fset, config); len(bugFindings) > 0 {
				findings = append(findings, bugFindings...)
			}
		}
		return true
	})

	return findings, nil
}

// analyzeFunctionConcurrency analyzes a function for concurrency bug patterns
func (cbd *ASTConcurrencyBugDetector) analyzeFunctionConcurrency(funcDecl *ast.FuncDecl, fset *token.FileSet, config valueobjects.AnalysisConfiguration) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	if funcDecl.Body == nil {
		return findings
	}

	// Analyze for blocking bugs
	findings = append(findings, cbd.detectBlockingBugs(funcDecl, fset)...)

	// Analyze for race conditions
	findings = append(findings, cbd.detectRaceConditions(funcDecl, fset)...)

	return findings
}

// detectBlockingBugs detects blocking concurrency bugs
func (cbd *ASTConcurrencyBugDetector) detectBlockingBugs(funcDecl *ast.FuncDecl, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	blockingPatterns := cbd.analyzeBlockingPatterns(funcDecl.Body)

	for _, pattern := range blockingPatterns {
		location, _ := valueobjects.NewSourceLocation(
			fset.Position(funcDecl.Pos()).Filename,
			fset.Position(funcDecl.Pos()).Line,
			fset.Position(funcDecl.Pos()).Column,
		)

		var message string
		var severity valueobjects.SeverityLevel

		switch pattern.cause {
		case BlockingCauseChannelMisuse:
			message = fmt.Sprintf("Potential blocking bug in %s: channel misuse detected - %s", funcDecl.Name.Name, pattern.description)
			severity = valueobjects.SeverityWarning
		case BlockingCauseSharedMemorySync:
			message = fmt.Sprintf("Potential blocking bug in %s: shared memory synchronization issue - %s", funcDecl.Name.Name, pattern.description)
			severity = valueobjects.SeverityError
		case BlockingCauseDeadlock:
			message = fmt.Sprintf("Potential blocking bug in %s: deadlock pattern detected - %s", funcDecl.Name.Name, pattern.description)
			severity = valueobjects.SeverityCritical
		}

		finding, _ := entities.NewAnalysisFinding(
			fmt.Sprintf("%s_%s_%d", pattern.cause.String(), funcDecl.Name.Name, fset.Position(funcDecl.Pos()).Line),
			entities.FindingTypeBug,
			location,
			message,
			severity,
		)
		findings = append(findings, finding)
	}

	return findings
}

// blockingPattern represents a detected blocking pattern
type blockingPattern struct {
	cause       BlockingCause
	description string
}

// analyzeBlockingPatterns analyzes code for blocking concurrency patterns
func (cbd *ASTConcurrencyBugDetector) analyzeBlockingPatterns(block *ast.BlockStmt) []blockingPattern {
	var patterns []blockingPattern

	channelOps := make(map[string][]string) // channel name -> operations
	mutexOps := make(map[string][]string)   // mutex name -> operations

	ast.Inspect(block, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.SendStmt:
			if ident, ok := stmt.Chan.(*ast.Ident); ok {
				channelOps[ident.Name] = append(channelOps[ident.Name], "send")
			}
		case *ast.UnaryExpr:
			if stmt.Op == token.ARROW {
				if ident, ok := stmt.X.(*ast.Ident); ok {
					channelOps[ident.Name] = append(channelOps[ident.Name], "receive")
				}
			}
		case *ast.CallExpr:
			cbd.analyzeCallForBlocking(stmt, mutexOps)
		case *ast.GoStmt:
			// Also analyze goroutine bodies for blocking patterns
			if funcLit, ok := stmt.Call.Fun.(*ast.FuncLit); ok {
				goroutinePatterns := cbd.analyzeBlockingPatterns(funcLit.Body)
				patterns = append(patterns, goroutinePatterns...)
			}
		case *ast.SelectStmt:
			// Check for select statements that might block indefinitely
			if cbd.isPotentiallyBlockingSelect(stmt) {
				patterns = append(patterns, blockingPattern{
					cause:       BlockingCauseChannelMisuse,
					description: "select statement without default or timeout case may block indefinitely",
				})
			}
		}
		return true
	})

	// Analyze channel operations for misuse patterns
	for channelName, ops := range channelOps {
		if cbd.hasPotentialChannelBlock(ops) {
			patterns = append(patterns, blockingPattern{
				cause:       BlockingCauseChannelMisuse,
				description: fmt.Sprintf("channel '%s' operations may cause blocking: %v", channelName, ops),
			})
		}
	}

	// Analyze mutex operations for deadlock potential
	for mutexName, ops := range mutexOps {
		if cbd.hasPotentialDeadlock(ops) {
			patterns = append(patterns, blockingPattern{
				cause:       BlockingCauseDeadlock,
				description: fmt.Sprintf("mutex '%s' usage pattern may cause deadlock: %v", mutexName, ops),
			})
		}
	}

	return patterns
}

// analyzeCallForBlocking analyzes function calls for blocking patterns
func (cbd *ASTConcurrencyBugDetector) analyzeCallForBlocking(call *ast.CallExpr, mutexOps map[string][]string) {
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selector.X.(*ast.Ident); ok {
			switch selector.Sel.Name {
			case "Lock", "RLock":
				mutexOps[ident.Name] = append(mutexOps[ident.Name], "lock")
			case "Unlock", "RUnlock":
				mutexOps[ident.Name] = append(mutexOps[ident.Name], "unlock")
			}
		}
	}

	// Also check for nested mutex operations
	ast.Inspect(call, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				if _, exists := mutexOps[id.Name]; exists {
					switch sel.Sel.Name {
					case "Lock", "RLock":
						mutexOps[id.Name] = append(mutexOps[id.Name], "lock")
					case "Unlock", "RUnlock":
						mutexOps[id.Name] = append(mutexOps[id.Name], "unlock")
					}
				}
			}
		}
		return true
	})
}

// isPotentiallyBlockingSelect checks if a select statement might block indefinitely
func (cbd *ASTConcurrencyBugDetector) isPotentiallyBlockingSelect(selectStmt *ast.SelectStmt) bool {
	hasDefault := false
	hasTimeout := false
	hasContextDone := false

	caseCount := 0
	for _, clause := range selectStmt.Body.List {
		if commClause, ok := clause.(*ast.CommClause); ok {
			if commClause.Comm == nil {
				hasDefault = true
			} else {
				caseCount++
			}

			// Check for timeout patterns and context done
			if commClause.Comm != nil {
				if exprStmt, ok := commClause.Comm.(*ast.ExprStmt); ok {
					if unary, ok := exprStmt.X.(*ast.UnaryExpr); ok && unary.Op == token.ARROW {
						if cbd.isContextDoneCall(unary.X) {
							hasContextDone = true
						}
						if call, ok := unary.X.(*ast.CallExpr); ok {
							if cbd.isTimeoutCall(call) {
								hasTimeout = true
							}
						}
					}
				}
			}
		}
	}

	// Flag as potentially blocking if multiple cases and no escape routes
	return caseCount >= 2 && !hasDefault && !hasTimeout && !hasContextDone
}

// hasDefaultCase checks if a select statement has a default case
func (cbd *ASTConcurrencyBugDetector) hasDefaultCase(selectStmt *ast.SelectStmt) bool {
	for _, clause := range selectStmt.Body.List {
		if caseClause, ok := clause.(*ast.CaseClause); ok {
			if caseClause.List == nil {
				return true
			}
		}
	}
	return false
}

// hasTimeoutCase checks if a select statement has a timeout case
func (cbd *ASTConcurrencyBugDetector) hasTimeoutCase(selectStmt *ast.SelectStmt) bool {
	for _, clause := range selectStmt.Body.List {
		if caseClause, ok := clause.(*ast.CaseClause); ok {
			for _, comm := range caseClause.List {
				if call, ok := comm.(*ast.CallExpr); ok {
					if cbd.isTimeoutCall(call) {
						return true
					}
				}
			}
		}
	}
	return false
}

// isTimeoutCall checks if a call expression is a timeout operation
func (cbd *ASTConcurrencyBugDetector) isTimeoutCall(call *ast.CallExpr) bool {
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		return selector.Sel.Name == "After" || strings.Contains(selector.Sel.Name, "Timeout")
	}
	return false
}

// isContextDoneCall checks if an expression is ctx.Done() or similar
func (cbd *ASTConcurrencyBugDetector) isContextDoneCall(expr ast.Expr) bool {
	// Check for direct selector (ctx.Done)
	if selector, ok := expr.(*ast.SelectorExpr); ok {
		if selector.Sel.Name == "Done" {
			if ident, ok := selector.X.(*ast.Ident); ok {
				// Check if variable name suggests context (ctx, context, etc.)
				return strings.Contains(strings.ToLower(ident.Name), "ctx") ||
					   strings.Contains(strings.ToLower(ident.Name), "context")
			}
		}
	}
	// Check for method call (ctx.Done())
	if call, ok := expr.(*ast.CallExpr); ok {
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			if selector.Sel.Name == "Done" {
				if ident, ok := selector.X.(*ast.Ident); ok {
					return strings.Contains(strings.ToLower(ident.Name), "ctx") ||
						   strings.Contains(strings.ToLower(ident.Name), "context")
				}
			}
		}
	}
	return false
}

// hasPotentialChannelBlock checks if channel operations may cause blocking
func (cbd *ASTConcurrencyBugDetector) hasPotentialChannelBlock(ops []string) bool {
	sendCount := 0
	receiveCount := 0

	for _, op := range ops {
		if op == "send" {
			sendCount++
		} else if op == "receive" {
			receiveCount++
		}
	}

	// Flag significant imbalances
	totalOps := sendCount + receiveCount
	if totalOps < 2 {
		return false // Don't flag minimal usage
	}

	// Flag significant imbalances
	return (sendCount > receiveCount * 2 && sendCount > 1) || (receiveCount > sendCount * 2 && receiveCount > 1)
}

// hasPotentialDeadlock checks if mutex operations may cause deadlock
func (cbd *ASTConcurrencyBugDetector) hasPotentialDeadlock(ops []string) bool {
	lockCount := 0
	unlockCount := 0

	for _, op := range ops {
		if op == "lock" {
			lockCount++
		} else if op == "unlock" {
			unlockCount++
		}
	}

	// Potential deadlock if locks significantly outnumber unlocks
	return lockCount > unlockCount + 1
}

// detectRaceConditions detects race condition patterns
func (cbd *ASTConcurrencyBugDetector) detectRaceConditions(funcDecl *ast.FuncDecl, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	// Check if function contains goroutines
	hasGoroutines := cbd.containsGoroutines(funcDecl.Body)
	if !hasGoroutines {
		return findings
	}

	racePatterns := cbd.analyzeRaceConditionPatterns(funcDecl.Body)

	for _, pattern := range racePatterns {
		location, _ := valueobjects.NewSourceLocation(
			fset.Position(funcDecl.Pos()).Filename,
			fset.Position(funcDecl.Pos()).Line,
			fset.Position(funcDecl.Pos()).Column,
		)

		finding, _ := entities.NewAnalysisFinding(
			fmt.Sprintf("race_condition_%s_%d", funcDecl.Name.Name, fset.Position(funcDecl.Pos()).Line),
			entities.FindingTypeBug,
			location,
			fmt.Sprintf("Potential race condition in %s: %s", funcDecl.Name.Name, pattern.description),
			valueobjects.SeverityCritical,
		)
		findings = append(findings, finding)
	}

	return findings
}

// containsGoroutines checks if a function body contains goroutine statements
func (cbd *ASTConcurrencyBugDetector) containsGoroutines(block *ast.BlockStmt) bool {
	hasGo := false
	ast.Inspect(block, func(n ast.Node) bool {
		if _, ok := n.(*ast.GoStmt); ok {
			hasGo = true
			return false // Stop traversal
		}
		return true
	})
	return hasGo
}

// raceConditionPattern represents a detected race condition pattern
type raceConditionPattern struct {
	description string
}

// analyzeRaceConditionPatterns analyzes code for race condition patterns
func (cbd *ASTConcurrencyBugDetector) analyzeRaceConditionPatterns(block *ast.BlockStmt) []raceConditionPattern {
	var patterns []raceConditionPattern

	// For now, disable race condition detection as it's too aggressive
	// and produces too many false positives on correct concurrent code
	// TODO: Implement more sophisticated race condition detection

	return patterns
}

// findGoroutineBlocks finds all goroutine function bodies
func (cbd *ASTConcurrencyBugDetector) findGoroutineBlocks(block *ast.BlockStmt) []*ast.BlockStmt {
	var blocks []*ast.BlockStmt

	ast.Inspect(block, func(n ast.Node) bool {
		if goStmt, ok := n.(*ast.GoStmt); ok {
			// Check for function literal in goroutine
			if funcLit, ok := goStmt.Call.Fun.(*ast.FuncLit); ok {
				blocks = append(blocks, funcLit.Body)
			}
			// Check for function literal as argument
			if callExpr, ok := goStmt.Call.Fun.(*ast.CallExpr); ok {
				for _, arg := range callExpr.Args {
					if funcLit, ok := arg.(*ast.FuncLit); ok {
						blocks = append(blocks, funcLit.Body)
					}
				}
			}
		}
		return true
	})

	return blocks
}

// findSharedVariables finds variables that are clearly shared between goroutines
func (cbd *ASTConcurrencyBugDetector) findSharedVariables(block *ast.BlockStmt) []string {
	sharedVars := make(map[string]bool)
	goroutineVars := make(map[string]bool)

	// Find variables declared in main scope
	mainScopeVars := make(map[string]bool)
	ast.Inspect(block, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok {
			if assign.Tok == token.DEFINE {
				for _, lhs := range assign.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						mainScopeVars[ident.Name] = true
					}
				}
			}
		}
		return true
	})

	// Find variables used in goroutines
	goroutineBlocks := cbd.findGoroutineBlocks(block)
	for _, goBlock := range goroutineBlocks {
		ast.Inspect(goBlock, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				if mainScopeVars[ident.Name] {
					sharedVars[ident.Name] = true
				}
				goroutineVars[ident.Name] = true
			}
			return true
		})
	}

	// Convert map to slice
	var result []string
	for varName := range sharedVars {
		result = append(result, varName)
	}

	return result
}

// findSynchronizationPrimitives finds mutexes and other sync primitives
func (cbd *ASTConcurrencyBugDetector) findSynchronizationPrimitives(block *ast.BlockStmt, syncPrimitives map[string]string) {
	ast.Inspect(block, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.GenDecl:
			// Find mutex variable declarations
			for _, spec := range stmt.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						if valueSpec.Type != nil {
							if selector, ok := valueSpec.Type.(*ast.SelectorExpr); ok {
								if ident, ok := selector.X.(*ast.Ident); ok {
									if ident.Name == "sync" {
										switch selector.Sel.Name {
										case "Mutex", "RWMutex":
											syncPrimitives[name.Name] = "mutex"
										case "WaitGroup":
											syncPrimitives[name.Name] = "waitgroup"
										case "Once":
											syncPrimitives[name.Name] = "once"
										}
									}
								}
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			// Also check for mutex assignments like: var mu sync.Mutex
			for _, rhs := range stmt.Rhs {
				if composite, ok := rhs.(*ast.CompositeLit); ok {
					if selector, ok := composite.Type.(*ast.SelectorExpr); ok {
						if ident, ok := selector.X.(*ast.Ident); ok {
							if ident.Name == "sync" {
								for _, lhs := range stmt.Lhs {
									if lhsIdent, ok := lhs.(*ast.Ident); ok {
										switch selector.Sel.Name {
										case "Mutex", "RWMutex":
											syncPrimitives[lhsIdent.Name] = "mutex"
										case "WaitGroup":
											syncPrimitives[lhsIdent.Name] = "waitgroup"
										case "Once":
											syncPrimitives[lhsIdent.Name] = "once"
										}
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})
}

// analyzeBlockForRaces analyzes a block of code for variable accesses that could cause races
func (cbd *ASTConcurrencyBugDetector) analyzeBlockForRaces(block *ast.BlockStmt, variableAccesses map[string][]accessInfo, goroutineID string, syncPrimitives map[string]string) {
	// Track if we're inside a locked section
	lockedVariables := make(map[string]bool)

	ast.Inspect(block, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			cbd.analyzeAssignmentForRace(stmt, variableAccesses, goroutineID, lockedVariables)
		case *ast.IncDecStmt:
			if ident, ok := stmt.X.(*ast.Ident); ok {
				accessType := "write"
				if lockedVariables[ident.Name] {
					accessType = "protected_write"
				}
				cbd.recordAccess(variableAccesses, ident.Name, accessType, goroutineID)
			}
		case *ast.CallExpr:
			// Track mutex lock/unlock calls
			if selector, ok := stmt.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := selector.X.(*ast.Ident); ok {
					if _, isMutex := syncPrimitives[ident.Name]; isMutex {
						switch selector.Sel.Name {
						case "Lock", "RLock":
							// For simplicity, assume all variables in the same function scope are protected
							// In a real implementation, this would need more sophisticated scope analysis
							for varName := range variableAccesses {
								lockedVariables[varName] = true
							}
						case "Unlock", "RUnlock":
							for varName := range variableAccesses {
								lockedVariables[varName] = false
							}
						}
					}
				}
			}
		}
		return true
	})
}

// accessInfo represents information about a variable access
type accessInfo struct {
	variable   string // variable name
	accessType string // "read" or "write" or "protected_read/write"
	goroutine  string // simplified goroutine identifier
}

// analyzeAssignmentForRace analyzes assignment statements for race conditions
func (cbd *ASTConcurrencyBugDetector) analyzeAssignmentForRace(assign *ast.AssignStmt, variableAccesses map[string][]accessInfo, goroutineID string, lockedVariables map[string]bool) {
	for _, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			accessType := "write"
			if assign.Tok == token.DEFINE {
				accessType = "write" // variable declaration with assignment
			}
			if lockedVariables[ident.Name] {
				accessType = "protected_" + accessType
			}
			cbd.recordAccess(variableAccesses, ident.Name, accessType, goroutineID)
		}
	}

	// Also check RHS for reads
	for _, rhs := range assign.Rhs {
		ast.Inspect(rhs, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				accessType := "read"
				if lockedVariables[ident.Name] {
					accessType = "protected_" + accessType
				}
				cbd.recordAccess(variableAccesses, ident.Name, accessType, goroutineID)
			}
			return true
		})
	}
}

// recordAccess records a variable access
func (cbd *ASTConcurrencyBugDetector) recordAccess(accesses map[string][]accessInfo, varName, accessType, goroutine string) {
	accesses[varName] = append(accesses[varName], accessInfo{
		variable:   varName,
		accessType: accessType,
		goroutine:  goroutine,
	})
}

// hasUnprotectedConcurrentAccess checks if a variable has unprotected concurrent access
func (cbd *ASTConcurrencyBugDetector) hasUnprotectedConcurrentAccess(accesses []accessInfo, syncPrimitives map[string]string) bool {
	hasUnprotectedWrite := false
	goroutines := make(map[string]bool)

	// Skip variables that are synchronization primitives themselves
	for varName := range syncPrimitives {
		for _, access := range accesses {
			if access.variable == varName {
				return false // Don't report sync primitives as race conditions
			}
		}
	}

	for _, access := range accesses {
		goroutines[access.goroutine] = true
		if strings.HasPrefix(access.accessType, "write") && !strings.HasPrefix(access.accessType, "protected_") {
			hasUnprotectedWrite = true
		}
	}

	// If accessed from multiple goroutines and has unprotected writes, potential race condition
	return len(goroutines) > 1 && hasUnprotectedWrite
}