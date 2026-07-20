package inventory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
)

// reconcileUniverse proves that every tracked file has exactly one
// disposition. Outside-universe roots are filtered first: their files
// receive no per-file records — only per-category counts and one
// aggregate digest attest what the filter removed. Build inputs claimed
// by inventoried packages are in-package; module metadata, profile
// tooling roots, testdata, and the toolchain's documented
// underscore/dot exclusions classify the rest. A tracked .go file with
// no disposition fails the census; other files fall into the explicit
// non-build-file class, enumerated by extension.
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
	universe.OutsideUniverse = map[string]int{}
	classMembers := map[string][]string{}

	// A file classifies through its directory's winning schema-2 rule;
	// files whose directory matches no rule fall through to the
	// non-package handling below (repo-root metadata, docs, ...).
	fileRule := func(filePath string) (profile.PackageDisposition, string, bool) {
		rule, err := prof.SourceUniverse.Classify(path.Dir(filePath))
		if err != nil {
			return "", "", false
		}
		return rule.Disposition, rule.Category, true
	}
	outsideCategory := func(filePath string) (string, bool) {
		if disposition, category, ok := fileRule(filePath); ok && disposition == profile.DispositionOutside {
			return category, true
		}
		return "", false
	}

	var unclassifiedGo []string
	for _, filePath := range tracked {
		if category, outside := outsideCategory(filePath); outside {
			universe.OutsideUniverseFiles++
			universe.OutsideUniverse[category]++
			classMembers["outside-universe"] = append(classMembers["outside-universe"], filePath)
			continue
		}
		if inPackage[filePath] {
			universe.InPackages++
			classMembers["in-package"] = append(classMembers["in-package"], filePath)
			continue
		}
		base := path.Base(filePath)
		isGo := strings.HasSuffix(filePath, ".go")

		var class string
		switch {
		case base == "go.mod" || base == "go.sum" || base == "go.work" || base == "go.work.sum":
			universe.ModuleMetadata = append(universe.ModuleMetadata, filePath)
			classMembers["module-metadata"] = append(classMembers["module-metadata"], filePath)
			continue
		case func() bool {
			disposition, _, ok := fileRule(filePath)
			return ok && disposition == profile.DispositionTooling
		}():
			class = "tooling"
		case func() bool {
			disposition, _, ok := fileRule(filePath)
			return ok && disposition == profile.DispositionPolicyExcluded
		}():
			class = "product-policy-excluded"
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
				classMembers["non-build"] = append(classMembers["non-build"], filePath)
			}
			continue
		}
		classMembers[class] = append(classMembers[class], filePath)
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

	universe.ClassDigests = map[string]string{}
	for class, members := range classMembers {
		sort.Strings(members)
		hash := sha256.New()
		for _, member := range members {
			hash.Write([]byte(member))
			hash.Write([]byte{0})
		}
		universe.ClassDigests[class] = hex.EncodeToString(hash.Sum(nil))
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
