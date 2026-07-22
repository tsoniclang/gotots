package identity

import (
	"errors"
	"testing"
)

func mustModule(t *testing.T, path, version string) ModuleID {
	t.Helper()
	m, err := NewModuleID(path, version)
	if err != nil {
		t.Fatalf("NewModuleID(%q,%q): %v", path, version, err)
	}
	return m
}

// TestModuleIDValidation proves module identity is semantic (path+version) and
// validated.
func TestModuleIDValidation(t *testing.T) {
	m := mustModule(t, "example.com/proj", "")
	if m.String() != "example.com/proj" {
		t.Errorf("String() = %q", m.String())
	}
	v := mustModule(t, "example.com/dep", "v1.2.3")
	if v.String() != "example.com/dep@v1.2.3" {
		t.Errorf("String() = %q", v.String())
	}
	if (ModuleID{}).IsZero() != true || m.IsZero() {
		t.Error("IsZero misreports")
	}
	for _, bad := range [][2]string{
		{"", ""}, {"/abs", ""}, {"trail/", ""}, {"a//b", ""}, {"a/../b", ""},
		{"a b", ""}, {"a#b", ""}, {"a::b", ""}, {"ok", "v1/2"}, {"ok", "v 1"},
	} {
		if _, err := NewModuleID(bad[0], bad[1]); err == nil {
			t.Errorf("NewModuleID(%q,%q) accepted invalid input", bad[0], bad[1])
		}
	}
}

// TestPackageIDValidation proves package identity requires a containing module.
func TestPackageIDValidation(t *testing.T) {
	m := mustModule(t, "example.com/proj", "")
	p, err := NewPackageID(m, "example.com/proj/internal/x")
	if err != nil {
		t.Fatalf("NewPackageID: %v", err)
	}
	if p.String() != "example.com/proj::example.com/proj/internal/x" {
		t.Errorf("String() = %q", p.String())
	}
	if _, err := NewPackageID(m, "other.com/pkg"); err == nil {
		t.Error("accepted import path outside the module")
	}
	if _, err := NewPackageID(ModuleID{}, "example.com/proj"); err == nil {
		t.Error("accepted zero module")
	}
	if _, err := NewPackageID(m, "example.com/projX"); err == nil {
		t.Error("accepted prefix collision (segment boundary violated)")
	}
}

// TestFileIDDistinguishesModules is the collision regression: identical
// relative paths in two different modules yield distinct identities, while the
// same module relocated on disk yields the same identity (disk location never
// enters the constructor).
func TestFileIDDistinguishesModules(t *testing.T) {
	a := mustModule(t, "example.com/a", "")
	b := mustModule(t, "example.com/b", "")
	fa, err := NewFileID(a, "pkg/same.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	fb, err := NewFileID(b, "pkg/same.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	if fa == fb || fa.String() == fb.String() {
		t.Errorf("identical relative paths in distinct modules collide: %q vs %q", fa, fb)
	}
	fa2, err := NewFileID(mustModule(t, "example.com/a", ""), "pkg/same.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	if fa != fa2 {
		t.Error("same module identity + relative path must be equal")
	}
	for _, bad := range []string{"", ".", "/abs.go", "../up.go", "a/../b.go", `a\b.go`, "d/", "a//b.go", "a#b.go", "a::b.go"} {
		if _, err := NewFileID(a, bad); err == nil {
			t.Errorf("NewFileID(%q) accepted invalid path", bad)
		}
	}
	if _, err := NewFileID(ModuleID{}, "ok.go"); err == nil {
		t.Error("accepted zero module")
	}
}

// TestSpanAndOccurrenceIDs proves span/occurrence construction is validated and
// encodes the pinned numeric kind identity, not a spelling.
func TestSpanAndOccurrenceIDs(t *testing.T) {
	m := mustModule(t, "example.com/a", "")
	f, err := NewFileID(m, "pkg/x.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	s, err := NewSpanID(f, 10, 20)
	if err != nil {
		t.Fatalf("NewSpanID: %v", err)
	}
	o, err := NewOccurrenceID(s, 47)
	if err != nil {
		t.Fatalf("NewOccurrenceID: %v", err)
	}
	if o.String() != "example.com/a::pkg/x.go#10-20/K47" {
		t.Errorf("String() = %q", o.String())
	}
	if o.KindID() != 47 {
		t.Errorf("KindID() = %d", o.KindID())
	}
	if _, err := NewSpanID(FileID{}, 0, 1); err == nil {
		t.Error("accepted zero file")
	}
	if _, err := NewSpanID(f, -1, 1); err == nil {
		t.Error("accepted negative start")
	}
	if _, err := NewSpanID(f, 5, 4); err == nil {
		t.Error("accepted end before start")
	}
	if _, err := NewOccurrenceID(SpanID{}, 1); err == nil {
		t.Error("accepted zero span")
	}
	if _, err := NewOccurrenceID(s, 0); err == nil {
		t.Error("accepted zero kind")
	}
	var typed *Error
	_, err = NewOccurrenceID(s, 0)
	if !errors.As(err, &typed) {
		t.Errorf("error type = %T, want *identity.Error", err)
	}
}
