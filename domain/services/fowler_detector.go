package services

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"goastanalyzer/domain/entities"
	"goastanalyzer/domain/valueobjects"
)

// FowlerSmellDetector detects code smells from Martin Fowler's catalog (Ch.3 "Code Smells").
// Each detector maps to a specific smell described in "Refactoring: Improving the Design of Existing Code".
type FowlerSmellDetector struct{}

// NewFowlerSmellDetector creates a new Fowler smell detector
func NewFowlerSmellDetector() *FowlerSmellDetector {
	return &FowlerSmellDetector{}
}

// Detect analyzes a Go AST file for all Fowler code smells
func (d *FowlerSmellDetector) Detect(node ast.Node, fset *token.FileSet, config valueobjects.AnalysisConfiguration) ([]entities.AnalysisFinding, error) {
	var findings []entities.AnalysisFinding

	findings = append(findings, d.detectLongParameterLists(node, fset)...)
	findings = append(findings, d.detectSwitchStatements(node, fset)...)
	findings = append(findings, d.detectMessageChains(node, fset)...)
	findings = append(findings, d.detectDataClumps(node, fset)...)
	findings = append(findings, d.detectFeatureEnvy(node, fset)...)
	findings = append(findings, d.detectPrimitiveObsession(node, fset)...)
	findings = append(findings, d.detectMiddleMan(node, fset)...)
	findings = append(findings, d.detectSpeculativeGenerality(node, fset)...)
	findings = append(findings, d.detectDuplicatedCode(node, fset)...)
	findings = append(findings, d.detectDivergentChange(node, fset)...)
	findings = append(findings, d.detectShotgunSurgery(node, fset)...)
	findings = append(findings, d.detectTemporaryField(node, fset)...)
	findings = append(findings, d.detectComments(node, fset)...)
	findings = append(findings, d.detectRefusedBequest(node, fset)...)

	return findings, nil
}

// --- Long Parameter List (Fowler p.78) ---
// "More than three or four parameters is a good indicator"
// Threshold: > 5 parameters for Go (Go-adjusted)

func (d *FowlerSmellDetector) detectLongParameterLists(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil {
			return true
		}

		paramCount := countFields(fn.Type.Params.List)
		if paramCount > 5 {
			pos := fset.Position(fn.Pos())
			location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("long_parameter_list_%s_%d", fn.Name.Name, pos.Line),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Function %s has too many parameters: %d (max recommended: 5). "+
					"Fowler: 'Long Parameter List' — consider Introduce Parameter Object or Preserve Whole Object.",
					fn.Name.Name, paramCount),
				valueobjects.SeverityWarning,
			)
			findings = append(findings, finding)
		}

		return true
	})

	return findings
}

// --- Switch Statements (Fowler p.82) ---
// "One of the most obvious symptoms of object-oriented code is its comparative lack of switch statements"
// Detect large switch/type-switch statements and repeated switches on the same type

func (d *FowlerSmellDetector) detectSwitchStatements(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	// Track switch expressions to detect repeated switches on same variable
	switchVars := make(map[string]int) // var name -> count of switches
	var switchInfos []struct {
		name     string
		pos      token.Pos
		caseCount int
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.SwitchStmt:
			caseCount := countSwitchCases(stmt.Body)
			varName := getSwitchVarName(stmt.Tag)

			if caseCount > 0 {
				switchInfos = append(switchInfos, struct {
					name     string
					pos      token.Pos
					caseCount int
				}{varName, stmt.Pos(), caseCount})
			}

			if varName != "" {
				switchVars[varName]++
			}

		case *ast.TypeSwitchStmt:
			caseCount := countSwitchCases(stmt.Body)
			varName := getTypeSwitchVarName(stmt.Assign)

			if caseCount > 0 {
				switchInfos = append(switchInfos, struct {
					name     string
					pos      token.Pos
					caseCount int
				}{varName, stmt.Pos(), caseCount})
			}

			if varName != "" {
				switchVars[varName]++
			}
		}
		return true
	})

	for _, info := range switchInfos {
		pos := fset.Position(info.pos)
		location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)

		// Flag large switches (> 8 cases)
		if info.caseCount > 8 {
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("large_switch_%d", pos.Line),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Switch statement has %d cases (max recommended: 8). "+
					"Fowler: 'Switch Statements' — consider Replace Conditional with Polymorphism.",
					info.caseCount),
				valueobjects.SeverityWarning,
			)
			findings = append(findings, finding)
		}

		// Flag repeated switches on same variable (appears 3+ times)
		if info.name != "" && switchVars[info.name] >= 3 {
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("repeated_switch_%s_%d", info.name, pos.Line),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Switch on '%s' appears %d times in this file. "+
					"Fowler: 'Switch Statements' — repeated switches on same type indicate need for Replace Conditional with Polymorphism.",
					info.name, switchVars[info.name]),
				valueobjects.SeverityWarning,
			)
			findings = append(findings, finding)
		}
	}

	return findings
}

// --- Message Chains (Fowler p.84) ---
// "You see message chains as a sequence of calls: a.b().c().d()"
// Violates Law of Demeter

