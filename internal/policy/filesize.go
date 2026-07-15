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
// docs/spec/file-size-and-decomposition.md: no hand-maintained
// implementation or test source file may exceed this many physical lines.
const MaxSourceFileLines = 600

// skippedDirs are trees that contain no hand-maintained gotots source:
// VCS state, local scratch/evidence areas, dependency caches, and test
// fixture data (fixture content is input, not implementation).
var skippedDirs = map[string]bool{
	".git":         true,
	".analysis":    true,
	".temp":        true,
	".tests":       true,
	"node_modules": true,
	"testdata":     true,
}

// sourceExtensions are the hand-maintained source classes the limit
// applies to. Extend this list when the repository gains new
// implementation languages; do not exempt individual files.
var sourceExtensions = map[string]bool{
	".go": true,
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
		if !sourceExtensions[filepath.Ext(entry.Name())] {
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
	files, err := SourceFiles(root)
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
