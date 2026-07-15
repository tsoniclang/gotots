package profile

import "testing"

func testProfile() *Profile {
	return &Profile{
		GoModule:      "example.com/mod",
		OwnedRoots:    []string{"internal/execute", "internal/scanner", "internal/vfs"},
		TestOnlyRoots: []string{"internal/execute/tsctests", "internal/testutil", "internal/vfs/vfstest"},
		HardExcludedRoots: map[string][]string{
			"lsp":            {"internal/lsp"},
			"editor-service": {"internal/ls", "internal/testutil/lsptestutil"},
		},
	}
}

func TestClassify(t *testing.T) {
	p := testProfile()
	cases := []struct {
		pkg      string
		class    PackageClass
		category string
	}{
		{"example.com/mod/internal/scanner", ClassOwned, ""},
		{"example.com/mod/internal/scanner/sub", ClassOwned, ""},
		{"example.com/mod/internal/vfs", ClassOwned, ""},
		// Test-only carve-out inside an owned root wins over the owned root.
		{"example.com/mod/internal/vfs/vfstest", ClassTestOnly, ""},
		{"example.com/mod/internal/execute/tsctests", ClassTestOnly, ""},
		{"example.com/mod/internal/testutil", ClassTestOnly, ""},
		// Hard-excluded carve-out inside a test-only root wins over test-only.
		{"example.com/mod/internal/testutil/lsptestutil", ClassHardExcluded, "editor-service"},
		{"example.com/mod/internal/lsp", ClassHardExcluded, "lsp"},
		{"example.com/mod/internal/lsp/lsproto", ClassHardExcluded, "lsp"},
		{"example.com/mod/internal/ls", ClassHardExcluded, "editor-service"},
		// Prefix matching is by path segment, not by string prefix.
		{"example.com/mod/internal/lspx", ClassUnselected, ""},
		{"example.com/mod/internal/other", ClassUnselected, ""},
		{"example.com/mod/cmd/tool", ClassUnselected, ""},
		// External split: std has no dot in the first segment.
		{"fmt", ClassExternalStd, ""},
		{"go/types", ClassExternalStd, ""},
		{"github.com/zeebo/xxh3", ClassExternalMod, ""},
		{"golang.org/x/sync/errgroup", ClassExternalMod, ""},
	}
	for _, c := range cases {
		class, category := p.Classify(c.pkg)
		if class != c.class || category != c.category {
			t.Errorf("Classify(%q) = (%s, %q), want (%s, %q)", c.pkg, class, category, c.class, c.category)
		}
	}
}

func TestInvalidRootNesting(t *testing.T) {
	valid := testProfile()
	if problem := valid.invalidRootNesting(); problem != "" {
		t.Fatalf("valid profile rejected: %s", problem)
	}

	ownedInsideExcluded := testProfile()
	ownedInsideExcluded.OwnedRoots = append(ownedInsideExcluded.OwnedRoots, "internal/lsp/inner")
	if problem := ownedInsideExcluded.invalidRootNesting(); problem == "" {
		t.Error("owned root inside hard-excluded root was not rejected")
	}

	testOnlyInsideExcluded := testProfile()
	testOnlyInsideExcluded.TestOnlyRoots = append(testOnlyInsideExcluded.TestOnlyRoots, "internal/ls/testutil")
	if problem := testOnlyInsideExcluded.invalidRootNesting(); problem == "" {
		t.Error("test-only root inside hard-excluded root was not rejected")
	}

	ownedInsideTestOnly := testProfile()
	ownedInsideTestOnly.OwnedRoots = append(ownedInsideTestOnly.OwnedRoots, "internal/testutil/prod")
	if problem := ownedInsideTestOnly.invalidRootNesting(); problem == "" {
		t.Error("owned root inside test-only root was not rejected")
	}
}
