package productinputs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const repoRoot = "../.."

func TestCommittedPinLoadsAndVerifies(t *testing.T) {
	pin, err := Load(filepath.Join(repoRoot, "pins", "product-toolchain.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := pin.Verify(repoRoot); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if pin.Verified == nil || pin.Verified.NodeExecutableSha256 == "" || pin.Verified.NodeVersion == "" {
		t.Fatalf("verified runtime evidence missing: %+v", pin.Verified)
	}
}

func tamper(t *testing.T, mutate func(raw map[string]any)) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, "pins", "product-toolchain.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	mutate(raw)
	out, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "pin.json")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTamperedIdentitiesFailClosed(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(raw map[string]any)
		mention string
	}{
		{
			name: "helper runtime digest",
			mutate: func(raw map[string]any) {
				raw["generatedHelperRuntime"].(map[string]any)["sha256"] = strings.Repeat("0", 64)
			},
			mention: "generatedHelperRuntime digest mismatch",
		},
		{
			name: "helper runtime version",
			mutate: func(raw map[string]any) {
				raw["generatedHelperRuntime"].(map[string]any)["abiVersion"] = 1
			},
			mention: "abiVersion",
		},
		{
			name: "resolver loader digest",
			mutate: func(raw map[string]any) {
				raw["moduleResolver"].(map[string]any)["loaderSha256"] = strings.Repeat("0", 64)
			},
			mention: "loader digest mismatch",
		},
		{
			name: "strict configuration digest",
			mutate: func(raw map[string]any) {
				raw["strictTypeScriptConfig"].(map[string]any)["sha256"] = strings.Repeat("0", 64)
			},
			mention: "strictTypeScriptConfig digest mismatch",
		},
		{
			name: "runtime version",
			mutate: func(raw map[string]any) {
				raw["javascriptRuntime"].(map[string]any)["version"] = "v0.0.1"
			},
			mention: "does not match pinned",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pin, err := Load(tamper(t, c.mutate))
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			err = pin.Verify(repoRoot)
			if err == nil || !strings.Contains(err.Error(), c.mention) {
				t.Fatalf("expected verification failure mentioning %q, got: %v", c.mention, err)
			}
		})
	}
}

func TestUnknownFieldFailsClosed(t *testing.T) {
	path := tamper(t, func(raw map[string]any) {
		raw["surprise"] = true
	})
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field rejection, got: %v", err)
	}
}

func TestMissingIdentityFailsClosed(t *testing.T) {
	path := tamper(t, func(raw map[string]any) {
		raw["moduleResolver"].(map[string]any)["policy"] = ""
	})
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "moduleResolver.policy is required") {
		t.Fatalf("expected required-field rejection, got: %v", err)
	}
}
