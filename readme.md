# Go AST Analyzer

[![Go Version](https://img.shields.io/badge/go-1.25.5-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/username/goastanalyzer)](https://goreportcard.com/report/github.com/username/goastanalyzer)

**Go AST Analyzer** is a comprehensive static analysis tool for Go code quality assessment. Built on empirical research and clean architecture principles, it detects complexity issues, code smells, and concurrency bugs in Go applications.

## 🚀 Key Features

### 🔍 **Advanced Complexity Analysis**
- **Cyclomatic Complexity**: McCabe's algorithm with Go-adjusted thresholds (10-15)
- **Cognitive Complexity**: SonarSource algorithm penalizing nested structures
- **Function Length Analysis**: Configurable line and statement limits

### 👃 **Architectural Smell Detection**
- **God Package/Struct**: Identifies bloated packages and oversized structs
- **Interface Pollution**: Detects interfaces with too many methods (>7)
- **Function Length Violations**: Configurable max lines (default: 80)
- **Deep Nesting**: Configurable nesting depth limits (default: 4)

### 🐛 **Concurrency Bug Detection**
- **Goroutine Leaks**: Channel receive operations without close detection
- **Channel Misuse**: Select statements without default/timeout cases
- **Blocking Operations**: Potential deadlock scenarios

### 🏗️ **Clean Architecture Design**
- **Domain-Driven Design**: Core business logic separated from infrastructure
- **SOLID Principles**: Single responsibility, dependency inversion, interface segregation
- **Testable Architecture**: Dependency injection enables 100% test coverage
- **Extensible Framework**: Plugin architecture for new analysis types

## 📋 Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Configuration](#configuration)
- [Analysis Types](#analysis-types)
- [Research Foundation](#research-foundation)
- [Architecture](#architecture)
- [Contributing](#contributing)
- [License](#license)

## 🔧 Installation

### From Source
```bash
git clone https://github.com/username/goastanalyzer.git
cd goastanalyzer
go build -o goastanalyzer
```

### Direct Download
```bash
# Download the latest binary for your platform
curl -L https://github.com/username/goastanalyzer/releases/latest/download/goastanalyzer-$(uname -s)-$(uname -m) -o goastanalyzer
chmod +x goastanalyzer
```

## 🚀 Quick Start

Analyze a single Go file:
```bash
./goastanalyzer sample.go
```

Analyze an entire project recursively:
```bash
./goastanalyzer -recursive ./src
```

Generate JSON output for CI/CD integration:
```bash
./goastanalyzer -output json -recursive ./ > analysis.json
```

## 📖 Usage

### Command Line Options

```
Usage: goastanalyzer [options] <files...>

Options:
  -files string
        Comma-separated list of Go files to analyze
  -output string
        Output mode: text, json, table (default "text")
  -recursive, -r
        Recursively analyze directories for Go files
  -config
        Show current configuration
  -help
        Show help information

Examples:
  goastanalyzer file1.go file2.go
  goastanalyzer -files file1.go,file2.go -output table
  goastanalyzer -recursive ./src
  goastanalyzer -r /path/to/project
  goastanalyzer -config
```

### Output Formats

#### Text Output (Default)
```
=== Go AST Analyzer Results ===
Analysis complete: 114 files, 680 functions analyzed. Found 181 issues (78 high severity) in 23.464292ms.

Found 181 issues:

🔍 Complexity Issues:
  [error] complexity: Function syncClean: cyclomatic=11, cognitive=69 at yay/clean.go:51:1
  [error] complexity: Function cleanAUR: cyclomatic=17, cognitive=154 at yay/clean.go:103:1

👃 Code Smells:
  [warning] smell: Function main is too long: 109 lines (max: 80) at yay/main.go:46:1
  [warning] smell: Interface Executor has too many methods: 27 (max recommended: 7) at yay/db/executor.go:38:6

🐛 Bug Issues:
  [warning] bug: Potential blocking bug in downloadPKGBUILDSourceFanout: channel misuse detected at yay/sync/workdir/aur_source.go:70:1
```

#### JSON Output
```json
{
  "success": true,
  "summary": "Analysis complete: 114 files, 680 functions analyzed. Found 181 issues (78 high severity) in 23.464292ms.",
  "total_findings": 181,
  "high_severity_count": 78
}
```

#### Table Output
```
FILE                 | LINE | TYPE       | SEVERITY | MESSAGE
---------------------|------|------------|----------|--------
yay/clean.go         | 51   | complexity | error    | Function syncClean: cyclomatic=11, cognitive=69
yay/clean.go         | 103  | complexity | error    | Function cleanAUR: cyclomatic=17, cognitive=154
yay/main.go          | 46   | smell      | warning  | Function main is too long: 109 lines (max: 80)
```

## ⚙️ Configuration

Configure analysis parameters using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `GOAST_MAX_CYCLOMATIC` | 15 | Maximum cyclomatic complexity |
| `GOAST_MAX_COGNITIVE` | 20 | Maximum cognitive complexity |
| `GOAST_MAX_FUNCTION_LENGTH` | 80 | Maximum function length (lines) |
| `GOAST_ENABLE_SMELL_DETECTION` | true | Enable code smell detection |

### Example Configuration
```bash
export GOAST_MAX_CYCLOMATIC=10
export GOAST_MAX_COGNITIVE=15
export GOAST_MAX_FUNCTION_LENGTH=60
./goastanalyzer -recursive ./src
```

## 🔬 Analysis Types

### Complexity Analysis
- **Cyclomatic Complexity**: Measures decision points in code (if, for, switch, etc.)
- **Cognitive Complexity**: Penalizes nested boolean expressions and complex logic
- **Function Metrics**: Lines of code, statement count, parameter count

### Architectural Smells
- **God Objects**: Structs/packages with too many responsibilities
- **Interface Bloat**: Interfaces with excessive methods (>7 recommended)
- **Function Length**: Functions exceeding line/statement limits
- **Nesting Depth**: Deeply nested code structures

### Concurrency Bugs
- **Goroutine Leaks**: Unclosed channels, missing context cancellation
- **Channel Misuse**: Blocking select statements without timeouts
- **Race Conditions**: Potential data races in concurrent operations

## 📚 Research Foundation

This analyzer is built on extensive empirical research into Go code quality:

### Key Research Findings
- **58% of Go concurrency bugs** stem from channel misuse, not shared memory
- **Cyclomatic complexity threshold**: 10-15 (adjusted for Go's error handling patterns)
- **Cognitive complexity**: Better reflects human comprehension than cyclomatic alone
- **Goroutine leaks**: Can cause up to 9.2× memory overhead and 34% increased CPU consumption

### Academic Sources
- Tu et al. (2019) ASPLOS: Analysis of 171 concurrency bugs across Docker, Kubernetes, etcd
- Saioc et al. (2023): Dynamic detection of 857 goroutine leaks in 75M+ lines of production Go code
- Chabbi & Ramanathan (2022) PLDI: 1,000+ production data races analysis
- Costa et al. (2021): Unsafe package usage patterns across 2,438 Go projects

## 🏛️ Architecture

Built using Clean Architecture principles with Domain-Driven Design:

```
Presentation Layer (CLI) → Application Layer (Use Cases) → Domain Layer (Entities, Services) ← Infrastructure Layer (Adapters)
```

### Layer Responsibilities
- **Domain**: Core analysis logic, independent of frameworks
- **Application**: Use cases orchestrating domain operations
- **Infrastructure**: External dependencies (file parsing, configuration)
- **Presentation**: CLI interface with multiple output formats

### Quality Attributes
- **Testability**: 100% domain logic test coverage through dependency injection
- **Maintainability**: Single responsibility principle across all modules
- **Extensibility**: Plugin architecture for new analysis types
- **Performance**: Sub-millisecond analysis for typical Go files

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup
```bash
git clone https://github.com/username/goastanalyzer.git
cd goastanalyzer
go mod download
go test ./...
```

### Adding New Analysis Types
1. Implement domain service interface
2. Add configuration parameters
3. Update CLI output formatting
4. Add comprehensive tests

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Related Projects

- [golangci-lint](https://github.com/golangci/golangci-lint) - Comprehensive Go linter
- [staticcheck](https://staticcheck.io/) - Advanced Go static analysis
- [errcheck](https://github.com/kisielk/errcheck) - Error checking tool

## 📞 Support

- 📧 **Email**: support@goastanalyzer.com
- 🐛 **Issues**: [GitHub Issues](https://github.com/username/goastanalyzer/issues)
- 📖 **Documentation**: [Wiki](https://github.com/username/goastanalyzer/wiki)

---

**Keywords**: go ast analyzer, go code analysis, complexity analysis, code smells, concurrency bugs, clean architecture, domain driven design, static analysis, go linting, cyclomatic complexity, cognitive complexity, goroutine leaks, channel misuse