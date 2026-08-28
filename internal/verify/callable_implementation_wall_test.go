package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCallableImplementationSafetyHasOneCurrentPath(t *testing.T) {
	root := repositoryRoot(t)
	sources := make(map[string]string)
	err := filepath.Walk(
		filepath.Join(root, "internal"),
		func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() || filepath.Ext(path) != ".go" ||
				strings.HasSuffix(path, "_test.go") ||
				strings.HasSuffix(path, "_generated.go") {
				return nil
			}
			payload, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			relative, relativeErr := filepath.Rel(root, path)
			if relativeErr != nil {
				return relativeErr
			}
			sources[filepath.ToSlash(relative)] = string(payload)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := callableImplementationSafetyWall(sources); err != nil {
		t.Fatal(err)
	}
}

func TestCallableImplementationSafetyWallMutationControls(t *testing.T) {
	clean := map[string]string{
		"internal/load/source_snapshot.go": "func sealSourceSnapshot() {}",
	}
	if err := callableImplementationSafetyWall(clean); err != nil {
		t.Fatalf("clean safety owner was rejected: %v", err)
	}
	for name, mutation := range map[string]map[string]string{
		"retired dynamic scanner": {
			"internal/load/source_snapshot.go": "func sealSourceSnapshot() {}",
			"internal/target/tsgo/leak.go":     "func SourceForbiddenDynamicTypes() {}",
		},
		"retired command digest": {
			"internal/load/source_snapshot.go": "func sealSourceSnapshot() {}",
			"internal/command/leak.go":         "func programDigest() {}",
		},
		"duplicate snapshot owner": {
			"internal/load/source_snapshot.go": "func sealSourceSnapshot() {}",
			"internal/load/leak.go":            "func sealSourceSnapshot() {}",
		},
		"retired policy file": {
			"internal/load/source_snapshot.go":               "func sealSourceSnapshot() {}",
			"internal/target/tsgo/project_forbidden_type.go": "package tsgo",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := callableImplementationSafetyWall(mutation); err == nil {
				t.Fatal("safety-path mutation passed")
			}
		})
	}
}

func callableImplementationSafetyWall(sources map[string]string) error {
	owners := 0
	for path, source := range sources {
		if filepath.Base(path) == "project_forbidden_type.go" {
			return fmt.Errorf("%s: retired callable source-policy file survives", path)
		}
		for _, retired := range []string{
			"SourceForbiddenDynamicTypes",
			"func programDigest(",
		} {
			if strings.Contains(source, retired) {
				return fmt.Errorf("%s: retired callable safety route %q survives", path, retired)
			}
		}
		owners += strings.Count(source, "func sealSourceSnapshot(")
	}
	if owners != 1 {
		return fmt.Errorf("selected-source snapshot has %d owners, want exactly 1", owners)
	}
	return nil
}
