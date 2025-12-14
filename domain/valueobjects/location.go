package valueobjects

import (
	"fmt"
	"path/filepath"
)

// SourceLocation represents a location in source code
type SourceLocation struct {
	filePath string
	line     int
	column   int
}

// NewSourceLocation creates a new source location
func NewSourceLocation(filePath string, line, column int) (SourceLocation, error) {
	if filePath == "" {
		return SourceLocation{}, fmt.Errorf("file path cannot be empty")
	}
	if line < 1 {
		return SourceLocation{}, fmt.Errorf("line must be >= 1, got %d", line)
	}
	if column < 0 {
		return SourceLocation{}, fmt.Errorf("column must be >= 0, got %d", column)
	}

	// Clean the file path
	cleanPath := filepath.Clean(filePath)

	return SourceLocation{
		filePath: cleanPath,
		line:     line,
		column:   column,
	}, nil
}

// FilePath returns the file path
func (l SourceLocation) FilePath() string {
	return l.filePath
}

// Line returns the line number
func (l SourceLocation) Line() int {
	return l.line
}

// Column returns the column number
func (l SourceLocation) Column() int {
	return l.column
}

// String returns a human-readable representation
func (l SourceLocation) String() string {
	if l.column > 0 {
		return fmt.Sprintf("%s:%d:%d", l.filePath, l.line, l.column)
	}
	return fmt.Sprintf("%s:%d", l.filePath, l.line)
}

// Equals compares two source locations for equality
func (l SourceLocation) Equals(other SourceLocation) bool {
	return l.filePath == other.filePath &&
		l.line == other.line &&
		l.column == other.column
}