func (d *FowlerSmellDetector) detectMessageChains(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}

		ast.Inspect(fn.Body, func(expr ast.Node) bool {
			sel, ok := expr.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			chainLen := countSelectorChain(sel)
			if chainLen >= 4 {
				chainStr := buildChainString(sel)
				pos := fset.Position(sel.Pos())
				location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
				finding, _ := entities.NewAnalysisFinding(
					fmt.Sprintf("message_chain_%s_%d", fn.Name.Name, pos.Line),
					entities.FindingTypeSmell,
					location,
					fmt.Sprintf("Message chain of length %d: %s. "+
						"Fowler: 'Message Chains' — client is coupled to navigation structure. "+
						"Consider Hide Delegate or Extract Method.",
						chainLen, chainStr),
					valueobjects.SeverityInfo,
				)
				findings = append(findings, finding)
			}
			return true
		})

		return true
	})

	return findings
}

// --- Data Clumps (Fowler p.81) ---
// "The smell: three or four data items that regularly appear together"
// Detect parameter groups that appear in multiple functions

func (d *FowlerSmellDetector) detectDataClumps(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	// Collect all function parameter signatures
	type paramGroup struct {
		types    string
		funcName string
		pos      token.Pos
		params   []*ast.Field
	}

	var allParams []paramGroup

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil {
			return true
		}

		if len(fn.Type.Params.List) < 2 {
			return true
		}

		typeStrings := make([]string, 0, len(fn.Type.Params.List))
		for _, field := range fn.Type.Params.List {
			typeStrings = append(typeStrings, typeToString(field.Type))
		}

		allParams = append(allParams, paramGroup{
			types:    strings.Join(typeStrings, ","),
			funcName: fn.Name.Name,
			pos:      fn.Pos(),
			params:   fn.Type.Params.List,
		})

		return true
	})

	// Find clumps: same 2+ type group appearing in 3+ functions
	clumpCounts := make(map[string][]paramGroup)
	for _, pg := range allParams {
		types := strings.Split(pg.types, ",")
		// Generate all 2-type combinations
		for i := 0; i < len(types); i++ {
			for j := i + 1; j < len(types); j++ {
				key := types[i] + "," + types[j]
				clumpCounts[key] = append(clumpCounts[key], pg)
			}
		}
	}

	reported := make(map[string]bool)
	for key, groups := range clumpCounts {
		if len(groups) < 3 || reported[key] {
			continue
		}
		reported[key] = true

		// Report on the first occurrence
		pg := groups[0]
		pos := fset.Position(pg.pos)
		location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
		finding, _ := entities.NewAnalysisFinding(
			fmt.Sprintf("data_clump_%s_%d", key, pos.Line),
			entities.FindingTypeSmell,
			location,
			fmt.Sprintf("Parameter group [%s] appears together in %d functions. "+
				"Fowler: 'Data Clumps' — consider Extract Class or Introduce Parameter Object.",
				strings.ReplaceAll(key, ",", ", "), len(groups)),
			valueobjects.SeverityInfo,
		)
		findings = append(findings, finding)
	}

	return findings
}

// --- Feature Envy (Fowler p.80) ---
// "A method that seems more interested in a class other than the one it's in"
// Detect methods that call more methods/access more fields of other types than their own receiver

func (d *FowlerSmellDetector) detectFeatureEnvy(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return true
		}

		// Get receiver type name and variable name
		recvType := getReceiverTypeName(fn.Recv.List[0].Type)
		if recvType == "" {
			return true
		}

		// Collect parameter names (non-receiver)
		paramNames := make(map[string]bool)
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				for _, name := range field.Names {
					paramNames[name.Name] = true
				}
			}
		}

		// Count own receiver usage vs external usage
		ownUsage := 0
		externalUsage := make(map[string]int) // var name -> count

		ast.Inspect(fn.Body, func(expr ast.Node) bool {
			sel, ok := expr.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			// Check if this is a method call or field access on the receiver
			if ident.Name == recvType || isReceiverName(fn.Recv.List[0], ident.Name) {
				ownUsage++
			} else if paramNames[ident.Name] {
				// Parameter usage — potential Feature Envy
				externalUsage[ident.Name]++
			}
			return true
		})

		// If external usage significantly exceeds own usage, it's Feature Envy
		totalExternal := 0
		dominantName := ""
		dominantCount := 0
		for t, c := range externalUsage {
			totalExternal += c
			if c > dominantCount {
				dominantCount = c
				dominantName = t
			}
		}

		if totalExternal > ownUsage*2 && dominantCount >= 3 {
			pos := fset.Position(fn.Pos())
			location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("feature_envy_%s_%d", fn.Name.Name, pos.Line),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Method %s uses receiver %s %d times but parameter '%s' %d times. "+
					"Fowler: 'Feature Envy' — consider Move Method or Extract Method.",
					fn.Name.Name, recvType, ownUsage, dominantName, dominantCount),
				valueobjects.SeverityWarning,
			)
			findings = append(findings, finding)
		}

		return true
	})

	return findings
}

