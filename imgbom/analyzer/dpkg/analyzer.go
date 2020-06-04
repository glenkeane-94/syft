package dpkg

import (
	"io"
	"strings"

	"github.com/anchore/imgbom/imgbom/pkg"
	"github.com/anchore/imgbom/internal/log"
	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/tree"
)

var parserDispatch = map[string]parserFn{
	"/var/lib/dpkg/status": ParseDpkgStatusEntries,
}

type parserFn func(io.Reader) ([]pkg.DpkgMetadata, error)

type Analyzer struct {
	selectedFiles []file.Reference
	parsers       map[file.Reference]parserFn
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		selectedFiles: make([]file.Reference, 0),
		parsers:       make(map[file.Reference]parserFn),
	}
}

func (a *Analyzer) Name() string {
	return "dpkg-analyzer"
}

func (a *Analyzer) register(f file.Reference, parser parserFn) {
	a.selectedFiles = append(a.selectedFiles, f)
	a.parsers[f] = parser
}

func (a *Analyzer) clear() {
	a.selectedFiles = make([]file.Reference, 0)
	a.parsers = make(map[file.Reference]parserFn)
}

func (a *Analyzer) SelectFiles(trees []tree.FileTreeReader) []file.Reference {
	for _, tree := range trees {
		for exactPath, parser := range parserDispatch {
			match := tree.File(file.Path(exactPath))
			if match != nil {
				a.register(*match, parser)
			}
		}
	}

	return a.selectedFiles
}

func (a *Analyzer) Analyze(contents map[file.Reference]string) ([]pkg.Package, error) {
	defer a.clear()

	packages := make([]pkg.Package, 0)

	for _, reference := range a.selectedFiles {
		content, ok := contents[reference]
		if !ok {
			log.Errorf("analyzer '%s' file content missing: %+v", a.Name(), reference)
			continue
		}

		entries, err := ParseDpkgStatusEntries(strings.NewReader(content))
		if err != nil {
			log.Errorf("analyzer failed to parse entries (reference=%+v): %w", reference, err)
			continue
		}

		for _, entry := range entries {
			packages = append(packages, pkg.Package{
				Name:     entry.Package,
				Version:  entry.Version,
				Type:     pkg.DebPkg,
				FoundBy:  a.Name(),
				Source:   []file.Reference{reference},
				Metadata: entry,
			})
		}
	}

	return packages, nil
}
