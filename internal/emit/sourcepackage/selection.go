package sourcepackage

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

type Paths struct {
	assembly     string
	owned        map[string]struct{}
	bySourceFile map[string]string
}

func ResolvePaths(sourcePackage *load.Package) (Paths, error) {
	if sourcePackage == nil || sourcePackage.Kind() != load.PackageSource {
		return Paths{}, fmt.Errorf("source package is absent")
	}
	assemblyPath, err := output.PackageAssemblyPath(sourcePackage)
	if err != nil {
		return Paths{}, err
	}
	statePath, err := output.PackageStatePath(sourcePackage)
	if err != nil {
		return Paths{}, err
	}
	result := Paths{
		assembly:     assemblyPath,
		bySourceFile: make(map[string]string),
		owned: map[string]struct{}{
			assemblyPath: {},
			statePath:    {},
		},
	}
	for _, sourceFile := range sourcePackage.Files() {
		outputPath, err := output.SourcePath(sourcePackage, sourceFile)
		if err != nil {
			return Paths{}, err
		}
		result.owned[outputPath] = struct{}{}
		name := filepath.Base(sourceFile.Path())
		if _, duplicate := result.bySourceFile[name]; duplicate {
			return Paths{}, fmt.Errorf("Go source filename %q is duplicated", name)
		}
		result.bySourceFile[name] = outputPath
	}
	return result, nil
}

func (p Paths) AssemblyPath() string {
	return p.assembly
}

func (p Paths) Owns(outputPath string) bool {
	_, ok := p.owned[outputPath]
	return ok
}

func (p Paths) SourcePath(goFile string) (string, bool) {
	selected, ok := p.bySourceFile[goFile]
	return selected, ok
}

func (p Paths) SourcePaths() []string {
	result := make([]string, 0, len(p.bySourceFile))
	for _, outputPath := range p.bySourceFile {
		result = append(result, outputPath)
	}
	sort.Strings(result)
	return result
}

func (p Paths) OwnedPaths() []string {
	result := make([]string, 0, len(p.owned))
	for outputPath := range p.owned {
		result = append(result, outputPath)
	}
	sort.Strings(result)
	return result
}
