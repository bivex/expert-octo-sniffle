package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goastanalyzer/application/usecases"
	"goastanalyzer/domain/entities"
	"goastanalyzer/domain/services"
	"goastanalyzer/infrastructure/adapters"
	"goastanalyzer/infrastructure/config"
)

// AnalyzerCLI provides the command-line interface for the analyzer
type AnalyzerCLI struct {
	config     config.Config
	useCase    usecases.AnalyzeCodeUseCase
	outputMode OutputMode
	recursive  bool
}

// OutputMode defines how results should be displayed
type OutputMode int

const (
	OutputModeText OutputMode = iota
	OutputModeJSON
	OutputModeTable
)

// NewAnalyzerCLI creates a new CLI instance
func NewAnalyzerCLI() *AnalyzerCLI {
	cfg := config.LoadConfig()

	// Create dependencies
	complexityCalculator := services.NewASTComplexityCalculator()
	smellDetector := services.NewASTSmellDetector()
	fileParser := adapters.NewGoFileParser()
	idGenerator := adapters.NewUUIDGenerator()

	// Create use case
	useCase := usecases.NewAnalyzeCodeUseCase(
		complexityCalculator,
		smellDetector,
		fileParser,
		idGenerator,
	)

	return &AnalyzerCLI{
		config:     cfg,
		useCase:    useCase,
		outputMode: OutputModeText,
	}
}

// Run executes the CLI application
func (cli *AnalyzerCLI) Run(args []string) int {
	var (
		files        = flag.String("files", "", "Comma-separated list of Go files to analyze")
		outputMode   = flag.String("output", "text", "Output mode: text, json, table")
		showConfig   = flag.Bool("config", false, "Show current configuration")
		help         = flag.Bool("help", false, "Show help")
		recursive    = flag.Bool("recursive", false, "Recursively analyze directories for Go files")
	)

	flag.BoolVar(recursive, "r", false, "Recursively analyze directories for Go files (short for -recursive)")
	flag.Parse()

	cli.recursive = *recursive

	if *help {
		cli.showHelp()
		return 0
	}

	if *showConfig {
		cli.showConfiguration()
		return 0
	}

	// Parse output mode
	switch strings.ToLower(*outputMode) {
	case "json":
		cli.outputMode = OutputModeJSON
	case "table":
		cli.outputMode = OutputModeTable
	default:
		cli.outputMode = OutputModeText
	}

	// Get files to analyze
	fileList := cli.parseFileList(*files, flag.Args())
	if len(fileList) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No files specified for analysis\n")
		cli.showUsage()
		return 1
	}

	// Execute analysis
	return cli.analyzeFiles(fileList)
}

// parseFileList parses the file list from command line arguments
func (cli *AnalyzerCLI) parseFileList(filesFlag string, args []string) []string {
	var files []string

	// Add files from -files flag
	if filesFlag != "" {
		fileList := strings.Split(filesFlag, ",")
		// Trim spaces
		for _, file := range fileList {
			files = append(files, strings.TrimSpace(file))
		}
	}

	// Add remaining positional arguments as files
	files = append(files, args...)

	// If recursive mode is enabled, expand directories to Go files
	if cli.recursive {
		var expandedFiles []string
		for _, path := range files {
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				// It's a directory, find Go files recursively
				goFiles, err := cli.findGoFilesRecursively(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to read directory %s: %v\n", path, err)
					continue
				}
				expandedFiles = append(expandedFiles, goFiles...)
			} else {
				// It's a file or doesn't exist (let the analyzer handle the error)
				expandedFiles = append(expandedFiles, path)
			}
		}
		return expandedFiles
	}

	return files
}

// findGoFilesRecursively finds all .go files in a directory recursively
func (cli *AnalyzerCLI) findGoFilesRecursively(rootPath string) ([]string, error) {
	var goFiles []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			goFiles = append(goFiles, path)
		}

		return nil
	})

	return goFiles, err
}

// analyzeFiles performs the analysis on the specified files
func (cli *AnalyzerCLI) analyzeFiles(files []string) int {
	request := usecases.AnalyzeCodeRequest{
		FilePaths:             files,
		Configuration:         cli.config.Analysis,
		IncludeSmellDetection: cli.config.Analysis.IsSmellDetectionEnabled(),
	}

	response, err := cli.useCase.Execute(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing analysis: %v\n", err)
		return 1
	}

	if !response.Success {
		fmt.Fprintf(os.Stderr, "Analysis failed: %v\n", response.Error)
		return 1
	}

	cli.displayResults(response)
	return 0
}

// displayResults displays the analysis results based on output mode
func (cli *AnalyzerCLI) displayResults(response *usecases.AnalyzeCodeResponse) {
	switch cli.outputMode {
	case OutputModeJSON:
		cli.displayJSON(response)
	case OutputModeTable:
		cli.displayTable(response)
	default:
		cli.displayText(response)
	}
}

