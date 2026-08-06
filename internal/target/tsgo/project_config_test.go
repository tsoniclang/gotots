package tsgo

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestStrictProjectConfigOwnsResolutionAndStrictness(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "work", "generated")
	payload, err := EncodeStrictProjectConfig(root, project)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		CompilerOptions struct {
			Paths                      map[string][]string `json:"paths"`
			Strict                     bool                `json:"strict"`
			ExactOptionalPropertyTypes bool                `json:"exactOptionalPropertyTypes"`
			NoUncheckedIndexedAccess   bool                `json:"noUncheckedIndexedAccess"`
			NoImplicitOverride         bool                `json:"noImplicitOverride"`
			NoFallthroughCasesInSwitch bool                `json:"noFallthroughCasesInSwitch"`
			ForceConsistentCasing      bool                `json:"forceConsistentCasingInFileNames"`
			SkipLibCheck               bool                `json:"skipLibCheck"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	options := document.CompilerOptions
	if !options.Strict || !options.ExactOptionalPropertyTypes ||
		!options.NoUncheckedIndexedAccess || !options.NoImplicitOverride ||
		!options.NoFallthroughCasesInSwitch || !options.ForceConsistentCasing ||
		options.SkipLibCheck {
		t.Fatalf("strict project options are incomplete: %#v", options)
	}
	want := map[string][]string{
		"@gotots/runtime/*.js":   {"./runtime/*.ts"},
		"@gotots/gostdlib/*.js":  {"../../gostdlib/dist/src/*.d.ts"},
		"@gotots/externals/*.js": {"../../externals/dist/src/*.d.ts"},
	}
	if !equalProjectPaths(options.Paths, want) {
		t.Fatalf("project paths = %#v, want %#v", options.Paths, want)
	}
}

func equalProjectPaths(left map[string][]string, right map[string][]string) bool {
	if len(left) != len(right) {
		return false
	}
	for name, want := range right {
		got, ok := left[name]
		if !ok || len(got) != len(want) {
			return false
		}
		for index := range want {
			if got[index] != want[index] {
				return false
			}
		}
	}
	return true
}
