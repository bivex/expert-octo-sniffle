package valueobjects

import (
	"testing"
)

func TestNewComplexityScore(t *testing.T) {
	tests := []struct {
		name        string
		cyclomatic  int
		cognitive   int
		expectError bool
	}{
		{"valid scores", 5, 8, false},
		{"zero cyclomatic", 0, 5, true},
		{"negative cyclomatic", -1, 5, true},
		{"negative cognitive", 5, -1, true},
		{"zero cognitive", 5, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := NewComplexityScore(tt.cyclomatic, tt.cognitive)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if score.Cyclomatic() != tt.cyclomatic {
					t.Errorf("expected cyclomatic %d, got %d", tt.cyclomatic, score.Cyclomatic())
				}
				if score.Cognitive() != tt.cognitive {
					t.Errorf("expected cognitive %d, got %d", tt.cognitive, score.Cognitive())
				}
			}
		})
	}
}

func TestComplexityScore_IsHighComplexity(t *testing.T) {
	tests := []struct {
		name       string
		cyclomatic int
		cognitive  int
		expected   bool
	}{
		{"normal complexity", 5, 8, false},
		{"high cyclomatic only", 20, 8, true},
		{"high cognitive only", 5, 25, true},
		{"both high", 20, 25, true},
		{"at threshold", 15, 20, false},
		{"over threshold", 16, 21, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := NewComplexityScore(tt.cyclomatic, tt.cognitive)
			result := score.IsHighComplexity()

			if result != tt.expected {
				t.Errorf("expected IsHighComplexity() = %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestComplexityScore_String(t *testing.T) {
	score, _ := NewComplexityScore(10, 15)
	expected := "cyclomatic=10, cognitive=15"

	if score.String() != expected {
		t.Errorf("expected String() = %q, got %q", expected, score.String())
	}
}

func TestComplexityScore_Equals(t *testing.T) {
	score1, _ := NewComplexityScore(10, 15)
	score2, _ := NewComplexityScore(10, 15)
	score3, _ := NewComplexityScore(11, 15)

	if !score1.Equals(score2) {
		t.Error("expected equal scores to be equal")
	}

	if score1.Equals(score3) {
		t.Error("expected different scores to not be equal")
	}
}
