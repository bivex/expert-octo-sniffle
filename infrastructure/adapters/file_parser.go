package adapters

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// GoFileParser implements the FileParser interface using Go's standard library
type GoFileParser struct{}

// NewGoFileParser creates a new Go file parser
func NewGoFileParser() *GoFileParser {
	return &GoFileParser{}
}

// ParseFile parses a Go source file and returns its AST and file set
func (p *GoFileParser) ParseFile(filePath string) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()

	// Parse the file with all syntax features enabled
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	return file, fset, nil
}
