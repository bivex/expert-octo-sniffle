package entities

import (
	"fmt"
	"time"

	"goastanalyzer/domain/valueobjects"
)

// FindingType represents the type of issue found
type FindingType int

const (
	FindingTypeComplexity FindingType = iota
	FindingTypeSmell
	FindingTypeSecurity
	FindingTypePerformance
	FindingTypeBug
)

// String returns a string representation of the finding type
func (ft FindingType) String() string {
	switch ft {
	case FindingTypeComplexity:
		return "complexity"
	case FindingTypeSmell:
		return "smell"
	case FindingTypeSecurity:
		return "security"
	case FindingTypePerformance:
		return "performance"
	case FindingTypeBug:
		return "bug"
	default:
		return "unknown"
	}
}

// AnalysisFinding represents an individual issue found during code analysis
type AnalysisFinding struct {
	id          string
	findingType FindingType
	location    valueobjects.SourceLocation
	message     string
	severity    valueobjects.SeverityLevel
	timestamp   time.Time
	metadata    map[string]interface{}
}

// NewAnalysisFinding creates a new analysis finding
func NewAnalysisFinding(
	id string,
	findingType FindingType,
	location valueobjects.SourceLocation,
	message string,
	severity valueobjects.SeverityLevel,
) (AnalysisFinding, error) {

	if id == "" {
		return AnalysisFinding{}, fmt.Errorf("finding ID cannot be empty")
	}
	if message == "" {
		return AnalysisFinding{}, fmt.Errorf("finding message cannot be empty")
	}

	return AnalysisFinding{
		id:          id,
		findingType: findingType,
		location:    location,
		message:     message,
		severity:    severity,
		timestamp:   time.Now(),
		metadata:    make(map[string]interface{}),
	}, nil
}

// ID returns the unique identifier for this finding
func (f AnalysisFinding) ID() string {
	return f.id
}

// Type returns the type of finding
func (f AnalysisFinding) Type() FindingType {
	return f.findingType
}

// Location returns the source location of the finding
func (f AnalysisFinding) Location() valueobjects.SourceLocation {
	return f.location
}

// Message returns the human-readable message for this finding
func (f AnalysisFinding) Message() string {
	return f.message
}

// Severity returns the severity level of this finding
func (f AnalysisFinding) Severity() valueobjects.SeverityLevel {
	return f.severity
}

// Timestamp returns when this finding was detected
func (f AnalysisFinding) Timestamp() time.Time {
	return f.timestamp
}

// Metadata returns additional metadata for this finding
func (f AnalysisFinding) Metadata() map[string]interface{} {
	// Return a copy to prevent external modification
	metadata := make(map[string]interface{})
	for k, v := range f.metadata {
		metadata[k] = v
	}
	return metadata
}

// AddMetadata adds key-value metadata to this finding
func (f *AnalysisFinding) AddMetadata(key string, value interface{}) {
	if f.metadata == nil {
		f.metadata = make(map[string]interface{})
	}
	f.metadata[key] = value
}

// IsHighSeverity checks if this finding has high severity
func (f AnalysisFinding) IsHighSeverity() bool {
	return f.severity >= valueobjects.SeverityError
}

// String returns a human-readable representation
func (f AnalysisFinding) String() string {
	return fmt.Sprintf("[%s] %s: %s at %s",
		f.severity.String(),
		f.findingType.String(),
		f.message,
		f.location.String())
}

// Equals compares two findings for equality (excluding timestamp and metadata)
func (f AnalysisFinding) Equals(other AnalysisFinding) bool {
	return f.id == other.id &&
		f.findingType == other.findingType &&
		f.location.Equals(other.location) &&
		f.message == other.message &&
		f.severity == other.severity
}
