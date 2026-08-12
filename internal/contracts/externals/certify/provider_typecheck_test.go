package certify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalProviderCertificationRejectsTypeDiagnostics(t *testing.T) {
	providerRoot := t.TempDir()
	configPath := filepath.Join(providerRoot, "tsconfig.json")
	writeTypecheckFixture(t, configPath, `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "noEmit": true,
    "types": []
  },
  "include": ["source.ts"]
}
`)
	sourcePath := filepath.Join(providerRoot, "source.ts")
	writeTypecheckFixture(t, sourcePath, "export const value: string = 1;\n")

	config := testConfig(t)
	err := verifyProviderTypecheck(resolvedConfig{
		repositoryRoot: config.RepositoryRoot,
		providerRoot:   providerRoot,
		tsConfigPath:   configPath,
		tsgoTool:       config.TSGoTool,
	})
	if err == nil || !strings.Contains(err.Error(), "TS2322") {
		t.Fatalf("typecheck error = %v, want TS2322", err)
	}

	writeTypecheckFixture(t, sourcePath, "export const value: string = \"valid\";\n")
	if err := verifyProviderTypecheck(resolvedConfig{
		repositoryRoot: config.RepositoryRoot,
		providerRoot:   providerRoot,
		tsConfigPath:   configPath,
		tsgoTool:       config.TSGoTool,
	}); err != nil {
		t.Fatalf("valid provider failed certification: %v", err)
	}
}

func writeTypecheckFixture(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