// --- Primitive Obsession (Fowler p.78) ---
// "People new to objects are usually reluctant to use small objects for simple tasks"
// Detect multiple parameters of the same primitive type that should be a value object

func (d *FowlerSmellDetector) detectPrimitiveObsession(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	primitiveTypes := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "byte": true, "rune": true,
	}

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil {
			return true
		}

		// Count occurrences of each primitive type in parameters
		typeCounts := make(map[string][]string) // type -> param names
		for _, field := range fn.Type.Params.List {
			typeStr := typeToString(field.Type)
			if primitiveTypes[typeStr] {
				for _, name := range field.Names {
					typeCounts[typeStr] = append(typeCounts[typeStr], name.Name)
				}
			}
		}

		// If 3+ params of same primitive type, flag it
		for typeStr, names := range typeCounts {
			if len(names) >= 3 {
				pos := fset.Position(fn.Pos())
				location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
				finding, _ := entities.NewAnalysisFinding(
					fmt.Sprintf("primitive_obsession_%s_%d", fn.Name.Name, pos.Line),
					entities.FindingTypeSmell,
					location,
					fmt.Sprintf("Function %s has %d parameters of type '%s': %v. "+
						"Fowler: 'Primitive Obsession' — consider Replace Data Value with Object or Introduce Parameter Object.",
						fn.Name.Name, len(names), typeStr, names),
					valueobjects.SeverityInfo,
				)
				findings = append(findings, finding)
			}
		}

		return true
	})

	return findings
}

// --- Middle Man (Fowler p.85) ---
// "You look at a class and half its methods are delegating to another class"
// Detect methods whose body is just a single delegation call

func (d *FowlerSmellDetector) detectMiddleMan(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	// First collect all struct types and count their methods
	structMethodCount := make(map[string]int)
	structDelegationCount := make(map[string]int)

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 || fn.Body == nil {
			return true
		}

		recvType := getReceiverTypeName(fn.Recv.List[0].Type)
		if recvType == "" {
			return true
		}

		structMethodCount[recvType]++

		if isDelegatingMethod(fn) {
			structDelegationCount[recvType]++
		}

		return true
	})

	// If more than half of a struct's methods are delegating, flag it
	for recvType, totalMethods := range structMethodCount {
		delegations := structDelegationCount[recvType]
		if totalMethods >= 4 && delegations > totalMethods/2 {
			// Find a method of this type to report location
			ast.Inspect(node, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
					return true
				}

				if getReceiverTypeName(fn.Recv.List[0].Type) == recvType && isDelegatingMethod(fn) {
					pos := fset.Position(fn.Pos())
					location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
					finding, _ := entities.NewAnalysisFinding(
						fmt.Sprintf("middle_man_%s_%d", recvType, pos.Line),
						entities.FindingTypeSmell,
						location,
						fmt.Sprintf("Type %s: %d of %d methods are pure delegation. "+
							"Fowler: 'Middle Man' — consider Remove Middle Man or Inline Method.",
							recvType, delegations, totalMethods),
						valueobjects.SeverityWarning,
					)
					findings = append(findings, finding)
					return false // one finding per type is enough
				}
				return true
			})
		}
	}

	return findings
}

// --- Speculative Generality (Fowler p.83) ---
// "Oh, I think we might need this someday" — unused abstraction
// Detect: unused parameters, abstract interfaces with single implementation

func (d *FowlerSmellDetector) detectSpeculativeGenerality(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	// Detect unused function parameters
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Type.Params == nil {
			return true
		}

		for _, field := range fn.Type.Params.List {
			for _, paramName := range field.Names {
				if paramName.Name == "_" || paramName.Name == "" {
					continue
				}
				if !isParamUsed(fn.Body, paramName.Name) {
					pos := fset.Position(fn.Pos())
					location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
					finding, _ := entities.NewAnalysisFinding(
						fmt.Sprintf("unused_param_%s_%s_%d", fn.Name.Name, paramName.Name, pos.Line),
						entities.FindingTypeSmell,
						location,
						fmt.Sprintf("Parameter '%s' of function %s is never used. "+
							"Fowler: 'Speculative Generality' — remove unused parameters or use Remove Parameter.",
							paramName.Name, fn.Name.Name),
						valueobjects.SeverityInfo,
					)
					findings = append(findings, finding)
				}
			}
		}
		return true
	})

	// Detect single-implementation interfaces in the same file
	findings = append(findings, d.detectSingleImplInterfaces(node, fset)...)

	return findings
}

// --- Duplicated Code (Fowler p.76) ---
// "Number one in the stink parade is duplicated code"
// Detect structurally similar function bodies via AST subtree comparison

