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

func mustModuleOwner(t *testing.T, path, version string) Owner {
	t.Helper()
	o, err := NewModuleOwner(mustModule(t, path, version))
	if err != nil {
		t.Fatalf("NewModuleOwner: %v", err)
	}
	return o
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
	if _, err := NewModuleID("dotlessdep/lib", ""); err != nil {
		t.Errorf("dotless module path rejected: %v", err)
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

// TestOwnerClasses proves the closed owner domain: module owners are
// constructor-validated, the reserved owners are distinct, and no fabricated
// module stands in for the standard library.
func TestOwnerClasses(t *testing.T) {
	mod := mustModuleOwner(t, "example.com/a", "v1.0.0")
	if mod.Class() != OwnerModule || mod.String() != "mod=example.com/a@v1.0.0" {
		t.Errorf("module owner = %s (%s)", mod, mod.Class())
	}
	std := StandardLibraryOwner()
	tool := ToolchainOwner()
	lang := LanguagePseudoOwner()
	if std.String() != "std" || tool.String() != "toolchain" || lang.String() != "lang" {
		t.Errorf("reserved owner serializations: %s %s %s", std, tool, lang)
	}
	seen := map[string]bool{}
	for _, o := range []Owner{mod, std, tool, lang} {
		if o.IsZero() || !o.Class().Valid() {
			t.Errorf("owner %s invalid", o)
		}
		if seen[o.String()] {
			t.Errorf("owner serialization %q not unique", o)
		}
		seen[o.String()] = true
	}
	// A module literally named "std" cannot collide with the reserved owner.
	stdModule := mustModuleOwner(t, "std", "")
	if stdModule.String() == std.String() {
		t.Error("module path 'std' collides with the reserved standard-library owner")
	}
	if _, err := NewModuleOwner(ModuleID{}); err == nil {
		t.Error("zero module accepted as owner")
	}
	if (Owner{}).IsZero() != true {
		t.Error("zero owner must report IsZero")
	}
}

// TestPackageIDValidation proves package identity requires a containing owner:
// module paths stay inside their module while reserved owners admit toolchain
// import paths.
func TestPackageIDValidation(t *testing.T) {
	mod := mustModuleOwner(t, "example.com/proj", "")
	p, err := NewPackageID(mod, "example.com/proj/internal/x")
	if err != nil {
		t.Fatalf("NewPackageID: %v", err)
	}
	if p.String() != "mod=example.com/proj::example.com/proj/internal/x" {
		t.Errorf("String() = %q", p.String())
	}
	if _, err := NewPackageID(mod, "other.com/pkg"); err == nil {
		t.Error("accepted import path outside the module")
	}
	if _, err := NewPackageID(mod, "example.com/projX"); err == nil {
		t.Error("accepted prefix collision")
	}
	std, err := NewPackageID(StandardLibraryOwner(), "fmt")
	if err != nil {
		t.Fatalf("std package id: %v", err)
	}
	if std.String() != "std::fmt" {
		t.Errorf("std String() = %q", std.String())
	}
	if _, err := NewPackageID(ToolchainOwner(), "cmd/gofmt"); err != nil {
		t.Errorf("toolchain package id: %v", err)
	}
	if _, err := NewPackageID(Owner{}, "fmt"); err == nil {
		t.Error("accepted zero owner")
	}
}

// TestFileIDDistinguishesOwners is the collision regression: identical
// relative paths under different owners are distinct, and identity never
// embeds a machine location.
func TestFileIDDistinguishesOwners(t *testing.T) {
	a := mustModuleOwner(t, "example.com/a", "")
	b := mustModuleOwner(t, "example.com/b", "")
	fa, err := NewFileID(a, "pkg/same.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	fb, err := NewFileID(b, "pkg/same.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	fstd, err := NewFileID(StandardLibraryOwner(), "os/file.go")
	if err != nil {
		t.Fatalf("std file id: %v", err)
	}
	if fa == fb || fa.String() == fb.String() {
		t.Errorf("identical relative paths under distinct modules collide")
	}
	if fstd.String() != "std::os/file.go" {
		t.Errorf("std file String() = %q", fstd.String())
	}
	fa2, err := NewFileID(mustModuleOwner(t, "example.com/a", ""), "pkg/same.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	if fa != fa2 {
		t.Error("same owner + relative path must be equal")
	}
	for _, bad := range []string{"", ".", "/abs.go", "../up.go", "a/../b.go", `a\b.go`, "d/", "a//b.go", "a#b.go", "a::b.go"} {
		if _, err := NewFileID(a, bad); err == nil {
			t.Errorf("NewFileID(%q) accepted invalid path", bad)
		}
	}
	if _, err := NewFileID(Owner{}, "ok.go"); err == nil {
		t.Error("accepted zero owner")
	}
}

// TestSpanAndOccurrenceIDs proves span/occurrence construction is validated
// and encodes the pinned numeric kind identity, not a spelling.
func TestSpanAndOccurrenceIDs(t *testing.T) {
	owner := mustModuleOwner(t, "example.com/a", "")
	f, err := NewFileID(owner, "pkg/x.go")
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
	if o.String() != "mod=example.com/a::pkg/x.go#10-20/K47" {
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
	var typed *Error
	if _, err := NewOccurrenceID(s, 0); !errors.As(err, &typed) {
		t.Errorf("zero kind error = %T, want *identity.Error", err)
	}
}
