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

func TestDetectDivergentChange(t *testing.T) {
	src := `package test
type Order struct {
	CustomerName string
	CustomerAddr string
	ItemSKU      string
	ItemPrice    float64
	ItemQty      int
}
func (o *Order) GetCustomerInfo() string {
	return o.CustomerName + " " + o.CustomerAddr
}
func (o *Order) UpdateCustomer(name string, addr string) {
	o.CustomerName = name
	o.CustomerAddr = addr
}
func (o *Order) CalcTotal() float64 {
	return o.ItemPrice * float64(o.ItemQty)
}
func (o *Order) UpdateItem(sku string, price float64, qty int) {
	o.ItemSKU = sku
	o.ItemPrice = price
	o.ItemQty = qty
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
		if strings.Contains(f.Message(), "Divergent Change") && strings.Contains(f.Message(), "Order") {
			found = true
		}
	}
	if !found {
		t.Error("Divergent Change not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectShotgunSurgery(t *testing.T) {
	src := `package test
type Config struct { Value string }
type HandlerA struct{}
type HandlerB struct{}
type HandlerC struct{}
type HandlerD struct{}
func (h *HandlerA) Process(cfg Config) { _ = cfg.Value }
func (h *HandlerB) Handle(cfg Config)  { _ = cfg.Value }
func (h *HandlerC) Run(cfg Config)     { _ = cfg.Value }
func (h *HandlerD) Exec(cfg Config)    { _ = cfg.Value }
func Standalone(cfg Config)            { _ = cfg.Value }
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
		if strings.Contains(f.Message(), "Shotgun Surgery") && strings.Contains(f.Message(), "Config") {
			found = true
		}
	}
	if !found {
		t.Error("Shotgun Surgery not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectTemporaryField(t *testing.T) {
	src := `package test
type Report struct {
	Title   string
	Content string
	Temp    string
	Debug   string
}
func (r *Report) Generate() string {
	r.Title = "report"
	r.Content = "data"
	return r.Title + r.Content
}
func (r *Report) Format() string {
	return r.Title + " formatted: " + r.Content
}
func (r *Report) Save() error { return nil }
func (r *Report) Load() error { return nil }
func (r *Report) Print() { r.Title = "printed" }
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
		if strings.Contains(f.Message(), "Temporary Field") && (strings.Contains(f.Message(), "Temp") || strings.Contains(f.Message(), "Debug")) {
			found = true
		}
	}
	if !found {
		t.Error("Temporary Field not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectComments(t *testing.T) {
	src := `package test
func ProcessData(x int) int {
	// First we need to validate the input
	// Check that x is positive
	// Check that x is within range
	// Check that x is not zero
	if x <= 0 {
		return 0
	}
	// Now we transform the data
	// Step 1: double it
	// Step 2: add one
	// Step 3: multiply by 3
	// Step 4: subtract 2
	y := x * 2
	y = y + 1
	y = y * 3
	y = y - 2
	// Final cleanup and validation
	// Make sure result is positive
	// Return the computed value
	return y
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
		if strings.Contains(f.Message(), "Comments") && strings.Contains(f.Message(), "ProcessData") {
			found = true
		}
	}
	if !found {
		t.Error("Comments smell not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}

func TestDetectRefusedBequest(t *testing.T) {
	src := `package test
type Base struct{}
func (b *Base) MethodA() string { return "a" }
func (b *Base) MethodB() string { return "b" }
func (b *Base) MethodC() string { return "c" }
func (b *Base) MethodD() string { return "d" }
func (b *Base) MethodE() string { return "e" }
type Child struct {
	Base
}
func (c *Child) DoWork() { _ = c.MethodA() }
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
		if strings.Contains(f.Message(), "Refused Bequest") && strings.Contains(f.Message(), "Child") {
			found = true
		}
	}
	if !found {
		t.Error("Refused Bequest not detected")
		for _, f := range findings {
			t.Logf("  Found: %s", f.Message())
		}
	}
}