func (d *FowlerSmellDetector) detectDuplicatedCode(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	// Collect all function bodies
	type funcInfo struct {
		name    string
		body    *ast.BlockStmt
		pos     token.Pos
		sigHash string // signature hash for quick comparison
	}

	var funcs []funcInfo

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil || len(fn.Body.List) < 3 {
			return true
		}

		funcs = append(funcs, funcInfo{
			name:    fn.Name.Name,
			body:    fn.Body,
			pos:     fn.Pos(),
			sigHash: computeBodyHash(fn.Body),
		})
		return true
	})

	// Compare function bodies for structural similarity
	reported := make(map[string]bool)
	for i := 0; i < len(funcs); i++ {
		for j := i + 1; j < len(funcs); j++ {
			if len(funcs[i].body.List) < 3 || len(funcs[j].body.List) < 3 {
				continue
			}

			key := fmt.Sprintf("%s_%s", funcs[i].name, funcs[j].name)
			if reported[key] {
				continue
			}

			similarity := computeSimilarity(funcs[i].body, funcs[j].body)
			if similarity >= 0.8 {
				reported[key] = true
				pos := fset.Position(funcs[i].pos)
				location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
				finding, _ := entities.NewAnalysisFinding(
					fmt.Sprintf("duplicated_code_%s_%s_%d", funcs[i].name, funcs[j].name, pos.Line),
					entities.FindingTypeSmell,
					location,
					fmt.Sprintf("Functions %s and %s have %.0f%% structural similarity. "+
						"Fowler: 'Duplicated Code' — consider Extract Method, Pull Up Method, or Form Template Method.",
						funcs[i].name, funcs[j].name, similarity*100),
					valueobjects.SeverityWarning,
				)
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

// --- Divergent Change (Fowler p.79) ---
// "When you want to change one thing, you have to make lots of changes to different classes"
// Detect: struct where methods access disjoint field groups — the class changes for different reasons

func (d *FowlerSmellDetector) detectDivergentChange(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	type methodAccess struct {
		name   string
		fields map[string]bool
		pos    token.Pos
	}

	structMethods := make(map[string][]methodAccess)

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return true
		}
		recvType := getReceiverTypeName(fn.Recv.List[0].Type)
		recvVar := ""
		if len(fn.Recv.List[0].Names) > 0 {
			recvVar = fn.Recv.List[0].Names[0].Name
		}
		if recvType == "" || recvVar == "" {
			return true
		}

		accessed := make(map[string]bool)
		ast.Inspect(fn.Body, func(expr ast.Node) bool {
			sel, ok := expr.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if ok && ident.Name == recvVar {
				accessed[sel.Sel.Name] = true
			}
			return true
		})

		if len(accessed) > 0 {
			structMethods[recvType] = append(structMethods[recvType], methodAccess{
				name: fn.Name.Name, fields: accessed, pos: fn.Pos(),
			})
		}
		return true
	})

	for structName, methods := range structMethods {
		if len(methods) < 4 {
			continue
		}
		zeroOverlapPairs := 0
		for i := 0; i < len(methods); i++ {
			for j := i + 1; j < len(methods); j++ {
				if !hasOverlap(methods[i].fields, methods[j].fields) {
					zeroOverlapPairs++
				}
			}
		}
		if zeroOverlapPairs >= 1 {
			pos := methods[0].pos
			location, _ := valueobjects.NewSourceLocation(fset.Position(pos).Filename, fset.Position(pos).Line, 0)
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("divergent_change_%s", structName),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Type %s has %d methods with %d zero-overlap method pairs (methods accessing disjoint field sets). "+
					"Fowler: 'Divergent Change' - consider Extract Class to separate concerns.",
					structName, len(methods), zeroOverlapPairs),
				valueobjects.SeverityWarning,
			)
			findings = append(findings, finding)
		}
	}

	return findings
}

func (d *FowlerSmellDetector) detectShotgunSurgery(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	type paramUsage struct {
		recvTypes map[string]bool
		funcNames []string
		pos       token.Pos
	}

	// Track parameter types and which receiver types use them
	paramUsageMap := make(map[string]*paramUsage) // param type name -> usage

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil {
			return true
		}

		recvType := ""
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			recvType = getReceiverTypeName(fn.Recv.List[0].Type)
		}

		for _, field := range fn.Type.Params.List {
			typeName := extractTypeName(field.Type)
			if typeName == "" || isGoPrimitive(typeName) {
				continue
			}

			if paramUsageMap[typeName] == nil {
				paramUsageMap[typeName] = &paramUsage{
					recvTypes: make(map[string]bool),
				}
			}

			if recvType != "" {
				paramUsageMap[typeName].recvTypes[recvType] = true
			}
			paramUsageMap[typeName].funcNames = append(paramUsageMap[typeName].funcNames, fn.Name.Name)
			if paramUsageMap[typeName].pos == token.NoPos {
				paramUsageMap[typeName].pos = fn.Pos()
			}
		}

		return true
	})

	for typeName, usage := range paramUsageMap {
		// If a type is used across 3+ different receiver types in 5+ functions
		if len(usage.recvTypes) >= 3 && len(usage.funcNames) >= 5 {
			pos := usage.pos
			location, _ := valueobjects.NewSourceLocation(fset.Position(pos).Filename, fset.Position(pos).Line, 0)
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("shotgun_surgery_%s", typeName),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Type %s is passed as parameter to %d functions across %d different receiver types (%v). "+
					"Fowler: 'Shotgun Surgery' — consider Move Method or Extract Class to collocate.",
					typeName, len(usage.funcNames), len(usage.recvTypes), mapKeys(usage.recvTypes)),
				valueobjects.SeverityWarning,
			)
			findings = append(findings, finding)
		}
	}

	return findings
}

