package valueobjects

import "fmt"

// ComplexityScore represents calculated complexity metrics for a code construct
type ComplexityScore struct {
	cyclomatic int
	cognitive  int
}

// NewComplexityScore creates a new complexity score
func NewComplexityScore(cyclomatic, cognitive int) (ComplexityScore, error) {
	if cyclomatic < 1 {
		return ComplexityScore{}, fmt.Errorf("cyclomatic complexity must be >= 1, got %d", cyclomatic)
	}
	if cognitive < 0 {
		return ComplexityScore{}, fmt.Errorf("cognitive complexity must be >= 0, got %d", cognitive)
	}

	return ComplexityScore{
		cyclomatic: cyclomatic,
		cognitive:  cognitive,
	}, nil
}

// Cyclomatic returns the cyclomatic complexity score
func (c ComplexityScore) Cyclomatic() int {
	return c.cyclomatic
}

// Cognitive returns the cognitive complexity score
func (c ComplexityScore) Cognitive() int {
	return c.cognitive
}

// IsHighComplexity checks if either metric exceeds Go-adjusted thresholds
func (c ComplexityScore) IsHighComplexity() bool {
	return c.cyclomatic > 15 || c.cognitive > 20
}

// String returns a human-readable representation
func (c ComplexityScore) String() string {
	return fmt.Sprintf("cyclomatic=%d, cognitive=%d", c.cyclomatic, c.cognitive)
}

// Equals compares two complexity scores for equality
func (c ComplexityScore) Equals(other ComplexityScore) bool {
	return c.cyclomatic == other.cyclomatic && c.cognitive == other.cognitive
}
