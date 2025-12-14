package config

import (
	"os"
	"strconv"

	"goastanalyzer/domain/valueobjects"
)

// Config holds application configuration
type Config struct {
	Analysis valueobjects.AnalysisConfiguration
}

// LoadConfig loads configuration from environment variables
func LoadConfig() Config {
	return Config{
		Analysis: loadAnalysisConfig(),
	}
}

// loadAnalysisConfig loads analysis configuration from environment
func loadAnalysisConfig() valueobjects.AnalysisConfiguration {
	config := valueobjects.DefaultAnalysisConfiguration()

	// Override with environment variables if present
	if maxCyclo := getEnvInt("GOAST_MAX_CYCLOMATIC", 0); maxCyclo > 0 {
		config = config.WithMaxCyclomaticComplexity(maxCyclo)
	}

	if maxCog := getEnvInt("GOAST_MAX_COGNITIVE", 0); maxCog > 0 {
		config = config.WithMaxCognitiveComplexity(maxCog)
	}

	if maxLen := getEnvInt("GOAST_MAX_FUNCTION_LENGTH", 0); maxLen > 0 {
		config = config.WithMaxFunctionLength(maxLen)
	}

	if smellDetect := getEnvBool("GOAST_ENABLE_SMELL_DETECTION", true); !smellDetect {
		config = config.WithSmellDetection(smellDetect)
	}

	return config
}

// getEnvInt gets an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
