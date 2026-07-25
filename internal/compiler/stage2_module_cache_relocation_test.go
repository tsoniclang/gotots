package compiler

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestStage2SemanticIdentitiesSurviveModuleCacheRelocation(
	t *testing.T,
) {
	const (
		dependency = "example.com/cachedep"
		version    = "v1.0.0"
	)
	proxy := t.TempDir()
	writeStage2ModuleProxy(
		t,
		proxy,
		dependency,
		version,
		map[string]string{
			"go.mod": "module " + dependency + "\n\ngo 1.26.0\n",
			"cachedep.go": `package cachedep

type Item struct {
	Value int
}

func Make(value int) Item {
	return Item{Value: value}
}
`,
		},
	)
	application := t.TempDir()
	writeCompilerFile(
		t,
		application,
		"go.mod",
		"module example.com/cache-app\n\ngo 1.26.0\n\n"+
			"require "+dependency+" "+version+"\n",
	)
	writeCompilerFile(
		t,
		application,
		"app.go",
		`package cacheapp

import "example.com/cachedep"

func Use(value int) int {
	return cachedep.Make(value).Value
}
`,
	)

	first := inspectStage2ModuleCache(
		t, application, proxy, writableModuleCache(t),
	)
	second := inspectStage2ModuleCache(
		t, application, proxy, writableModuleCache(t),
	)
	firstIDs := semanticIdentitySet(first)
	secondIDs := semanticIdentitySet(second)
	if !reflect.DeepEqual(firstIDs, secondIDs) {
		t.Fatalf(
			"semantic identities changed after module-cache relocation\nfirst=%v\nsecond=%v",
			firstIDs,
			secondIDs,
		)
	}
	for _, inspection := range []*Inspection{first, second} {
		var found bool
		for _, pkg := range inspection.Workspace().Packages() {
			if pkg.ID().ImportPath() != dependency {
				continue
			}
			found = true
			if pkg.Acquisition() != source.AcquisitionModuleCache {
				t.Fatalf(
					"dependency acquisition=%s, want module-cache",
					pkg.Acquisition(),
				)
			}
			semanticPackageByImportPath(
				t, inspection.Semantic(), dependency,
			)
		}
		if !found {
			t.Fatalf("module-cache dependency %s is absent", dependency)
		}
	}
}

func writableModuleCache(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.WalkDir(
			directory,
			func(path string, _ os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				return os.Chmod(path, 0o700)
			},
		)
	})
	return directory
}

func inspectStage2ModuleCache(
	t *testing.T,
	application string,
	proxy string,
	moduleCache string,
) *Inspection {
	t.Helper()
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir:      application,
		Patterns: []string{"."},
		Env: []string{
			"GOMODCACHE=" + moduleCache,
			"GOPROXY=file://" + filepath.ToSlash(proxy),
			"GOSUMDB=off",
			"GOFLAGS=-mod=mod",
		},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return inspection
}

func writeStage2ModuleProxy(
	t *testing.T,
	proxy string,
	module string,
	version string,
	files map[string]string,
) {
	t.Helper()
	versionDirectory := filepath.Join(
		proxy,
		filepath.FromSlash(module),
		"@v",
	)
	if err := os.MkdirAll(versionDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	writeCompilerFile(
		t,
		versionDirectory,
		version+".mod",
		files["go.mod"],
	)
	writeCompilerFile(
		t,
		versionDirectory,
		version+".info",
		`{"Version":"`+version+`","Time":"2026-01-01T00:00:00Z"}`,
	)
	writeCompilerFile(
		t, versionDirectory, "list", version+"\n",
	)
	archivePath := filepath.Join(
		versionDirectory, version+".zip",
	)
	archive, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	names := []string{"go.mod", "cachedep.go"}
	for _, name := range names {
		header := &zip.FileHeader{
			Name:   module + "@" + version + "/" + name,
			Method: zip.Deflate,
		}
		header.SetMode(0o600)
		header.SetModTime(time.Date(
			2026, time.January, 1, 0, 0, 0, 0, time.UTC,
		))
		entry, createErr := writer.CreateHeader(header)
		if createErr != nil {
			_ = writer.Close()
			_ = archive.Close()
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte(files[name])); writeErr != nil {
			_ = writer.Close()
			_ = archive.Close()
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		_ = archive.Close()
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
}
