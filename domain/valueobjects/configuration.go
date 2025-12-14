package valueobjects

import "fmt"

// AnalysisConfiguration defines the parameters for code analysis
type AnalysisConfiguration struct {
	maxCyclomaticComplexity int
	maxCognitiveComplexity  int
	maxFunctionLength       int
	enableSmellDetection    bool
	severityThreshold       SeverityLevel
}

// SeverityLevel represents the severity of detected issues
type SeverityLevel int

const (
	SeverityInfo SeverityLevel = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// String returns a string representation of the severity level
func (s SeverityLevel) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// DefaultAnalysisConfiguration returns a configuration with Go-adjusted thresholds
func DefaultAnalysisConfiguration() AnalysisConfiguration {
	return AnalysisConfiguration{
		maxCyclomaticComplexity: 15, // Go-adjusted from academic standard of 10
		maxCognitiveComplexity:  20,
		maxFunctionLength:       80,
		enableSmellDetection:    true,
		severityThreshold:       SeverityWarning,
	}
}

// NewAnalysisConfiguration creates a custom analysis configuration
func NewAnalysisConfiguration(maxCyclomatic, maxCognitive, maxLength int, enableSmells bool, severity SeverityLevel) (AnalysisConfiguration, error) {
	if maxCyclomatic < 1 {
		return AnalysisConfiguration{}, fmt.Errorf("max cyclomatic complexity must be >= 1, got %d", maxCyclomatic)
	}
	if maxCognitive < 0 {
		return AnalysisConfiguration{}, fmt.Errorf("max cognitive complexity must be >= 0, got %d", maxCognitive)
	}
	if maxLength < 1 {
		return AnalysisConfiguration{}, fmt.Errorf("max function length must be >= 1, got %d", maxLength)
	}

	return AnalysisConfiguration{
		maxCyclomaticComplexity: maxCyclomatic,
		maxCognitiveComplexity:  maxCognitive,
		maxFunctionLength:       maxLength,
		enableSmellDetection:    enableSmells,
		severityThreshold:       severity,
	}, nil
}

// WithMaxCyclomaticComplexity returns a new configuration with updated cyclomatic complexity threshold
func (c AnalysisConfiguration) WithMaxCyclomaticComplexity(max int) AnalysisConfiguration {
	return AnalysisConfiguration{
		maxCyclomaticComplexity: max,
		maxCognitiveComplexity:  c.maxCognitiveComplexity,
		maxFunctionLength:       c.maxFunctionLength,
		enableSmellDetection:    c.enableSmellDetection,
		severityThreshold:       c.severityThreshold,
	}
}

// WithMaxCognitiveComplexity returns a new configuration with updated cognitive complexity threshold
func (c AnalysisConfiguration) WithMaxCognitiveComplexity(max int) AnalysisConfiguration {
	return AnalysisConfiguration{
		maxCyclomaticComplexity: c.maxCyclomaticComplexity,
		maxCognitiveComplexity:  max,
		maxFunctionLength:       c.maxFunctionLength,
		enableSmellDetection:    c.enableSmellDetection,
		severityThreshold:       c.severityThreshold,
	}
}

// WithMaxFunctionLength returns a new configuration with updated function length threshold
func (c AnalysisConfiguration) WithMaxFunctionLength(max int) AnalysisConfiguration {
	return AnalysisConfiguration{
		maxCyclomaticComplexity: c.maxCyclomaticComplexity,
		maxCognitiveComplexity:  c.maxCognitiveComplexity,
		maxFunctionLength:       max,
		enableSmellDetection:    c.enableSmellDetection,
		severityThreshold:       c.severityThreshold,
	}
}

// WithSmellDetection returns a new configuration with updated smell detection setting
func (c AnalysisConfiguration) WithSmellDetection(enabled bool) AnalysisConfiguration {
	return AnalysisConfiguration{
		maxCyclomaticComplexity: c.maxCyclomaticComplexity,
		maxCognitiveComplexity:  c.maxCognitiveComplexity,
		maxFunctionLength:       c.maxFunctionLength,
		enableSmellDetection:    enabled,
		severityThreshold:       c.severityThreshold,
	}
}

// MaxCyclomaticComplexity returns the maximum allowed cyclomatic complexity
func (c AnalysisConfiguration) MaxCyclomaticComplexity() int {
	return c.maxCyclomaticComplexity
}

// MaxCognitiveComplexity returns the maximum allowed cognitive complexity
func (c AnalysisConfiguration) MaxCognitiveComplexity() int {
	return c.maxCognitiveComplexity
}

// MaxFunctionLength returns the maximum allowed function length
func (c AnalysisConfiguration) MaxFunctionLength() int {
	return c.maxFunctionLength
}

// IsSmellDetectionEnabled returns whether smell detection is enabled
func (c AnalysisConfiguration) IsSmellDetectionEnabled() bool {
	return c.enableSmellDetection
}

// SeverityThreshold returns the minimum severity level to report
func (c AnalysisConfiguration) SeverityThreshold() SeverityLevel {
	return c.severityThreshold
}
