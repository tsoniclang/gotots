package inventory

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
)

// reconcileUniverse proves that every tracked file has exactly one
// disposition. Build inputs claimed by inventoried packages are
// in-package; module metadata, profile tooling roots, testdata, and the
// toolchain's documented underscore/dot exclusions classify the rest. A
// tracked .go file with no disposition fails the census; other files fall
// into the explicit non-build-file class, enumerated by extension.
func reconcileUniverse(prof *profile.Profile, tree *pinning.Tree, modulePackages map[string]*ModulePackage, universe *Universe) error {
	inPackage := map[string]bool{}
	for _, pkg := range modulePackages {
		for _, file := range pkg.buildInputFiles() {
			inPackage[file] = true
		}
	}

	tracked := tree.Files()
	universe.TrackedFiles = len(tracked)
	universe.NonBuildByExtension = map[string]int{}

	underAny := func(filePath string, roots []string) bool {
		for _, root := range roots {
			if filePath == root || strings.HasPrefix(filePath, root+"/") {
				return true
			}
		}
		return false
	}

	var unclassifiedGo []string
	for _, filePath := range tracked {
		if inPackage[filePath] {
			universe.InPackages++
			continue
		}
		base := path.Base(filePath)
		isGo := strings.HasSuffix(filePath, ".go")

		var class string
		switch {
		case base == "go.mod" || base == "go.sum" || base == "go.work" || base == "go.work.sum":
			universe.ModuleMetadata = append(universe.ModuleMetadata, filePath)
			continue
		case underAny(filePath, prof.ToolingRoots):
			class = "tooling"
		case hasSegment(filePath, "testdata"):
			// The go toolchain's documented rule: testdata directories are
			// never part of any package.
			class = "testdata"
		case hasExcludedSegment(filePath):
			// The go toolchain's documented rule: path elements beginning
			// with "_" or "." are ignored by package discovery.
			class = "toolchain-excluded"
		default:
			if isGo {
				unclassifiedGo = append(unclassifiedGo, filePath)
			} else {
				universe.NonBuildFiles++
				extension := path.Ext(base)
				if extension == "" {
					extension = "(none)"
				}
				universe.NonBuildByExtension[extension]++
			}
			continue
		}
		if isGo {
			universe.GoOutsidePackages = append(universe.GoOutsidePackages, TrackedFile{Path: filePath, Class: class})
		} else {
			universe.NonBuildFiles++
			universe.NonBuildByExtension[class+":"+path.Ext(base)]++
		}
	}
	if len(unclassifiedGo) > 0 {
		return fmt.Errorf("universe reconciliation fails closed: %d tracked .go files have no disposition:\n%s",
			len(unclassifiedGo), strings.Join(unclassifiedGo, "\n"))
	}

	for submodulePath, revision := range tree.Submodules {
		universe.Submodules = append(universe.Submodules, SubmoduleRecord{Path: submodulePath, Revision: revision})
	}
	sort.Slice(universe.Submodules, func(i, j int) bool {
		return universe.Submodules[i].Path < universe.Submodules[j].Path
	})
	sort.Strings(universe.ModuleMetadata)
	return nil
}

func hasSegment(filePath, segment string) bool {
	for _, part := range strings.Split(filePath, "/") {
		if part == segment {
			return true
		}
	}
	return false
}

func hasExcludedSegment(filePath string) bool {
	// The go toolchain ignores directories and files whose names begin
	// with "_" or "." during package discovery.
	for _, part := range strings.Split(filePath, "/") {
		if strings.HasPrefix(part, "_") || strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}
