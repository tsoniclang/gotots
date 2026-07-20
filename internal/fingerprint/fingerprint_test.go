package fingerprint

import (
	"os"
	"path/filepath"
	"testing"
)

func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func classes() []string { return []string{"core", "extern", "sourcemaps"} }

func classify(path string) (string, bool) {
	return PrefixClassifier([]PrefixRule{
		{Prefix: "core", Class: "core"},
		{Prefix: "extern", Class: "extern"},
		{Prefix: "maps", Class: "sourcemaps"},
	})(path)
}

// Every declared class is present even when empty, with a manifest and
// digest.
func TestEmptyClassIsPresent(t *testing.T) {
	dir := tree(t, map[string]string{"core/a.ts": "a"})
	report, err := Build(dir, classes(), classify, map[string]string{"pin": "x"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Classes) != 3 {
		t.Fatalf("classes = %d; want all 3 declared", len(report.Classes))
	}
	for _, c := range report.Classes {
		if c.Sha256 == "" {
			t.Fatalf("class %s missing digest", c.Name)
		}
	}
	if report.Classes[2].Name != "sourcemaps" || len(report.Classes[2].Files) != 0 {
		t.Fatalf("empty class not represented: %+v", report.Classes[2])
	}
}

// MUTATION (omitted file): an unattributed file fails the build.
func TestUnattributedFileFails(t *testing.T) {
	dir := tree(t, map[string]string{"core/a.ts": "a", "stray.txt": "x"})
	if _, err := Build(dir, classes(), classify, nil, nil); err == nil {
		t.Fatal("unattributed file must fail closed")
	}
}

// MUTATION (class reassignment): a classifier answering outside the
// declared universe fails.
func TestUndeclaredClassFails(t *testing.T) {
	dir := tree(t, map[string]string{"core/a.ts": "a"})
	bad := func(string) (string, bool) { return "rogue", true }
	if _, err := Build(dir, classes(), bad, nil, nil); err == nil {
		t.Fatal("undeclared class must fail closed")
	}
}

// MUTATION (path relocation): moving a file to another class changes
// both class digests and reports exact per-file differences.
func TestRelocationIsVisibleInDiff(t *testing.T) {
	before, err := Build(tree(t, map[string]string{"core/a.ts": "a", "extern/b.ts": "b"}), classes(), classify, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Build(tree(t, map[string]string{"core/a.ts": "a", "core/b.ts": "b"}), classes(), classify, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	diffs := Diff(before, after)
	want := map[string]bool{"core/b.ts:only-right": false, "extern/b.ts:only-left": false}
	for _, d := range diffs {
		key := d.Path + ":" + d.Kind
		if _, tracked := want[key]; tracked {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Fatalf("relocation diff missing %s (got %+v)", key, diffs)
		}
	}
}

// MUTATION (content change): a hash change is a per-file difference.
func TestHashChangeIsVisibleInDiff(t *testing.T) {
	before, err := Build(tree(t, map[string]string{"core/a.ts": "a"}), classes(), classify, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Build(tree(t, map[string]string{"core/a.ts": "CHANGED"}), classes(), classify, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	diffs := Diff(before, after)
	if len(diffs) != 1 || diffs[0].Kind != "hash-changed" || diffs[0].Path != "core/a.ts" {
		t.Fatalf("diffs = %+v", diffs)
	}
}

// Identical trees produce identical reports (reproducibility), and the
// semantic/environment split is preserved.
func TestReproducibleAndSplitIdentity(t *testing.T) {
	files := map[string]string{"core/a.ts": "a", "maps/a.map": "m"}
	one, err := Build(tree(t, files), classes(), classify, map[string]string{"pin": "p"}, map[string]string{"host": "local"})
	if err != nil {
		t.Fatal(err)
	}
	two, err := Build(tree(t, files), classes(), classify, map[string]string{"pin": "p"}, map[string]string{"host": "other"})
	if err != nil {
		t.Fatal(err)
	}
	if len(Diff(one, two)) != 0 {
		t.Fatal("identical trees must fingerprint identically")
	}
	if one.Semantic["pin"] != "p" || one.Environment["host"] != "local" {
		t.Fatal("semantic/environment split lost")
	}
	// Environment differences never affect class digests.
	for i := range one.Classes {
		if one.Classes[i].Sha256 != two.Classes[i].Sha256 {
			t.Fatal("environment leaked into class identity")
		}
	}
}