// --- Temporary Field (Fowler p.84) ---
// "Instance variables that are set only in certain circumstances"
// Detect: struct fields that are accessed only in a small subset of methods

func (d *FowlerSmellDetector) detectTemporaryField(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	// Collect struct definitions with their field names
	structFields := make(map[string][]string) // struct name -> field names
	var structPositions []struct {
		name string
		pos  token.Pos
	}

	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok || structType.Fields == nil {
				continue
			}

			var fields []string
			for _, f := range structType.Fields.List {
				for _, name := range f.Names {
					fields = append(fields, name.Name)
				}
			}
			if len(fields) >= 4 {
				structFields[typeSpec.Name.Name] = fields
				structPositions = append(structPositions, struct {
					name string
					pos  token.Pos
				}{typeSpec.Name.Name, typeSpec.Pos()})
			}
		}
		return true
	})

	// For each struct, track which fields are accessed in which methods
	type fieldUsage struct {
		fieldName  string
		methods    []string
	}

	for structName, fields := range structFields {
		methodCount := 0
		fieldAccessCount := make(map[string][]string) // field -> methods that access it

		ast.Inspect(node, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv == nil || len(fn.Recv.List) == 0 {
				return true
			}

			recvType := getReceiverTypeName(fn.Recv.List[0].Type)
			if recvType != structName {
				return true
			}

			methodCount++
			recvVar := ""
			if len(fn.Recv.List[0].Names) > 0 {
				recvVar = fn.Recv.List[0].Names[0].Name
			}

			ast.Inspect(fn.Body, func(expr ast.Node) bool {
				sel, ok := expr.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if ok && ident.Name == recvVar {
					for _, f := range fields {
						if sel.Sel.Name == f {
							fieldAccessCount[f] = append(fieldAccessCount[f], fn.Name.Name)
						}
					}
				}
				return true
			})
			return true
		})

		if methodCount < 4 {
			continue
		}

		// Find fields accessed by <=1 methods
		for _, fieldName := range fields {
			methods := fieldAccessCount[fieldName]
			if len(methods) <= 1 {
				// Find struct position for reporting
				for _, sp := range structPositions {
					if sp.name == structName {
						pos := sp.pos
						location, _ := valueobjects.NewSourceLocation(fset.Position(pos).Filename, fset.Position(pos).Line, 0)
						methodStr := "no methods"
						if len(methods) == 1 {
							methodStr = fmt.Sprintf("only method %s", methods[0])
						}
						finding, _ := entities.NewAnalysisFinding(
							fmt.Sprintf("temporary_field_%s_%s", structName, fieldName),
							entities.FindingTypeSmell,
							location,
							fmt.Sprintf("Field %s.%s is accessed in %s out of %d methods. "+
								"Fowler: 'Temporary Field' — consider Extract Class or Move Field.",
								structName, fieldName, methodStr, methodCount),
							valueobjects.SeverityInfo,
						)
						findings = append(findings, finding)
						break
					}
				}
			}
		}
	}

	return findings
}

// --- Comments (Fowler p.86) ---
// "When you feel the need to write a comment, first try to refactor the code so that any comment becomes superfluous"
// Detect: functions with excessive comment-to-code ratio

func (d *FowlerSmellDetector) detectComments(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}

		startPos := fset.Position(fn.Body.Lbrace)
		endPos := fset.Position(fn.Body.Rbrace)
		codeLines := endPos.Line - startPos.Line
		if codeLines < 10 {
			return true
		}

		commentLines := 0
		if file, ok := node.(*ast.File); ok {
			for _, cg := range file.Comments {
				for _, c := range cg.List {
					cpos := fset.Position(c.Pos())
					if cpos.Line > startPos.Line && cpos.Line < endPos.Line {
						if len(c.Text) >= 2 && c.Text[0:2] == "/*" {
							commentLines += countLines(c.Text)
						} else {
							commentLines++
						}
					}
				}
			}
		}

		if commentLines > 0 && float64(commentLines)/float64(codeLines) > 0.4 {
			pos := fset.Position(fn.Pos())
			location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("comments_%s_%d", fn.Name.Name, pos.Line),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Function %s has %d comment lines out of %d total (%.0f%%). "+
					"Fowler: 'Comments' — excessive comments often indicate code that needs refactoring. "+
					"Consider Extract Method or Rename to make code self-documenting.",
					fn.Name.Name, commentLines, codeLines, float64(commentLines)/float64(codeLines)*100),
				valueobjects.SeverityInfo,
			)
			findings = append(findings, finding)
		}

		return true
	})

	return findings
}

