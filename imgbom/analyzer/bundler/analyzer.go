package bundler

import (
	"github.com/anchore/imgbom/imgbom/analyzer/common"
	"github.com/anchore/imgbom/imgbom/pkg"
	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/tree"
)

type Analyzer struct {
	analyzer common.GenericAnalyzer
}

func NewAnalyzer() *Analyzer {
	globParserDispatch := map[string]common.ParserFn{
		"*/Gemfile.lock": parseGemfileLockEntries,
	}

	return &Analyzer{
		analyzer: common.NewGenericAnalyzer(nil, globParserDispatch),
	}
}

func (a *Analyzer) Name() string {
	return "bundler-analyzer"
}

func (a *Analyzer) SelectFiles(trees []tree.FileTreeReader) []file.Reference {
	return a.analyzer.SelectFiles(trees)
}

func (a *Analyzer) Analyze(contents map[file.Reference]string) ([]pkg.Package, error) {
	return a.analyzer.Analyze(contents, a.Name())
}
