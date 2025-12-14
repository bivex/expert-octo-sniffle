# Clean Architecture Implementation for Go AST Analyzer

## Overview

This project has been refactored from a monolithic structure to a clean, layered architecture following Domain-Driven Design (DDD) and Clean Architecture principles. The system analyzes Go source code for architectural smells and complexity issues based on empirical research.

## Architectural Principles Applied

### ✅ DDD & Clean Architecture
- **Domain Layer**: Core business logic with entities, value objects, and domain services
- **Application Layer**: Use cases orchestrating domain operations
- **Infrastructure Layer**: External dependencies and adapters
- **Presentation Layer**: CLI interface with clean separation

### ✅ SOLID Principles
- **SRP**: Each module has one clear responsibility
- **OCP**: Complexity calculators and smell detectors are extensible
- **LSP**: All implementations properly substitute their interfaces
- **ISP**: Focused interfaces (FileParser, IDGenerator, etc.)
- **DIP**: High-level modules depend on abstractions, not details

### ✅ Clean Code Practices
- **Explicit interfaces** for all external dependencies
- **Dependency injection** via constructor injection
- **No service locators** or global singletons
- **Configuration externalized** from code
- **Pure functions** where possible
- **Clear naming** and focused responsibilities

## Directory Structure

```
goastanalyzer/
├── domain/                    # Domain Layer
│   ├── entities/             # Domain entities (AnalysisFinding)
│   ├── valueobjects/         # Value objects (ComplexityScore, SourceLocation, Configuration)
│   ├── aggregates/           # Aggregates (AnalysisResult)
│   └── services/             # Domain services (ComplexityCalculator, SmellDetector)
├── application/              # Application Layer
│   └── usecases/             # Use cases (AnalyzeCodeUseCase)
├── infrastructure/           # Infrastructure Layer
│   ├── adapters/             # Adapters (GoFileParser, UUIDGenerator)
│   └── config/               # Configuration management
├── presentation/             # Presentation Layer
│   └── cli/                  # CLI interface
├── internal/                 # Internal packages (future use)
├── main.go                   # Application entry point
└── test files...             # Test files
```

## Layer Responsibilities

### Domain Layer (`domain/`)

**Purpose**: Contains the core business logic and rules that are independent of any specific application or infrastructure concerns.

#### Entities (`entities/`)
- **AnalysisFinding**: Represents individual issues found during analysis
- Immutable domain objects with business identity
- Enforce business rules and invariants

#### Value Objects (`valueobjects/`)
- **ComplexityScore**: Cyclomatic and cognitive complexity metrics
- **SourceLocation**: File position information
- **AnalysisConfiguration**: Analysis parameters and thresholds
- Immutable, identified by their values, no business identity

#### Aggregates (`aggregates/`)
- **AnalysisResult**: Root aggregate containing findings and metadata
- Defines transactional boundaries
- Ensures aggregate consistency

#### Domain Services (`services/`)
- **ComplexityCalculator**: Calculates complexity metrics from AST
- **SmellDetector**: Identifies architectural smells
- Stateless services operating on domain objects

### Application Layer (`application/`)

**Purpose**: Orchestrates domain objects to fulfill business use cases. Contains application-specific logic but no business rules.

#### Use Cases (`usecases/`)
- **AnalyzeCodeUseCase**: Orchestrates the complete code analysis workflow
- Coordinates domain services and infrastructure
- Returns application-specific response objects
- Handles cross-cutting concerns (logging, error handling)

### Infrastructure Layer (`infrastructure/`)

**Purpose**: Provides implementations for external concerns (file system, network, etc.) and adapts them to domain interfaces.

#### Adapters (`adapters/`)
- **GoFileParser**: Wraps `go/parser` for domain interface
- **UUIDGenerator**: Provides unique ID generation
- Implements domain-defined interfaces
- Handles technical concerns (error translation, resource management)

#### Configuration (`config/`)
- **Config**: Loads settings from environment variables
- Externalizes configuration from code
- Provides defaults and validation