func (d *FowlerSmellDetector) detectRefusedBequest(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	// Collect all methods by receiver type
	methodsByType := make(map[string]map[string]bool) // type -> {method name: true}
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return true
		}
		recvType := getReceiverTypeName(fn.Recv.List[0].Type)
		if methodsByType[recvType] == nil {
			methodsByType[recvType] = make(map[string]bool)
		}
		methodsByType[recvType][fn.Name.Name] = true
		return true
	})

	// Find structs with embedded types
	type embedInfo struct {
		parentType   string
		embeddedType string
		pos          token.Pos
	}
	var embeddings []embedInfo

	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok || structType.Fields == nil {
				continue
			}
			for _, f := range structType.Fields.List {
				// Anonymous (embedded) field — no names, just a type
				if len(f.Names) == 0 {
					embeddedName := extractTypeName(f.Type)
					if embeddedName != "" && embeddedName != typeSpec.Name.Name {
						embeddings = append(embeddings, embedInfo{
							parentType:   typeSpec.Name.Name,
							embeddedType: embeddedName,
							pos:          typeSpec.Pos(),
						})
					}
				}
			}
		}
		return true
	})

	// For each embedding, check how many inherited methods are actually used
	for _, emb := range embeddings {
		inheritedMethods := methodsByType[emb.embeddedType]
		if len(inheritedMethods) < 3 {
			continue
		}

		// Check which inherited methods are called on the parent type
		usedCount := 0
		for method := range inheritedMethods {
			// Check if parent type or any code calls parentType.method()
			used := false
			ast.Inspect(node, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if sel.Sel.Name == method {
					// Check if called on a variable of parentType
					if ident, ok := sel.X.(*ast.Ident); ok {
						if isLikelyTypeRef(ident.Name) || ident.Name == strings.ToLower(emb.parentType[:1]) {
							used = true
							return false
						}
					}
				}
				return true
			})
			if used {
				usedCount++
			}
		}

		// If less than 30% of inherited methods are used, flag it
		usageRatio := float64(usedCount) / float64(len(inheritedMethods))
		if usageRatio < 0.3 {
			pos := emb.pos
			location, _ := valueobjects.NewSourceLocation(fset.Position(pos).Filename, fset.Position(pos).Line, 0)
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("refused_bequest_%s_%s", emb.parentType, emb.embeddedType),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Type %s embeds %s (%d methods) but only uses %d of them (%.0f%%). "+
					"Fowler: 'Refused Bequest' — consider Replace Inheritance with Delegation "+
					"or extract the used methods into a smaller interface.",
					emb.parentType, emb.embeddedType, len(inheritedMethods), usedCount, usageRatio*100),
				valueobjects.SeverityWarning,
			)
			findings = append(findings, finding)
		}
	}

	return findings
}

// ======== Helper functions ========

// extractTypeName gets the base type name from an AST expression
func extractTypeName(t ast.Expr) string {
	switch v := t.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return extractTypeName(v.X)
	case *ast.SelectorExpr:
		return v.Sel.Name
	case *ast.ArrayType:
		return extractTypeName(v.Elt)
	}
	return ""
}

// isGoPrimitive checks if a type name is a Go primitive
func isGoPrimitive(name string) bool {
	prims := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "byte": true, "rune": true,
		"error": true,
	}
	return prims[name]
}

// mapKeys returns the keys of a map[string]bool as a slice
func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// countLines counts the number of newlines in a string
func countLines(s string) int {
	n := 0
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n + 1
}

// countFields counts the actual number of parameters (names in each field)
func countFields(fields []*ast.Field) int {
	count := 0
	for _, f := range fields {
		if len(f.Names) == 0 {
			count++ // unnamed parameter
		} else {
			count += len(f.Names)
		}
	}
	return count
}

// countSwitchCases counts case clauses in a switch body
func countSwitchCases(body *ast.BlockStmt) int {
	count := 0
	for _, stmt := range body.List {
		if _, ok := stmt.(*ast.CaseClause); ok {
			count++
		} else if _, ok := stmt.(*ast.CommClause); ok {
			count++
		}
	}
	return count
}

// getSwitchVarName extracts the variable name from a switch tag expression
func getSwitchVarName(tag ast.Expr) string {
	if tag == nil {
		return ""
	}
	if ident, ok := tag.(*ast.Ident); ok {
		return ident.Name
	}
	if sel, ok := tag.(*ast.SelectorExpr); ok {
		return fmt.Sprintf("%s.%s", getSwitchVarName(sel.X), sel.Sel.Name)
	}
	if call, ok := tag.(*ast.CallExpr); ok {
		return getSwitchVarName(call.Fun)
	}
	return ""
}

// getTypeSwitchVarName extracts the variable name from a type switch assignment
func getTypeSwitchVarName(assign ast.Stmt) string {
	if assign == nil {
		return ""
	}
	if short, ok := assign.(*ast.AssignStmt); ok {
		if len(short.Lhs) > 0 {
			if ident, ok := short.Lhs[0].(*ast.Ident); ok {
				return ident.Name
			}
		}
	}
	return ""
}

