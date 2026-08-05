package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderOwnsStableScalarImports(t *testing.T) {
	root := repositoryRoot(t)
	provider := filepath.Join(root, "gostdlib", "src")
	const productScalarImport = "@gotots/runtime/scalars.js"
	const providerScalarImport = "@gotots/gostdlib/internal/scalars.js"
	imports := 0
	err := filepath.Walk(provider, func(
		path string,
		info os.FileInfo,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(path) != ".ts" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(content)
		if strings.Contains(text, productScalarImport) {
			t.Errorf("%s imports product-selected scalars", path)
		}
		imports += strings.Count(text, providerScalarImport)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if imports == 0 {
		t.Fatal("provider scalar import wall is vacuous")
	}
}
