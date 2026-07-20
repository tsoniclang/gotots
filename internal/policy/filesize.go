// Package policy implements repository-wide normative gates from the
// gotots specification (docs/spec/). Gates run as ordinary tests so every
// CI and local run enforces them; they scan the whole tree, never a
// reviewer-named subset.
package policy

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MaxSourceFileLines is the normative limit from
// docs/spec/governance-upgrades.md: no hand-maintained
// implementation or test source file may exceed this many physical lines.
const MaxSourceFileLines = 600

// skippedDirs are trees that contain no hand-maintained gotots source:
// VCS state, local scratch/evidence areas, dependency caches, and test
// fixture data (fixture content is input, not implementation).
var skippedDirs = map[string]bool{
	".git":         true,
	".analysis":    true,
	".claude":      true,
	".temp":        true,
	".tests":       true,
	"node_modules": true,
	"testdata":     true,
	// The calibration corpus (hand ports, extracted Go spans, derived
	// manifests) is measurement input/evidence sized by its source
	// spans; its implementation lives in internal/calibration.
	"calibration": true,
}

// sourceExtensions are Go files consumed by Go-specific gates such as gofmt.
var sourceExtensions = map[string]bool{
	".go": true,
}

// maintainedExtensions are repository source/specification classes governed
// by the physical-line limit. Product-generated trees belong under separately
// manifest-owned output roots, not the maintained repository tree scanned here.
var maintainedExtensions = map[string]bool{
	".cjs": true,
	".go":  true,
	".js":  true,
	".md":  true,
	".mjs": true,
	".ts":  true,
	".tsx": true,
}

// Violation is one file exceeding the limit.
type Violation struct {
	Path  string
	Lines int
}

func (v Violation) String() string {
	return fmt.Sprintf("GOTOTS_FILE_TOO_LARGE:\n%s has %d lines; maximum is %d.\nSplit it by semantic responsibility.",
		v.Path, v.Lines, MaxSourceFileLines)
}

// SourceFiles walks root and returns every hand-maintained source file —
// the single tree definition every repository-wide gate shares — as
// root-relative slash paths, sorted.
func SourceFiles(root string) ([]string, error) {
	return walkFiles(root, sourceExtensions)
}

// MaintainedFiles returns every implementation, test, and normative Markdown
// file governed by the physical-line policy.
func MaintainedFiles(root string) ([]string, error) {
	return walkFiles(root, maintainedExtensions)
}

func walkFiles(root string, extensions map[string]bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !extensions[filepath.Ext(entry.Name())] {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relative = path
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// CheckTree returns every source file exceeding the limit, sorted by path.
func CheckTree(root string) ([]Violation, error) {
	files, err := MaintainedFiles(root)
	if err != nil {
		return nil, err
	}
	var violations []Violation
	for _, relative := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return nil, err
		}
		lines := countPhysicalLines(data)
		if lines > MaxSourceFileLines {
			violations = append(violations, Violation{Path: relative, Lines: lines})
		}
	}
	return violations, nil
}

// countPhysicalLines counts lines the way an editor displays them: a final
// line without a trailing newline still counts.
func countPhysicalLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	lines := strings.Count(string(data), "\n")
	if data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}
