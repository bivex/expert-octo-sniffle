package usecases

import (
	"fmt"
	"go/ast"
	"go/token"

	"goastanalyzer/domain/aggregates"
	"goastanalyzer/domain/entities"
	"goastanalyzer/domain/services"
	"goastanalyzer/domain/valueobjects"
)

// AnalyzeCodeUseCase defines the contract for analyzing Go code
type AnalyzeCodeUseCase interface {
	Execute(request AnalyzeCodeRequest) (*AnalyzeCodeResponse, error)
}

// AnalyzeCodeRequest represents the input for code analysis
type AnalyzeCodeRequest struct {
	FilePaths        []string
	Configuration    valueobjects.AnalysisConfiguration
	IncludeSmellDetection bool
}

// AnalyzeCodeResponse represents the output of code analysis
type AnalyzeCodeResponse struct {
	AnalysisResult aggregates.AnalysisResult
	Summary        string
	Success        bool
	Error          error
}

// analyzeCodeUseCaseImpl implements AnalyzeCodeUseCase
type analyzeCodeUseCaseImpl struct {
	complexityCalculator services.ComplexityCalculator
	smellDetector        services.SmellDetector
	fileParser           FileParser
	idGenerator          IDGenerator
}

// FileParser defines the interface for parsing Go files
type FileParser interface {
	ParseFile(filePath string) (*ast.File, *token.FileSet, error)
}

// IDGenerator defines the interface for generating unique IDs
type IDGenerator interface {
	GenerateID() string
}

// NewAnalyzeCodeUseCase creates a new analyze code use case
func NewAnalyzeCodeUseCase(
	complexityCalculator services.ComplexityCalculator,
	smellDetector services.SmellDetector,
	fileParser FileParser,
	idGenerator IDGenerator,
) AnalyzeCodeUseCase {
	return &analyzeCodeUseCaseImpl{
		complexityCalculator: complexityCalculator,
		smellDetector:        smellDetector,
		fileParser:           fileParser,
		idGenerator:          idGenerator,
	}
}

// Execute performs the code analysis
func (uc *analyzeCodeUseCaseImpl) Execute(request AnalyzeCodeRequest) (*AnalyzeCodeResponse, error) {
	// Create analysis result
	resultID := uc.idGenerator.GenerateID()
	analysisResult, err := aggregates.NewAnalysisResult(resultID, request.Configuration)
	if err != nil {
		return &AnalyzeCodeResponse{
			Success: false,
			Error:   fmt.Errorf("failed to create analysis result: %w", err),
		}, nil
	}

	totalFunctions := 0
	totalCyclomatic := 0
	totalCognitive := 0

	// Analyze each file
	for _, filePath := range request.FilePaths {
		fileResult, err := uc.analyzeFile(filePath, request.Configuration, request.IncludeSmellDetection)
		if err != nil {
			return &AnalyzeCodeResponse{
				Success: false,
				Error:   fmt.Errorf("failed to analyze file %s: %w", filePath, err),
			}, nil
		}

		// Add findings to result
		for _, finding := range fileResult.Findings {
			analysisResult.AddFinding(finding)
		}

		// Update totals
		totalFunctions += fileResult.FunctionCount
		totalCyclomatic += fileResult.TotalCyclomatic
		totalCognitive += fileResult.TotalCognitive

		analysisResult.AddAnalyzedFile(filePath)
	}

	// Set aggregate metrics
	if totalFunctions > 0 {
		avgCyclomatic := totalCyclomatic / totalFunctions
		avgCognitive := totalCognitive / totalFunctions
		totalComplexity, _ := valueobjects.NewComplexityScore(avgCyclomatic, avgCognitive)
		analysisResult.SetTotalComplexity(totalComplexity)
	}

	analysisResult.SetTotalFunctions(totalFunctions)
	analysisResult.Complete()

	// Create summary
	summary := uc.createSummary(analysisResult)

	return &AnalyzeCodeResponse{
		AnalysisResult: analysisResult,
		Summary:        summary,
		Success:        true,
	}, nil
}

// analyzeFile analyzes a single Go file
func (uc *analyzeCodeUseCaseImpl) analyzeFile(
	filePath string,
	config valueobjects.AnalysisConfiguration,
	includeSmells bool,
) (*FileAnalysisResult, error) {

	// Parse the file
	astFile, fset, err := uc.fileParser.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var findings []entities.AnalysisFinding
	functionCount := 0
	totalCyclomatic := 0
	totalCognitive := 0

	// Analyze each function
	for _, decl := range astFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			functionCount++

			// Calculate complexity
			complexity, err := uc.complexityCalculator.CalculateComplexity(funcDecl, fset)
			if err != nil {
				continue // Skip functions that can't be analyzed
			}

			totalCyclomatic += complexity.Cyclomatic()
			totalCognitive += complexity.Cognitive()

			// Check complexity thresholds
			if complexity.IsHighComplexity() {
				pos := fset.Position(funcDecl.Pos())
				location, _ := valueobjects.NewSourceLocation(filePath, pos.Line, pos.Column)

				var severity valueobjects.SeverityLevel
				if complexity.Cyclomatic() > 20 || complexity.Cognitive() > 30 {
					severity = valueobjects.SeverityError
				} else {
					severity = valueobjects.SeverityWarning
				}

				finding, _ := entities.NewAnalysisFinding(
					fmt.Sprintf("complexity_%s_%d", funcDecl.Name.Name, pos.Line),
					entities.FindingTypeComplexity,
					location,
					fmt.Sprintf("Function %s: %s", funcDecl.Name.Name, complexity.String()),
					severity,
				)
				findings = append(findings, finding)
			}
		}
	}

	// Detect smells if requested
	if includeSmells {
		smellFindings, err := uc.smellDetector.DetectSmells(astFile, fset, config)
		if err != nil {
			return nil, fmt.Errorf("failed to detect smells: %w", err)
		}
		findings = append(findings, smellFindings...)
	}

	return &FileAnalysisResult{
		FilePath:       filePath,
		Findings:       findings,
		FunctionCount:  functionCount,
		TotalCyclomatic: totalCyclomatic,
		TotalCognitive:  totalCognitive,
	}, nil
}

// createSummary creates a human-readable summary of the analysis
func (uc *analyzeCodeUseCaseImpl) createSummary(result aggregates.AnalysisResult) string {
	summary := result.Summary()

	return fmt.Sprintf(
		"Analysis complete: %d files, %d functions analyzed. Found %d issues (%d high severity) in %v. Complexity: %s",
		summary.TotalFiles,
		summary.TotalFunctions,
		summary.TotalFindings,
		summary.HighSeverityCount,
		summary.Duration,
		result.TotalComplexity().String(),
	)
}

// FileAnalysisResult represents the analysis result for a single file
type FileAnalysisResult struct {
	FilePath        string
	Findings        []entities.AnalysisFinding
	FunctionCount   int
	TotalCyclomatic int
	TotalCognitive  int
}
