package services

import (
	"go/ast"
	"go/token"

	"goastanalyzer/domain/valueobjects"
)

// ComplexityCalculator calculates complexity metrics for Go code constructs
type ComplexityCalculator interface {
	CalculateComplexity(node ast.Node, fset *token.FileSet) (valueobjects.ComplexityScore, error)
}

// ASTComplexityCalculator implements ComplexityCalculator using AST analysis
type ASTComplexityCalculator struct{}

// NewASTComplexityCalculator creates a new AST-based complexity calculator
func NewASTComplexityCalculator() *ASTComplexityCalculator {
	return &ASTComplexityCalculator{}
}

// CalculateComplexity calculates both cyclomatic and cognitive complexity
func (c *ASTComplexityCalculator) CalculateComplexity(node ast.Node, fset *token.FileSet) (valueobjects.ComplexityScore, error) {
	cyclomatic := c.calculateCyclomaticComplexity(node)
	cognitive := c.calculateCognitiveComplexity(node, 0)

	return valueobjects.NewComplexityScore(cyclomatic, cognitive)
}

// calculateCyclomaticComplexity implements McCabe's cyclomatic complexity
func (c *ASTComplexityCalculator) calculateCyclomaticComplexity(node ast.Node) int {
	complexity := 1 // Base complexity

	ast.Inspect(node, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.SwitchStmt:
			complexity++
		case *ast.TypeSwitchStmt:
			complexity++
		case *ast.SelectStmt:
			complexity++
		case *ast.BinaryExpr:
			// Count && and || operators
			if be := n.(*ast.BinaryExpr); be.Op == token.LAND || be.Op == token.LOR {
				complexity++
			}
		}
		return true
	})

	return complexity
}

// calculateCognitiveComplexity implements Cognitive Complexity metric
func (c *ASTComplexityCalculator) calculateCognitiveComplexity(node ast.Node, nesting int) int {
	complexity := 0

	ast.Inspect(node, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.IfStmt:
			complexity += 1 + nesting
			// Handle else-if chains
			if stmt.Else != nil {
				if elseIf, ok := stmt.Else.(*ast.IfStmt); ok {
					complexity += c.calculateCognitiveComplexity(&ast.BlockStmt{List: []ast.Stmt{&ast.IfStmt{Cond: elseIf.Cond, Body: elseIf.Body}}}, nesting)
				}
			}
		case *ast.ForStmt, *ast.RangeStmt:
			complexity += 1 + nesting
		case *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity += 1 + nesting
		case *ast.SelectStmt:
			complexity += 1 + nesting
		case *ast.BinaryExpr:
			if be := n.(*ast.BinaryExpr); be.Op == token.LAND || be.Op == token.LOR {
				complexity += 1 + nesting
			}
		case *ast.BlockStmt:
			// Increase nesting level for nested blocks
			if n != node { // Don't count the root block
				nesting++
			}
		}
		return true
	})

	return complexity
}