// countSelectorChain counts the depth of chained selector expressions
// Handles method call chains like a.B().C().D() where CallExpr wraps SelectorExpr
func countSelectorChain(expr ast.Expr) int {
	count := 0
	for {
		sel, ok := expr.(*ast.SelectorExpr)
		if !ok {
			break
		}
		count++
		// If X is a call expression (method call result), unwrap it
		if call, ok := sel.X.(*ast.CallExpr); ok {
			expr = call.Fun
		} else {
			expr = sel.X
		}
	}
	return count
}

// buildChainString reconstructs a readable chain string
func buildChainString(sel *ast.SelectorExpr) string {
	parts := []string{}
	expr := ast.Expr(sel)
	for {
		s, ok := expr.(*ast.SelectorExpr)
		if !ok {
			break
		}
		parts = append([]string{s.Sel.Name}, parts...)
		expr = s.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		parts = append([]string{ident.Name}, parts...)
	}
	return strings.Join(parts, ".")
}

// typeToString converts an AST type expression to a string representation
func typeToString(t ast.Expr) string {
	switch v := t.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "*" + typeToString(v.X)
	case *ast.SelectorExpr:
		return typeToString(v.X) + "." + v.Sel.Name
	case *ast.ArrayType:
		if v.Len == nil {
			return "[]" + typeToString(v.Elt)
		}
		return fmt.Sprintf("[%s]%s", typeToString(v.Len), typeToString(v.Elt))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", typeToString(v.Key), typeToString(v.Value))
	case *ast.ChanType:
		return "chan " + typeToString(v.Value)
	case *ast.FuncType:
		return "func"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + typeToString(v.Elt)
	default:
		return "unknown"
	}
}

// getReceiverTypeName extracts the type name from a receiver declaration
func getReceiverTypeName(t ast.Expr) string {
	switch v := t.(type) {
	case *ast.StarExpr:
		return getReceiverTypeName(v.X)
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return v.Sel.Name
	}
	return ""
}

// isReceiverName checks if the given identifier name matches the receiver variable name
func isReceiverName(field *ast.Field, name string) bool {
	for _, n := range field.Names {
		if n.Name == name {
			return true
		}
	}
	return false
}

// isLikelyTypeRef checks if an identifier looks like a type reference (PascalCase exported)
func isLikelyTypeRef(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z' && !isGoBuiltin(name)
}

func isGoBuiltin(name string) bool {
	builtins := map[string]bool{
		"true": true, "false": true, "nil": true,
		"len": true, "cap": true, "make": true, "new": true,
		"append": true, "copy": true, "delete": true,
		"panic": true, "recover": true, "close": true,
		"fmt": true, "log": true, "os": true, "io": true,
		"err": true, "ctx": true, "req": true, "resp": true,
	}
	return builtins[name]
}

// isDelegatingMethod checks if a method's body is just a single call to another object's method
func isDelegatingMethod(fn *ast.FuncDecl) bool {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}

	stmt := fn.Body.List[0]

	// Check for simple expression statement that is a method call
	if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
		return isForeignMethodCall(exprStmt.X)
	}

	// Check for return statement that just delegates
	if retStmt, ok := stmt.(*ast.ReturnStmt); ok {
		if len(retStmt.Results) == 1 {
			return isForeignMethodCall(retStmt.Results[0])
		}
	}

	return false
}

// isForeignMethodCall checks if expression is a method call on a non-receiver object
func isForeignMethodCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Accept both direct identifier (x.Method()) and selector chain (x.field.Method())
	switch sel.X.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return true
	}
	return false
}

// isParamUsed checks if a parameter is used within a function body
func isParamUsed(body *ast.BlockStmt, paramName string) bool {
	used := false
	ast.Inspect(body, func(n ast.Node) bool {
		if used {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if ok && ident.Name == paramName {
			used = true
			return false
		}
		return true
	})
	return used
}

// detectSingleImplInterfaces finds interfaces with only one implementation in the same file
func (d *FowlerSmellDetector) detectSingleImplInterfaces(node ast.Node, fset *token.FileSet) []entities.AnalysisFinding {
	var findings []entities.AnalysisFinding

	// Collect interface method sets
	type ifaceInfo struct {
		name    string
		methods map[string]bool
		pos     token.Pos
	}

	var interfaces []ifaceInfo

	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			iface, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok || iface.Methods == nil {
				continue
			}

			methods := make(map[string]bool)
			for _, method := range iface.Methods.List {
				if len(method.Names) > 0 {
					methods[method.Names[0].Name] = true
				}
			}

			if len(methods) >= 3 { // Only flag non-trivial interfaces
				interfaces = append(interfaces, ifaceInfo{
					name:    typeSpec.Name.Name,
					methods: methods,
					pos:     typeSpec.Pos(),
				})
			}
		}
		return true
	})

	// Count implementations (structs with methods matching the interface)
	for _, iface := range interfaces {
		implCount := 0
		ast.Inspect(node, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
				return true
			}

			if iface.methods[fn.Name.Name] {
				implCount++
			}
			return true
		})

		// Rough heuristic: if total method matches equal interface size, likely single impl
		if implCount == len(iface.methods) && len(iface.methods) >= 3 {
			pos := fset.Position(iface.pos)
			location, _ := valueobjects.NewSourceLocation(pos.Filename, pos.Line, pos.Column)
			finding, _ := entities.NewAnalysisFinding(
				fmt.Sprintf("speculative_generality_%s_%d", iface.name, pos.Line),
				entities.FindingTypeSmell,
				location,
				fmt.Sprintf("Interface %s may have only one implementation in this file. "+
					"Fowler: 'Speculative Generality' — if an interface has only one implementor, "+
					"consider Collapse Hierarchy or Inline Class unless you need the abstraction.",
					iface.name),
				valueobjects.SeverityInfo,
			)
			findings = append(findings, finding)
		}
	}

	return findings
}

