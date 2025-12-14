package services

import (
	"fmt"
	"go/ast"
	"go/token"

	"goastanalyzer/domain/entities"
	"goastanalyzer/domain/valueobjects"
)

// SmellType represents different types of architectural smells
type SmellType int

const (
	SmellTypeGodStruct SmellType = iota
	SmellTypeGodPackage
	SmellTypeInterfacePollution
	SmellTypeGoroutineLeak
	SmellTypeLongFunction
	SmellTypeDeepNesting
	SmellTypeConcurrencyBug
	SmellTypeChannelReceiveLeak
	SmellTypeSelectStatementLeak
	SmellTypeChannelSendLeak
	SmellTypeBlockingBug
	SmellTypeRaceCondition
)

// String returns a string representation of the smell type
func (st SmellType) String() string {
	switch st {
	case SmellTypeGodStruct:
		return "god_struct"
	case SmellTypeGodPackage:
		return "god_package"
	case SmellTypeInterfacePollution:
		return "interface_pollution"
	case SmellTypeGoroutineLeak:
		return "goroutine_leak"
	case SmellTypeLongFunction:
		return "long_function"
	case SmellTypeDeepNesting:
		return "deep_nesting"
	case SmellTypeConcurrencyBug:
		return "concurrency_bug"
	case SmellTypeChannelReceiveLeak:
		return "channel_receive_leak"
	case SmellTypeSelectStatementLeak:
		return "select_statement_leak"
	case SmellTypeChannelSendLeak:
		return "channel_send_leak"
	case SmellTypeBlockingBug:
		return "blocking_bug"
	case SmellTypeRaceCondition:
		return "race_condition"
	default:
		return "unknown"
	}
}

// SmellDetector detects architectural smells in Go code
type SmellDetector interface {
	DetectSmells(node ast.Node, fset *token.FileSet, config valueobjects.AnalysisConfiguration) ([]entities.AnalysisFinding, error)
}

// ASTSmellDetector implements SmellDetector using AST analysis
type ASTSmellDetector struct {
	goroutineLeakDetector   GoroutineLeakDetector
	concurrencyBugDetector  ConcurrencyBugDetector
}

// NewASTSmellDetector creates a new AST-based smell detector
func NewASTSmellDetector() *ASTSmellDetector {
	return &ASTSmellDetector{
		goroutineLeakDetector:  NewASTGoroutineLeakDetector(),
		concurrencyBugDetector: NewASTConcurrencyBugDetector(),
	}
}

// DetectSmells analyzes code for architectural smells
func (sd *ASTSmellDetector) DetectSmells(node ast.Node, fset *token.FileSet, config valueobjects.AnalysisConfiguration) ([]entities.AnalysisFinding, error) {
	var findings []entities.AnalysisFinding

	// Detect traditional architectural smells
	ast.Inspect(node, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if funcFindings := sd.detectFunctionSmells(node, fset, config); len(funcFindings) > 0 {
				findings = append(findings, funcFindings...)
			}
		case *ast.GenDecl:
			if declFindings := sd.detectDeclarationSmells(node, fset); len(declFindings) > 0 {
				findings = append(findings, declFindings...)
			}
		}
		return true
	})

	// Detect goroutine leaks
	if leakFindings, err := sd.goroutineLeakDetector.DetectLeaks(node, fset, config); err == nil {
		findings = append(findings, leakFindings...)
	}

	// Detect concurrency bugs
	if bugFindings, err := sd.concurrencyBugDetector.DetectBugs(node, fset, config); err == nil {
		findings = append(findings, bugFindings...)
	}

	return findings, nil
}

// detectFunctionSmells detects smells in function declarations
func (sd *ASTSmellDetector) detectFunctionSmells(funcDecl *ast.FuncDecl, fset *token.FileSet, config valueobjects.AnalysisConfiguration) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding
	pos := fset.Position(funcDecl.Pos())

	// Check for long functions
	if funcDecl.Body != nil {
		lineCount := sd.countLinesInBlock(funcDecl.Body, fset)
		if lineCount > config.MaxFunctionLength() {
			location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("long_function_%s_%d", funcDecl.Name.Name, pos.Line),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Function %s is too long: %d lines (max: %d)",
					funcDecl.Name.Name, lineCount, config.MaxFunctionLength()),
				valueobjects.SeverityWarning,
			)
			findings = append(findings, finding)
		}

		// Check for deep nesting
		maxNesting := sd.calculateMaxNesting(funcDecl.Body)
		if maxNesting > 4 { // Go-adjusted threshold
			location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("deep_nesting_%s_%d", funcDecl.Name.Name, pos.Line),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Function %s has deep nesting: level %d (max recommended: 4)",
					funcDecl.Name.Name, maxNesting),
				valueobjects.SeverityWarning,
			)
			findings = append(findings, finding)
		}
	}

	return findings
}

