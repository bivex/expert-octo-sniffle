package services

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"goastanalyzer/domain/entities"
	"goastanalyzer/domain/valueobjects"
)

// LeakPatternType represents different types of goroutine leak patterns
type LeakPatternType int

const (
	LeakPatternChannelReceive LeakPatternType = iota
	LeakPatternSelectStatement
	LeakPatternChannelSend
)

// String returns a string representation of the leak pattern type
func (lpt LeakPatternType) String() string {
	switch lpt {
	case LeakPatternChannelReceive:
		return "channel_receive_leak"
	case LeakPatternSelectStatement:
		return "select_statement_leak"
	case LeakPatternChannelSend:
		return "channel_send_leak"
	default:
		return "unknown"
	}
}

// GoroutineLeakDetector detects goroutine leak patterns in Go code
type GoroutineLeakDetector interface {
	DetectLeaks(node ast.Node, fset *token.FileSet, config valueobjects.AnalysisConfiguration) ([]entities.AnalysisFinding, error)
}

// ASTGoroutineLeakDetector implements GoroutineLeakDetector using AST analysis
type ASTGoroutineLeakDetector struct{}

// NewASTGoroutineLeakDetector creates a new AST-based goroutine leak detector
func NewASTGoroutineLeakDetector() *ASTGoroutineLeakDetector {
	return &ASTGoroutineLeakDetector{}
}

// goroutineContext represents the context of a goroutine analysis
type goroutineContext struct {
	hasChannelReceive bool
	hasChannelClose   bool
	hasSelectStmt     bool
	hasContextCancel  bool
	hasTimeout        bool
	hasPrematureReturn bool
	hasChannelSend    bool
	hasDoubleSend     bool
	hasDeferClose     bool
	functionName      string
	position          token.Position
}

// DetectLeaks analyzes code for goroutine leak patterns
func (gld *ASTGoroutineLeakDetector) DetectLeaks(node ast.Node, fset *token.FileSet, config valueobjects.AnalysisConfiguration) ([]entities.AnalysisFinding, error) {
	var findings []entities.AnalysisFinding

	// First pass: collect all goroutines and their contexts
	goroutines := gld.collectGoroutines(node)

	// Second pass: analyze each goroutine for leak patterns
	for _, goStmt := range goroutines {
		if leakFindings := gld.analyzeGoroutine(goStmt, fset, config); len(leakFindings) > 0 {
			findings = append(findings, leakFindings...)
		}
	}

	return findings, nil
}

// collectGoroutines collects all goroutine statements from the AST
func (gld *ASTGoroutineLeakDetector) collectGoroutines(node ast.Node) []*ast.GoStmt {
	var goroutines []*ast.GoStmt

	ast.Inspect(node, func(n ast.Node) bool {
		if goStmt, ok := n.(*ast.GoStmt); ok {
			goroutines = append(goroutines, goStmt)
		}
		return true
	})

	return goroutines
}

// analyzeGoroutine analyzes a single goroutine for leak patterns
func (gld *ASTGoroutineLeakDetector) analyzeGoroutine(goStmt *ast.GoStmt, fset *token.FileSet, config valueobjects.AnalysisConfiguration) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	context := &goroutineContext{
		functionName: "anonymous_goroutine",
		position:     fset.Position(goStmt.Pos()),
	}

	// Check if it's a function call (named function)
	if callExpr, ok := goStmt.Call.Fun.(*ast.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
			context.functionName = ident.Name
		}
	}

	// Check if it's a function literal (anonymous function)
	if funcLit, ok := goStmt.Call.Fun.(*ast.FuncLit); ok {
		gld.analyzeFunctionBody(funcLit.Body, context)
	}

	// Check if it's a function call with function literal as argument
	if callExpr, ok := goStmt.Call.Fun.(*ast.CallExpr); ok {
		for _, arg := range callExpr.Args {
			if funcLit, ok := arg.(*ast.FuncLit); ok {
				gld.analyzeFunctionBody(funcLit.Body, context)
			}
		}
	}

	// If no function body found (e.g., named function call), still check for leaks
	// based on the goroutine statement itself

	// Check for leak patterns
	findings = append(findings, gld.checkLeakPatterns(context, fset)...)

	return findings
}

