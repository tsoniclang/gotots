// Package fingerprint is the permanent reproducible fingerprint
// producer: every input and output class of a baseline carries its
// sorted path/hash manifest, every class exists even when empty, every
// file belongs to exactly one class, and semantic identities are
// separated from machine-environment evidence. A future run reproduces
// the report through this component alone; there is no one-off
// generation path.
package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SchemaVersion = 1

// FileHash is one file's normalized path and content digest.
type FileHash struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
}

// Class is one named fingerprint class: its complete sorted manifest
// and the digest over that manifest. A class with no files is present
// with an empty manifest — absence and emptiness are different facts.
type Class struct {
	Name   string     `json:"name"`
	Files  []FileHash `json:"files"`
	Sha256 string     `json:"sha256"`
}

// Report is the versioned fingerprint evidence.
type Report struct {
	SchemaVersion int `json:"schemaVersion"`
	// Classes hold every declared class in declaration order.
	Classes []Class `json:"classes"`
	// Semantic identities participate in reproducibility comparisons:
	// pins, schema/profile/spec digests, tool versions.
	Semantic map[string]string `json:"semantic"`
	// Environment records machine evidence (paths, hostnames, local
	// executable digests) that must NEVER enter semantic identity.
	Environment map[string]string `json:"environment,omitempty"`
}

// Classifier assigns one file to exactly one class. Returning ok=false
// leaves the file unattributed, which fails the build.
type Classifier func(path string) (class string, ok bool)

// Build walks root, classifies every regular file, and produces the
// report. declaredClasses fixes the class universe: a classifier
// answer outside it, a duplicate path, or an unattributed file fails
// closed.
func Build(root string, declaredClasses []string, classify Classifier, semantic, environment map[string]string) (*Report, error) {
	declared := map[string]bool{}
	for _, name := range declaredClasses {
		if declared[name] {
			return nil, fmt.Errorf("class %s declared twice", name)
		}
		declared[name] = true
	}
	manifests := map[string][]FileHash{}
	seen := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if seen[relative] {
			return fmt.Errorf("duplicate path %s", relative)
		}
		seen[relative] = true
		class, ok := classify(relative)
		if !ok {
			return fmt.Errorf("unattributed file %s: every file belongs to exactly one class", relative)
		}
		if !declared[class] {
			return fmt.Errorf("file %s classified into undeclared class %s", relative, class)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(content)
		manifests[class] = append(manifests[class], FileHash{Path: relative, Sha256: hex.EncodeToString(digest[:])})
		return nil
	})
	if err != nil {
		return nil, err
	}
	report := &Report{SchemaVersion: SchemaVersion, Semantic: semantic, Environment: environment}
	for _, name := range declaredClasses {
		files := manifests[name]
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
		report.Classes = append(report.Classes, Class{Name: name, Files: files, Sha256: manifestDigest(files)})
	}
	return report, nil
}

// manifestDigest digests the sorted path/hash manifest; the empty
// manifest has the well-known empty digest.
func manifestDigest(files []FileHash) string {
	h := sha256.New()
	for _, file := range files {
		h.Write([]byte(file.Path))
		h.Write([]byte{0})
		h.Write([]byte(file.Sha256))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Diff joins two reports class-by-class and returns every per-file
// difference. Classes must match by name; a class present in one
// report only is itself a difference.
type Difference struct {
	Class string `json:"class"`
	Path  string `json:"path,omitempty"`
	Kind  string `json:"kind"` // class-only-left | class-only-right | only-left | only-right | hash-changed
}

func Diff(left, right *Report) []Difference {
	var out []Difference
	leftClasses := map[string]Class{}
	for _, c := range left.Classes {
		leftClasses[c.Name] = c
	}
	rightClasses := map[string]Class{}
	for _, c := range right.Classes {
		rightClasses[c.Name] = c
	}
	names := map[string]bool{}
	for name := range leftClasses {
		names[name] = true
	}
	for name := range rightClasses {
		names[name] = true
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		l, hasLeft := leftClasses[name]
		r, hasRight := rightClasses[name]
		switch {
		case !hasRight:
			out = append(out, Difference{Class: name, Kind: "class-only-left"})
			continue
		case !hasLeft:
			out = append(out, Difference{Class: name, Kind: "class-only-right"})
			continue
		case l.Sha256 == r.Sha256:
			continue
		}
		lFiles := map[string]string{}
		for _, f := range l.Files {
			lFiles[f.Path] = f.Sha256
		}
		rFiles := map[string]string{}
		for _, f := range r.Files {
			rFiles[f.Path] = f.Sha256
		}
		for _, f := range l.Files {
			hash, has := rFiles[f.Path]
			switch {
			case !has:
				out = append(out, Difference{Class: name, Path: f.Path, Kind: "only-left"})
			case hash != f.Sha256:
				out = append(out, Difference{Class: name, Path: f.Path, Kind: "hash-changed"})
			}
		}
		for _, f := range r.Files {
			if _, has := lFiles[f.Path]; !has {
				out = append(out, Difference{Class: name, Path: f.Path, Kind: "only-right"})
			}
		}
	}
	return out
}

// PrefixClassifier builds a Classifier from ordered (prefix, class)
// rules with exact segment-boundary matching; the first matching rule
// wins and rules must not shadow each other ambiguously (validated by
// construction order being part of the declared contract).
func PrefixClassifier(rules []PrefixRule) Classifier {
	return func(path string) (string, bool) {
		for _, rule := range rules {
			if rule.Prefix == "" || path == rule.Prefix || strings.HasPrefix(path, rule.Prefix+"/") {
				return rule.Class, true
			}
		}
		return "", false
	}
}

// PrefixRule maps one path prefix (a directory subtree, "" for
// everything) to a class.
type PrefixRule struct {
	Prefix string `json:"prefix"`
	Class  string `json:"class"`
}
