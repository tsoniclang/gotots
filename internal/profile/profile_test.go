package profile

import "testing"

func testProfile() *Profile {
	return &Profile{
		GoModule:      "example.com/mod",
		OwnedRoots:    []string{"internal/execute", "internal/scanner", "internal/vfs"},
		TestOnlyRoots: []string{"internal/execute/tsctests", "internal/testutil", "internal/vfs/vfstest"},
		OutsideUniverseRoots: map[string][]string{
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
		{"example.com/mod/internal/testutil/lsptestutil", ClassOutsideUniverse, "editor-service"},
		{"example.com/mod/internal/lsp", ClassOutsideUniverse, "lsp"},
		{"example.com/mod/internal/lsp/lsproto", ClassOutsideUniverse, "lsp"},
		{"example.com/mod/internal/ls", ClassOutsideUniverse, "editor-service"},
		// Prefix matching is by path segment, not by string prefix.
		{"example.com/mod/internal/lspx", ClassUnselected, ""},
		{"example.com/mod/internal/other", ClassUnselected, ""},
		{"example.com/mod/cmd/tool", ClassUnselected, ""},
		// Everything outside the module is one external class here; the
		// std/module split needs loader evidence and happens in inventory.
		{"fmt", ClassExternal, ""},
		{"go/types", ClassExternal, ""},
		{"github.com/zeebo/xxh3", ClassExternal, ""},
		{"golang.org/x/sync/errgroup", ClassExternal, ""},
	}
	for _, c := range cases {
		class, category := p.Classify(c.pkg)
		if class != c.class || category != c.category {
			t.Errorf("Classify(%q) = (%s, %q), want (%s, %q)", c.pkg, class, category, c.class, c.category)
		}
	}
}

func TestValidate(t *testing.T) {
	base := func() *Profile {
		p := testProfile()
		p.BuildProfiles = []BuildProfile{{Name: "linux-amd64", GOOS: "linux", GOARCH: "amd64", GOAMD64: "v1"}}
		return p
	}
	if err := base().validate(); err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}

	duplicateBuild := base()
	duplicateBuild.BuildProfiles = append(duplicateBuild.BuildProfiles, duplicateBuild.BuildProfiles[0])
	if err := duplicateBuild.validate(); err == nil {
		t.Error("duplicate build profile name was not rejected")
	}

	for _, bad := range []string{"./internal/x", "internal/x/", "/abs", "../up", ".", ""} {
		p := base()
		p.OwnedRoots = append(p.OwnedRoots, bad)
		if err := p.validate(); err == nil {
			t.Errorf("non-normalized root %q was not rejected", bad)
		}
	}

	duplicateRoot := base()
	duplicateRoot.TestOnlyRoots = append(duplicateRoot.TestOnlyRoots, "internal/scanner")
	if err := duplicateRoot.validate(); err == nil {
		t.Error("root appearing in two lists was not rejected")
	}

	missingAmd64 := base()
	missingAmd64.BuildProfiles[0].GOAMD64 = ""
	if err := missingAmd64.validate(); err == nil {
		t.Error("amd64 profile without goamd64 was not rejected")
	}

	cgo := base()
	cgo.BuildProfiles[0].CgoEnabled = true
	if err := cgo.validate(); err == nil {
		t.Error("cgoEnabled=true was not rejected")
	}

	tags := base()
	tags.BuildProfiles[0].Tags = []string{"race"}
	if err := tags.validate(); err == nil {
		t.Error("build tags were not rejected")
	}

	crossCategory := base()
	crossCategory.OutsideUniverseRoots["other-category"] = []string{"internal/ls/nested"}
	if err := crossCategory.validate(); err == nil {
		t.Error("cross-category nested exclusion roots were not rejected")
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
		t.Error("owned root inside outside-universe root was not rejected")
	}

	testOnlyInsideExcluded := testProfile()
	testOnlyInsideExcluded.TestOnlyRoots = append(testOnlyInsideExcluded.TestOnlyRoots, "internal/ls/testutil")
	if problem := testOnlyInsideExcluded.invalidRootNesting(); problem == "" {
		t.Error("test-only root inside outside-universe root was not rejected")
	}

	ownedInsideTestOnly := testProfile()
	ownedInsideTestOnly.OwnedRoots = append(ownedInsideTestOnly.OwnedRoots, "internal/testutil/prod")
	if problem := ownedInsideTestOnly.invalidRootNesting(); problem == "" {
		t.Error("owned root inside test-only root was not rejected")
	}
}