// analyzeFunctionBody analyzes the body of a goroutine function
func (gld *ASTGoroutineLeakDetector) analyzeFunctionBody(block *ast.BlockStmt, context *goroutineContext) {
	ast.Inspect(block, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.SendStmt:
			context.hasChannelSend = true
			// Check for potential double send patterns
			gld.checkForDoubleSend(block, stmt, context)
		case *ast.UnaryExpr:
			// Check for channel receive operations
			if stmt.Op == token.ARROW {
				context.hasChannelReceive = true
			}
		case *ast.AssignStmt:
			// Check for channel receives in assignments like: val := <-ch or val, ok := <-ch
			for _, rhs := range stmt.Rhs {
				if unary, ok := rhs.(*ast.UnaryExpr); ok && unary.Op == token.ARROW {
					context.hasChannelReceive = true
				}
			}
		case *ast.CallExpr:
			gld.analyzeCallExpression(stmt, context)
		case *ast.SelectStmt:
			context.hasSelectStmt = true
			gld.analyzeSelectStatement(stmt, context)
		case *ast.DeferStmt:
			// Check for defer close operations
			gld.checkDeferClose(stmt, context)
		case *ast.ReturnStmt:
			// Check if return happens before proper channel operations
			gld.checkPrematureReturn(block, stmt, context)
		}
		return true
	})
}

// analyzeCallExpression analyzes function calls for channel close and context operations
func (gld *ASTGoroutineLeakDetector) analyzeCallExpression(call *ast.CallExpr, context *goroutineContext) {
	if ident, ok := call.Fun.(*ast.Ident); ok {
		switch ident.Name {
		case "close":
			context.hasChannelClose = true
		}
	}

	// Check for context operations
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selector.X.(*ast.Ident); ok {
			if ident.Name == "ctx" || ident.Name == "context" {
				switch selector.Sel.Name {
				case "Done":
					context.hasContextCancel = true
				}
			}
		}
	}
}

// analyzeSelectStatement analyzes select statements for leak patterns
func (gld *ASTGoroutineLeakDetector) analyzeSelectStatement(selectStmt *ast.SelectStmt, context *goroutineContext) {
	hasDefault := false
	hasContextDone := false
	hasTimeout := false

	for _, clause := range selectStmt.Body.List {
		if commClause, ok := clause.(*ast.CommClause); ok {
			if commClause.Comm == nil {
				// Default case
				hasDefault = true
			}

			// Check for context cancellation and other channel receives
			if commClause.Comm != nil {
				// commClause.Comm is *ast.Stmt, but for select cases it's usually *ast.ExprStmt
				if exprStmt, ok := commClause.Comm.(*ast.ExprStmt); ok {
					comm := exprStmt.X
					if unary, ok := comm.(*ast.UnaryExpr); ok && unary.Op == token.ARROW {
						if gld.isContextDoneCall(unary.X) {
							hasContextDone = true
						}
						// Check for potential done/quit channels
						if gld.isDoneChannel(unary.X) {
							hasContextDone = true
						}
						// Check for timeout patterns in channel receives (e.g., <-time.After(...))
						if call, ok := unary.X.(*ast.CallExpr); ok {
							if gld.isTimeoutCall(call) {
								hasTimeout = true
							}
						}
					}
					// Check for direct timeout calls (less common)
					if call, ok := comm.(*ast.CallExpr); ok {
						if gld.isTimeoutCall(call) {
							hasTimeout = true
						}
					}
				}
			}
		}
	}

	// Mark as having proper cancellation if there's default, context cancel, or timeout
	if hasDefault || hasContextDone || hasTimeout {
		context.hasContextCancel = true
	}
	// Set timeout flag if timeout was found
	if hasTimeout {
		context.hasTimeout = true
	}
}

// isContextDoneCall checks if an expression is ctx.Done() or similar
func (gld *ASTGoroutineLeakDetector) isContextDoneCall(expr ast.Expr) bool {
	if selector, ok := expr.(*ast.SelectorExpr); ok {
		if selector.Sel.Name == "Done" {
			if ident, ok := selector.X.(*ast.Ident); ok {
				// Check if variable name suggests context (ctx, context, etc.)
				return strings.Contains(strings.ToLower(ident.Name), "ctx") ||
					   strings.Contains(strings.ToLower(ident.Name), "context")
			}
		}
	}
	return false
}

// isDoneChannel checks if an expression refers to a done/quit/cancel channel
func (gld *ASTGoroutineLeakDetector) isDoneChannel(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		name := strings.ToLower(ident.Name)
		return strings.Contains(name, "done") ||
			   strings.Contains(name, "quit") ||
			   strings.Contains(name, "stop") ||
			   strings.Contains(name, "cancel") ||
			   strings.Contains(name, "exit")
	}
	return false
}

