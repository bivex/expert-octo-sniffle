package aggregates

import (
	"fmt"
	"time"

	"goastanalyzer/domain/entities"
	"goastanalyzer/domain/valueobjects"
)

// AnalysisResult represents the complete result of analyzing a codebase
type AnalysisResult struct {
	id               string
	analyzedFiles    []string
	findings         []entities.AnalysisFinding
	configuration    valueobjects.AnalysisConfiguration
	startTime        time.Time
	endTime          time.Time
	totalFiles       int
	totalFunctions   int
	totalComplexity  valueobjects.ComplexityScore
}

// NewAnalysisResult creates a new analysis result
func NewAnalysisResult(id string, config valueobjects.AnalysisConfiguration) (AnalysisResult, error) {
	if id == "" {
		return AnalysisResult{}, fmt.Errorf("analysis result ID cannot be empty")
	}

	return AnalysisResult{
		id:            id,
		analyzedFiles: make([]string, 0),
		findings:      make([]entities.AnalysisFinding, 0),
		configuration: config,
		startTime:     time.Now(),
		totalFiles:    0,
		totalFunctions: 0,
		totalComplexity: valueobjects.ComplexityScore{}, // Will be calculated
	}, nil
}

// ID returns the unique identifier for this analysis result
func (ar AnalysisResult) ID() string {
	return ar.id
}

// AnalyzedFiles returns the list of files that were analyzed
func (ar AnalysisResult) AnalyzedFiles() []string {
	// Return a copy to prevent external modification
	files := make([]string, len(ar.analyzedFiles))
	copy(files, ar.analyzedFiles)
	return files
}

// Findings returns all findings from this analysis
func (ar AnalysisResult) Findings() []entities.AnalysisFinding {
	// Return a copy to prevent external modification
	findings := make([]entities.AnalysisFinding, len(ar.findings))
	copy(findings, ar.findings)
	return findings
}

// Configuration returns the analysis configuration used
func (ar AnalysisResult) Configuration() valueobjects.AnalysisConfiguration {
	return ar.configuration
}

// StartTime returns when the analysis started
func (ar AnalysisResult) StartTime() time.Time {
	return ar.startTime
}

// EndTime returns when the analysis ended
func (ar AnalysisResult) EndTime() time.Time {
	return ar.endTime
}

// Duration returns the total duration of the analysis
func (ar AnalysisResult) Duration() time.Duration {
	if ar.endTime.IsZero() {
		return time.Since(ar.startTime)
	}
	return ar.endTime.Sub(ar.startTime)
}

// TotalFiles returns the total number of files analyzed
func (ar AnalysisResult) TotalFiles() int {
	return ar.totalFiles
}

// TotalFunctions returns the total number of functions analyzed
func (ar AnalysisResult) TotalFunctions() int {
	return ar.totalFunctions
}

// TotalComplexity returns the aggregate complexity metrics
func (ar AnalysisResult) TotalComplexity() valueobjects.ComplexityScore {
	return ar.totalComplexity
}

// HighSeverityFindings returns only findings with high severity
func (ar AnalysisResult) HighSeverityFindings() []entities.AnalysisFinding {
	var highSeverity []entities.AnalysisFinding
	for _, finding := range ar.findings {
		if finding.IsHighSeverity() {
			highSeverity = append(highSeverity, finding)
		}
	}
	return highSeverity
}

// FindingsByType returns findings grouped by type
func (ar AnalysisResult) FindingsByType() map[entities.FindingType][]entities.AnalysisFinding {
	grouped := make(map[entities.FindingType][]entities.AnalysisFinding)
	for _, finding := range ar.findings {
		grouped[finding.Type()] = append(grouped[finding.Type()], finding)
	}
	return grouped
}

// AddFinding adds a new finding to this analysis result
func (ar *AnalysisResult) AddFinding(finding entities.AnalysisFinding) error {
	// Check for duplicate IDs
	for _, existing := range ar.findings {
		if existing.ID() == finding.ID() {
			return fmt.Errorf("finding with ID %s already exists", finding.ID())
		}
	}

	ar.findings = append(ar.findings, finding)
	return nil
}

// AddAnalyzedFile adds a file to the list of analyzed files
func (ar *AnalysisResult) AddAnalyzedFile(filePath string) {
	// Avoid duplicates
	for _, existing := range ar.analyzedFiles {
		if existing == filePath {
			return
		}
	}
	ar.analyzedFiles = append(ar.analyzedFiles, filePath)
	ar.totalFiles = len(ar.analyzedFiles)
}

// SetTotalFunctions sets the total number of functions analyzed
func (ar *AnalysisResult) SetTotalFunctions(count int) {
	ar.totalFunctions = count
}

// SetTotalComplexity sets the aggregate complexity metrics
func (ar *AnalysisResult) SetTotalComplexity(complexity valueobjects.ComplexityScore) {
	ar.totalComplexity = complexity
}

// Complete marks the analysis as complete
func (ar *AnalysisResult) Complete() {
	ar.endTime = time.Now()
}

// IsComplete returns whether the analysis is complete
func (ar AnalysisResult) IsComplete() bool {
	return !ar.endTime.IsZero()
}

// Summary returns a summary of the analysis results
func (ar AnalysisResult) Summary() AnalysisSummary {
	findingsByType := ar.FindingsByType()
	highSeverityCount := len(ar.HighSeverityFindings())

	return AnalysisSummary{
		TotalFiles:         ar.totalFiles,
		TotalFunctions:     ar.totalFunctions,
		TotalFindings:      len(ar.findings),
		HighSeverityCount:  highSeverityCount,
		ComplexityFindings: len(findingsByType[entities.FindingTypeComplexity]),
		SmellFindings:      len(findingsByType[entities.FindingTypeSmell]),
		SecurityFindings:   len(findingsByType[entities.FindingTypeSecurity]),
		PerformanceFindings: len(findingsByType[entities.FindingTypePerformance]),
		Duration:           ar.Duration(),
	}
}

// AnalysisSummary provides a concise overview of analysis results
type AnalysisSummary struct {
	TotalFiles          int
	TotalFunctions      int
	TotalFindings       int
	HighSeverityCount   int
	ComplexityFindings  int
	SmellFindings       int
	SecurityFindings    int
	PerformanceFindings int
	Duration            time.Duration
}
