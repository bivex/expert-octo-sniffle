package main

import (
	"os"

	"goastanalyzer/presentation/cli"
)

func main() {
	analyzer := cli.NewAnalyzerCLI()
	os.Exit(analyzer.Run(os.Args[1:]))
}