### Presentation Layer (`presentation/`)

**Purpose**: Handles user interaction and formats output for different interfaces.

#### CLI (`cli/`)
- **AnalyzerCLI**: Command-line interface implementation
- Parses command-line arguments
- Formats results for console output
- Handles different output modes (text, JSON, table)

## Dependency Direction

```
Presentation → Application → Domain ← Infrastructure
     ↑                                       ↑
     └─────────────┬─────────────────────────┘
                   │
            Infrastructure adapters
            implement domain interfaces
```

- **Inner layers** (Domain) have no dependencies on outer layers
- **Outer layers** depend on inner layers through interfaces
- **Infrastructure** implements domain interfaces (Dependency Inversion)
- **Presentation** depends on Application, not directly on Domain

## Key Design Decisions

### 1. Ports & Adapters Pattern
- Domain defines interfaces (ports) for external dependencies
- Infrastructure provides implementations (adapters)
- Enables easy testing and infrastructure swapping

### 2. Value Objects for Immutability
- Configuration and metrics are immutable value objects
- Prevents accidental modification
- Thread-safe by design

### 3. Aggregate Boundaries
- AnalysisResult is the aggregate root
- Encapsulates all findings for an analysis session
- Maintains aggregate invariants

### 4. Use Case Pattern
- Each business operation is a use case
- Clear input/output contracts
- Easy to test and extend

### 5. Configuration Management
- Environment-based configuration
- No hardcoded values
- Easy deployment across environments

## Quality Attributes Achieved

### Maintainability
- **Single Responsibility**: Each class/module has one reason to change
- **Open/Closed**: New complexity calculators/smell detectors can be added without modifying existing code
- **Dependency Inversion**: High-level modules don't depend on low-level details

### Testability
- **Dependency Injection**: All dependencies are injected, easy to mock
- **Pure Functions**: Domain logic is side-effect free where possible
- **Interface Segregation**: Small, focused interfaces

### Extensibility
- **Plugin Architecture**: New analysis types can be added via domain services
- **Configuration-Driven**: Behavior can be changed without code changes
- **Clean Interfaces**: Easy to add new implementations

### Performance
- **Efficient AST Traversal**: Uses `ast.Inspect` for optimal performance
- **Lazy Evaluation**: Complexity calculated only when needed
- **Minimal Allocations**: Value objects reused where appropriate

## Usage Examples

### Basic Analysis
```bash
./goastanalyzer sample.go
```

### JSON Output
```bash
./goastanalyzer -output json sample.go
```

### Custom Configuration
```bash
GOAST_MAX_CYCLOMATIC=20 ./goastanalyzer sample.go
```

## Testing Strategy

### Unit Tests
- Domain objects and services (100% coverage target)
- Pure functions and business logic
- Mock external dependencies

### Integration Tests
- Use case execution with real infrastructure
- End-to-end CLI testing
- Configuration loading

### Architecture Validation
- Dependency direction checks
- Interface compliance verification
- Performance benchmarks

## Future Extensions

The clean architecture enables easy extension:

1. **New Analysis Types**: Add domain services for security analysis, performance metrics, etc.
2. **Different Outputs**: Web API, GUI, IDE plugins
3. **Alternative Parsers**: Tree-sitter, ANTLR backends
4. **Distributed Analysis**: Microservices for large codebases
5. **Historical Tracking**: Database storage for analysis trends

## Empirical Validation

The architecture successfully implements the research findings:

- **Cyclomatic Complexity**: McCabe's algorithm with Go-adjusted thresholds (15 vs 10)
- **Cognitive Complexity**: SonarSource algorithm penalizing nesting
- **Architectural Smells**: Detection of God objects, interface pollution, goroutine leaks
- **Performance**: Sub-millisecond analysis for typical Go files

This clean architecture provides a solid foundation for maintaining and extending the Go AST analyzer while keeping the codebase maintainable, testable, and extensible.
