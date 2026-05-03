package main

import (
	"github.com/gostaticanalysis/nilerr"
	"github.com/kisielk/errcheck/errcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"gophkeeper/server/cmd/staticlint/hardcodechecker"
	"honnef.co/go/tools/staticcheck"
	"strings"
)

// Multichecker для GophKeeper
func main() {
	analyzers := []*analysis.Analyzer{
		assign.Analyzer,
		shadow.Analyzer,
		printf.Analyzer,
		nilness.Analyzer,
		lostcancel.Analyzer,
		copylock.Analyzer,
		loopclosure.Analyzer,
	}

	// SAxxxx из staticcheck
	for _, sa := range staticcheck.Analyzers {
		if strings.HasPrefix(sa.Analyzer.Name, "SA") {
			analyzers = append(analyzers, sa.Analyzer)
		}
	}

	// ST1000
	for _, sa := range staticcheck.Analyzers {
		if sa.Analyzer.Name == "ST1000" {
			analyzers = append(analyzers, sa.Analyzer)
			break
		}
	}

	// Кастомные + публичные
	analyzers = append(analyzers, hardcodechecker.Analyzer)
	analyzers = append(analyzers, errcheck.Analyzer)
	analyzers = append(analyzers, nilerr.Analyzer)

	multichecker.Main(analyzers...)
}