// detectDeclarationSmells detects smells in type/struct declarations
func (sd *ASTSmellDetector) detectDeclarationSmells(genDecl *ast.GenDecl, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	for _, spec := range genDecl.Specs {
		if typeSpec, ok := spec.(*ast.TypeSpec); ok {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				if fieldCount := sd.countStructFields(structType); fieldCount > 10 {
					pos := fset.Position(typeSpec.Pos())
					location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
					finding, _ := entities.NewAnalysisFinding(
						fmt.Sprintf("god_struct_%s_%d", typeSpec.Name.Name, pos.Line),
						entities.FindingTypeSmell,
						location,
						fmt.Sprintf("Struct %s has too many fields: %d (max recommended: 10)",
							typeSpec.Name.Name, fieldCount),
						valueobjects.SeverityWarning,
					)
					findings = append(findings, finding)
				}
			}

			if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				if methodCount := sd.countInterfaceMethods(interfaceType); methodCount > 7 {
					pos := fset.Position(typeSpec.Pos())
					location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
					finding, _ := entities.NewAnalysisFinding(
						fmt.Sprintf("interface_pollution_%s_%d", typeSpec.Name.Name, pos.Line),
						entities.FindingTypeSmell,
						location,
						fmt.Sprintf("Interface %s has too many methods: %d (max recommended: 7)",
							typeSpec.Name.Name, methodCount),
						valueobjects.SeverityWarning,
					)
					findings = append(findings, finding)
				}
			}
		}
	}

	return findings
}

// countLinesInBlock counts the number of lines in a block statement
func (sd *ASTSmellDetector) countLinesInBlock(block *ast.BlockStmt, fset *token.FileSet) int {
	if block == nil || len(block.List) == 0 {
		return 0
	}

	startPos := fset.Position(block.Lbrace)
	endPos := fset.Position(block.Rbrace)

	return endPos.Line - startPos.Line + 1
}

// calculateMaxNesting calculates the maximum nesting level in a block
func (sd *ASTSmellDetector) calculateMaxNesting(block *ast.BlockStmt) int {
	maxNesting := 0

	// Simple iterative approach to avoid stack overflow
	var inspect func(node ast.Node, currentNesting int)
	inspect = func(node ast.Node, currentNesting int) {
		if currentNesting > maxNesting {
			maxNesting = currentNesting
		}

		// Only inspect immediate children to avoid infinite recursion
		switch stmt := node.(type) {
		case *ast.IfStmt:
			if stmt.Body != nil {
				inspect(stmt.Body, currentNesting+1)
			}
			if stmt.Else != nil {
				if elseIf, ok := stmt.Else.(*ast.IfStmt); ok {
					inspect(elseIf, currentNesting)
				} else if block, ok := stmt.Else.(*ast.BlockStmt); ok {
					inspect(block, currentNesting+1)
				}
			}
		case *ast.ForStmt:
			if stmt.Body != nil {
				inspect(stmt.Body, currentNesting+1)
			}
		case *ast.RangeStmt:
			if stmt.Body != nil {
				inspect(stmt.Body, currentNesting+1)
			}
		case *ast.SwitchStmt:
			if stmt.Body != nil {
				inspect(stmt.Body, currentNesting+1)
			}
		case *ast.TypeSwitchStmt:
			if stmt.Body != nil {
				inspect(stmt.Body, currentNesting+1)
			}
		case *ast.SelectStmt:
			if stmt.Body != nil {
				inspect(stmt.Body, currentNesting+1)
			}
		case *ast.BlockStmt:
			for _, stmt := range stmt.List {
				inspect(stmt, currentNesting)
			}
		}
	}

	inspect(block, 0)
	return maxNesting
}

// countStructFields counts the number of fields in a struct
func (sd *ASTSmellDetector) countStructFields(structType *ast.StructType) int {
	if structType.Fields == nil {
		return 0
	}
	return len(structType.Fields.List)
}

// countInterfaceMethods counts the number of methods in an interface
func (sd *ASTSmellDetector) countInterfaceMethods(interfaceType *ast.InterfaceType) int {
	if interfaceType.Methods == nil {
		return 0
	}
	return len(interfaceType.Methods.List)
}