// computeBodyHash creates a structural hash of a function body for quick comparison
func computeBodyHash(body *ast.BlockStmt) string {
	var b strings.Builder
	ast.Inspect(body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.AssignStmt:
			b.WriteString("a")
		case *ast.ReturnStmt:
			b.WriteString("r")
		case *ast.IfStmt:
			b.WriteString("i")
		case *ast.ForStmt:
			b.WriteString("f")
		case *ast.CallExpr:
			b.WriteString("c")
		case *ast.SelectorExpr:
			b.WriteString("s")
		case *ast.BinaryExpr:
			b.WriteString("b")
		case *ast.IndexExpr:
			b.WriteString("x")
		}
		return true
	})
	return b.String()
}

// computeSimilarity computes structural similarity between two function bodies (0.0 - 1.0)
func computeSimilarity(body1, body2 *ast.BlockStmt) float64 {
	stmts1 := normalizeStatements(body1.List)
	stmts2 := normalizeStatements(body2.List)

	if len(stmts1) == 0 && len(stmts2) == 0 {
		return 1.0
	}
	if len(stmts1) == 0 || len(stmts2) == 0 {
		return 0.0
	}

	// Use LCS-based similarity on statement type sequences
	lcs := longestCommonSubsequence(stmts1, stmts2)
	maxLen := len(stmts1)
	if len(stmts2) > maxLen {
		maxLen = len(stmts2)
	}

	return float64(lcs) / float64(maxLen)
}

// normalizeStatements converts statements to type-based tokens for comparison
func normalizeStatements(stmts []ast.Stmt) []string {
	var result []string
	for _, stmt := range stmts {
		result = append(result, normalizeStatement(stmt)...)
	}
	return result
}

func normalizeStatement(stmt ast.Stmt) []string {
	var tokens []string

	switch v := stmt.(type) {
	case *ast.ExprStmt:
		tokens = append(tokens, normalizeExpr(v.X)...)
	case *ast.AssignStmt:
		tokens = append(tokens, "assign")
		for _, expr := range v.Rhs {
			tokens = append(tokens, normalizeExpr(expr)...)
		}
	case *ast.ReturnStmt:
		tokens = append(tokens, "return")
		for _, expr := range v.Results {
			tokens = append(tokens, normalizeExpr(expr)...)
		}
	case *ast.IfStmt:
		tokens = append(tokens, "if")
		if v.Body != nil {
			tokens = append(tokens, normalizeStatements(v.Body.List)...)
		}
	case *ast.ForStmt:
		tokens = append(tokens, "for")
		if v.Body != nil {
			tokens = append(tokens, normalizeStatements(v.Body.List)...)
		}
	case *ast.RangeStmt:
		tokens = append(tokens, "range")
		if v.Body != nil {
			tokens = append(tokens, normalizeStatements(v.Body.List)...)
		}
	case *ast.DeferStmt:
		tokens = append(tokens, "defer")
		tokens = append(tokens, normalizeExpr(v.Call)...)
	default:
		tokens = append(tokens, "stmt")
	}

	return tokens
}

func normalizeExpr(expr ast.Expr) []string {
	switch v := expr.(type) {
	case *ast.CallExpr:
		var tokens []string
		tokens = append(tokens, "call")
		if sel, ok := v.Fun.(*ast.SelectorExpr); ok {
			tokens = append(tokens, "sel:"+sel.Sel.Name)
		} else if ident, ok := v.Fun.(*ast.Ident); ok {
			tokens = append(tokens, "fn:"+ident.Name)
		}
		return tokens
	case *ast.SelectorExpr:
		return []string{"sel:" + v.Sel.Name}
	case *ast.Ident:
		return []string{"id"}
	case *ast.BasicLit:
		return []string{"lit"}
	case *ast.BinaryExpr:
		return append(normalizeExpr(v.X), normalizeExpr(v.Y)...)
	case *ast.UnaryExpr:
		return normalizeExpr(v.X)
	case *ast.IndexExpr:
		return normalizeExpr(v.X)
	default:
		return []string{"expr"}
	}
}

// longestCommonSubsequence computes LCS length
func longestCommonSubsequence(a, b []string) int {
	m, n := len(a), len(b)
	if m == 0 || n == 0 {
		return 0
	}

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] > dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	return dp[m][n]
}

// hasOverlap checks if two field sets share any fields
func hasOverlap(a, b map[string]bool) bool {
	for k := range a {
		if b[k] {
			return true
		}
	}
	return false
}