// isTimeoutCall checks if a call expression is a timeout operation
func (gld *ASTGoroutineLeakDetector) isTimeoutCall(call *ast.CallExpr) bool {
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		switch selector.Sel.Name {
		case "After":
			if ident, ok := selector.X.(*ast.Ident); ok {
				return ident.Name == "time"
			}
		case "WithTimeout", "WithDeadline":
			if ident, ok := selector.X.(*ast.Ident); ok {
				return strings.Contains(strings.ToLower(ident.Name), "context")
			}
		}
	}
	return false
}

// checkForDoubleSend checks for potential double send patterns
func (gld *ASTGoroutineLeakDetector) checkForDoubleSend(block *ast.BlockStmt, sendStmt *ast.SendStmt, context *goroutineContext) {
	sendCount := 0
	ast.Inspect(block, func(n ast.Node) bool {
		if s, ok := n.(*ast.SendStmt); ok {
			// Simple check - if there are multiple sends to the same channel
			if gld.sameChannel(sendStmt.Chan, s.Chan) {
				sendCount++
			}
		}
		return true
	})

	if sendCount > 1 {
		context.hasDoubleSend = true
	}
}

// checkDeferClose checks for defer close operations
func (gld *ASTGoroutineLeakDetector) checkDeferClose(deferStmt *ast.DeferStmt, context *goroutineContext) {
	if call, ok := deferStmt.Call.Fun.(*ast.Ident); ok && call.Name == "close" {
		context.hasDeferClose = true
	}
}

// checkPrematureReturn checks if return happens before proper cleanup
func (gld *ASTGoroutineLeakDetector) checkPrematureReturn(block *ast.BlockStmt, returnStmt *ast.ReturnStmt, context *goroutineContext) {
	// If there's a return statement and we have channel operations, mark as premature return
	// This is a simple heuristic - any return in a function with channel operations could be problematic
	if context.hasChannelReceive || context.hasChannelSend {
		context.hasPrematureReturn = true
	}
}

// sameChannel checks if two channel expressions refer to the same channel
func (gld *ASTGoroutineLeakDetector) sameChannel(chan1, chan2 ast.Expr) bool {
	if ident1, ok := chan1.(*ast.Ident); ok {
		if ident2, ok := chan2.(*ast.Ident); ok {
			return ident1.Name == ident2.Name
		}
	}
	return false
}

// checkLeakPatterns analyzes the context and creates findings for detected patterns
func (gld *ASTGoroutineLeakDetector) checkLeakPatterns(context *goroutineContext, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	location, _ := valueobjects.NewSourceLocation(context.position.Filename, context.position.Line, context.position.Column)


	// Only report issues if we actually found operations in the goroutine
	hasOperations := context.hasChannelReceive || context.hasChannelSend || context.hasSelectStmt

	if !hasOperations {
		return findings
	}

	// Channel receive leak pattern - if there's receive operation without close and no context cancellation
	if context.hasChannelReceive && !context.hasChannelClose && !context.hasContextCancel {
		finding, _ := entities.NewAnalysisFinding(
			fmt.Sprintf("channel_receive_leak_%s_%d", context.functionName, context.position.Line),
			entities.FindingTypeSmell,
			location,
			fmt.Sprintf("Potential goroutine leak in %s: channel_receive_leak detected - channel receive without close operation (confidence: 0.42)", context.functionName),
			valueobjects.SeverityWarning,
		)
		findings = append(findings, finding)
	}

	// Select statement leak pattern - only if no context cancellation detected
	if context.hasSelectStmt && !context.hasContextCancel && !context.hasTimeout {
		finding, _ := entities.NewAnalysisFinding(
			fmt.Sprintf("select_statement_leak_%s_%d", context.functionName, context.position.Line),
			entities.FindingTypeSmell,
			location,
			fmt.Sprintf("Potential goroutine leak in %s: select_statement_leak detected - select statement without escape hatch (context cancel or timeout) (confidence: 0.86)", context.functionName),
			valueobjects.SeverityError,
		)
		findings = append(findings, finding)
	}

	// Channel send leak pattern - if there's send operation with premature return and no defer close
	if context.hasChannelSend && context.hasPrematureReturn && !context.hasDeferClose {
		finding, _ := entities.NewAnalysisFinding(
			fmt.Sprintf("channel_send_leak_%s_%d", context.functionName, context.position.Line),
			entities.FindingTypeSmell,
			location,
			fmt.Sprintf("Potential goroutine leak in %s: channel_send_leak detected - channel send with premature return (confidence: 0.57)", context.functionName),
			valueobjects.SeverityWarning,
		)
		findings = append(findings, finding)
	}

	return findings
}