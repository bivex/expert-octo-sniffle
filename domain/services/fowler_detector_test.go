package services

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"goastanalyzer/domain/valueobjects"
)

// parseTestCode parses Go source code string for testing
func parseTestCode(src string) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	return file, fset, nil
}

func TestDetectLongParameterList(t *testing.T) {
	src := `package test
func Process(name string, age int, email string, phone string, address string, city string) {}
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewFowlerSmellDetector()
	findings, err := detector.Detect(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if strings.Contains(f.Message(), "too many parameters") && strings.Contains(f.Message(), "Process") {
			found = true
			if !strings.Contains(f.Message(), "6") {
				t.Errorf("Expected 6 parameters in message, got: %s", f.Message())
			}
		}
	}
	if !found {
		t.Errorf("Long Parameter List not detected. Findings: %d", len(findings))
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectSwitchStatements(t *testing.T) {
	src := `package test
func Classify(x int) string {
	switch x {
	case 1: return "one"
	case 2: return "two"
	case 3: return "three"
	case 4: return "four"
	case 5: return "five"
	case 6: return "six"
	case 7: return "seven"
	case 8: return "eight"
	case 9: return "nine"
	}
	return "other"
}

func HandleA(x int) {
	switch x {
	case 1: doSomething()
	}
}
func HandleB(x int) {
	switch x {
	case 2: doSomething()
	}
}
func HandleC(x int) {
	switch x {
	case 3: doSomething()
	}
}
func doSomething() {}
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewFowlerSmellDetector()
	findings, err := detector.Detect(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	foundLarge := false
	foundRepeated := false
	for _, f := range findings {
		if strings.Contains(f.Message(), "9 cases") {
			foundLarge = true
		}
		if strings.Contains(f.Message(), "appears") && strings.Contains(f.Message(), "times") {
			foundRepeated = true
		}
	}
	if !foundLarge {
		t.Error("Large switch statement not detected")
	}
	if !foundRepeated {
		t.Error("Repeated switch on same variable not detected")
	}
}

func TestDetectMessageChains(t *testing.T) {
	src := `package test
type A struct{}
func (a *A) B() *B { return nil }
type B struct{}
func (b *B) C() *C { return nil }
type C struct{}
func (c *C) D() *D { return nil }
type D struct{}
func (d *D) E() string { return "" }

func Example() {
	a := &A{}
	result := a.B().C().D().E()
	_ = result
}
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewFowlerSmellDetector()
	findings, err := detector.Detect(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if strings.Contains(f.Message(), "Message chain") && strings.Contains(f.Message(), "length 4") {
			found = true
		}
	}
	if !found {
		t.Error("Message chain not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectDataClumps(t *testing.T) {
	src := `package test
func CreateUser(name string, email string, age int) {}
func UpdateUser(name string, email string, age int) {}
func DeleteUser(name string, email string, age int) {}
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewFowlerSmellDetector()
	findings, err := detector.Detect(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if strings.Contains(f.Message(), "Data Clump") || strings.Contains(f.Message(), "appears together") {
			found = true
		}
	}
	if !found {
		t.Error("Data clump not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectPrimitiveObsession(t *testing.T) {
	src := `package test
func DrawLine(x1 int, y1 int, x2 int, color string) {}
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewFowlerSmellDetector()
	findings, err := detector.Detect(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if strings.Contains(f.Message(), "Primitive Obsession") || strings.Contains(f.Message(), "parameters of type 'int'") {
			found = true
		}
	}
	if !found {
		t.Error("Primitive obsession not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectMiddleMan(t *testing.T) {
	src := `package test
type Service struct {
	delegate *RealService
}
func (s *Service) DoA() string { return s.delegate.DoA() }
func (s *Service) DoB() int { return s.delegate.DoB() }
func (s *Service) DoC() bool { return s.delegate.DoC() }
func (s *Service) DoD() error { return s.delegate.DoD() }

type RealService struct{}
func (r *RealService) DoA() string { return "" }
func (r *RealService) DoB() int { return 0 }
func (r *RealService) DoC() bool { return false }
func (r *RealService) DoD() error { return nil }
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewFowlerSmellDetector()
	findings, err := detector.Detect(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if strings.Contains(f.Message(), "Middle Man") || strings.Contains(f.Message(), "pure delegation") {
			found = true
		}
	}
	if !found {
		t.Error("Middle man not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectSpeculativeGeneralityUnusedParam(t *testing.T) {
	src := `package test
func Calculate(a int, b int, unused string) int {
	return a + b
}
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewFowlerSmellDetector()
	findings, err := detector.Detect(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if strings.Contains(f.Message(), "unused") && strings.Contains(f.Message(), "unused") {
			found = true
		}
	}
	if !found {
		t.Error("Unused parameter (Speculative Generality) not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectDuplicatedCode(t *testing.T) {
	src := `package test
func CalculateTotal(items []int) int {
	total := 0
	for _, item := range items {
		total += item
	}
	return total
}

func CalculateSum(values []int) int {
	sum := 0
	for _, val := range values {
		sum += val
	}
	return sum
}
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewFowlerSmellDetector()
	findings, err := detector.Detect(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if strings.Contains(f.Message(), "structural similarity") && strings.Contains(f.Message(), "CalculateTotal") {
			found = true
		}
	}
	if !found {
		t.Error("Duplicated code not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestNoFalsePositivesOnCleanCode(t *testing.T) {
	src := `package test
func Add(a int, b int) int {
	return a + b
}

type Point struct {
	X int
	Y int
}

func (p *Point) Distance() float64 {
	return float64(p.X*p.X + p.Y*p.Y)
}
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewFowlerSmellDetector()
	findings, err := detector.Detect(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	if len(findings) > 0 {
		t.Errorf("Clean code should have 0 findings, got %d:", len(findings))
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestFowlerDetectorIntegrationWithSmellDetector(t *testing.T) {
	src := `package test
func LongFunc(a int, b int, c int, d int, e int, f int) int {
	return a + b + c + d + e + f
}
`
	file, fset, err := parseTestCode(src)
	if err != nil {
		t.Fatal(err)
	}

	detector := NewASTSmellDetector()
	findings, err := detector.DetectSmells(file, fset, valueobjects.DefaultAnalysisConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if strings.Contains(f.Message(), "too many parameters") {
			found = true
		}
	}
	if !found {
		t.Error("Fowler detector not wired into ASTSmellDetector")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}