// displayText displays results in human-readable text format
func (cli *AnalyzerCLI) displayText(response *usecases.AnalyzeCodeResponse) {
	fmt.Println("=== Go AST Analyzer Results ===")
	fmt.Println(response.Summary)
	fmt.Println()

	result := response.AnalysisResult

	if len(result.Findings()) == 0 {
		fmt.Println("✅ No issues found!")
		return
	}

	fmt.Printf("Found %d issues:\n\n", len(result.Findings()))

	findingsByType := result.FindingsByType()

	// Display complexity issues
	if complexityFindings := findingsByType[entities.FindingTypeComplexity]; len(complexityFindings) > 0 {
		fmt.Println("🔍 Complexity Issues:")
		for _, finding := range complexityFindings {
			fmt.Printf("  %s\n", finding.String())
		}
		fmt.Println()
	}

	// Display smell issues
	if smellFindings := findingsByType[entities.FindingTypeSmell]; len(smellFindings) > 0 {
		fmt.Println("👃 Code Smells:")
		for _, finding := range smellFindings {
			fmt.Printf("  %s\n", finding.String())
		}
		fmt.Println()
	}

	// Display other findings
	for findingType, findings := range findingsByType {
		if findingType == entities.FindingTypeComplexity || findingType == entities.FindingTypeSmell {
			continue
		}
		if len(findings) > 0 {
			fmt.Printf("%s Issues:\n", strings.Title(findingType.String()))
			for _, finding := range findings {
				fmt.Printf("  %s\n", finding.String())
			}
			fmt.Println()
		}
	}
}

// displayJSON displays results in JSON format
func (cli *AnalyzerCLI) displayJSON(response *usecases.AnalyzeCodeResponse) {
	// Simplified JSON output - in production, use proper JSON marshaling
	fmt.Printf(`{
  "success": %t,
  "summary": "%s",
  "total_findings": %d,
  "high_severity_count": %d
}`,
		response.Success,
		strings.ReplaceAll(response.Summary, `"`, `\"`),
		len(response.AnalysisResult.Findings()),
		len(response.AnalysisResult.HighSeverityFindings()),
	)
	fmt.Println()
}

// displayTable displays results in a tabular format
func (cli *AnalyzerCLI) displayTable(response *usecases.AnalyzeCodeResponse) {
	fmt.Println("FILE                 | LINE | TYPE       | SEVERITY | MESSAGE")
	fmt.Println("---------------------|------|------------|----------|--------")

	for _, finding := range response.AnalysisResult.Findings() {
		location := finding.Location()
		fmt.Printf("%-20s | %4d | %-10s | %-8s | %s\n",
			truncate(location.FilePath(), 20),
			location.Line(),
			finding.Type().String(),
			finding.Severity().String(),
			finding.Message(),
		)
	}
}

// showConfiguration displays the current configuration
func (cli *AnalyzerCLI) showConfiguration() {
	fmt.Println("=== Go AST Analyzer Configuration ===")
	fmt.Printf("Max Cyclomatic Complexity: %d\n", cli.config.Analysis.MaxCyclomaticComplexity())
	fmt.Printf("Max Cognitive Complexity:  %d\n", cli.config.Analysis.MaxCognitiveComplexity())
	fmt.Printf("Max Function Length:       %d\n", cli.config.Analysis.MaxFunctionLength())
	fmt.Printf("Smell Detection Enabled:   %t\n", cli.config.Analysis.IsSmellDetectionEnabled())
	fmt.Printf("Severity Threshold:        %s\n", cli.config.Analysis.SeverityThreshold().String())
}

// showUsage displays usage information
func (cli *AnalyzerCLI) showUsage() {
	fmt.Println("Usage: goastanalyzer [options] <files...>")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  goastanalyzer file1.go file2.go")
	fmt.Println("  goastanalyzer -files file1.go,file2.go -output table")
	fmt.Println("  goastanalyzer -recursive ./src")
	fmt.Println("  goastanalyzer -r /path/to/project")
	fmt.Println("  goastanalyzer -config")
}

// showHelp displays detailed help information
func (cli *AnalyzerCLI) showHelp() {
	cli.showUsage()
	fmt.Println()
	fmt.Println("Description:")
	fmt.Println("  Go AST Analyzer analyzes Go source code for architectural issues,")
	fmt.Println("  complexity problems, and code smells based on empirical research.")
	fmt.Println()
	fmt.Println("Recursive Analysis:")
	fmt.Println("  When using -recursive or -r flag, directories will be scanned recursively")
	fmt.Println("  for .go files. Individual files can still be specified alongside directories.")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Set environment variables to override defaults:")
	fmt.Println("  - GOAST_MAX_CYCLOMATIC: Maximum cyclomatic complexity (default: 15)")
	fmt.Println("  - GOAST_MAX_COGNITIVE:  Maximum cognitive complexity (default: 20)")
	fmt.Println("  - GOAST_MAX_FUNCTION_LENGTH: Maximum function length (default: 80)")
	fmt.Println("  - GOAST_ENABLE_SMELL_DETECTION: Enable smell detection (default: true)")
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
