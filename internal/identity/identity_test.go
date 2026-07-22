package identity

import (
	"errors"
	"testing"
)

// TestNewFileIDValidation proves the file-identity constructor accepts only
// workspace-relative, cleaned, slash-separated paths.
func TestNewFileIDValidation(t *testing.T) {
	for _, valid := range []string{"a.go", "pkg/file.go", "a/b/c_test.go"} {
		id, err := NewFileID(valid)
		if err != nil {
			t.Errorf("NewFileID(%q) rejected a valid path: %v", valid, err)
		}
		if string(id) != valid {
			t.Errorf("NewFileID(%q) = %q, want the path itself", valid, id)
		}
	}
	for _, invalid := range []string{
		"", ".", "/abs/a.go", "..", "../x.go", `a\b.go`, "./a.go", "pkg//f.go", "a#b.go", "pkg/",
	} {
		id, err := NewFileID(invalid)
		if err == nil {
			t.Errorf("NewFileID(%q) accepted an invalid path as %q", invalid, id)
			continue
		}
		var typed *Error
		if !errors.As(err, &typed) {
			t.Errorf("NewFileID(%q) error = %T, want *identity.Error", invalid, err)
		}
	}
}

// TestNewOccurrenceID proves the occurrence encoding is validated and
// deterministic.
func TestNewOccurrenceID(t *testing.T) {
	file, err := NewFileID("pkg/file.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	id, err := NewOccurrenceID(file, 10, 20, "Ident")
	if err != nil {
		t.Fatalf("NewOccurrenceID: %v", err)
	}
	if want := OccurrenceID("pkg/file.go#10-20/Ident"); id != want {
		t.Errorf("NewOccurrenceID = %q, want %q", id, want)
	}
	cases := []struct {
		file  FileID
		start int
		end   int
		kind  string
	}{
		{"", 0, 1, "Ident"},
		{file, -1, 1, "Ident"},
		{file, 5, 4, "Ident"},
		{file, 0, 1, ""},
		{file, 0, 1, "Bad#Kind"},
		{file, 0, 1, "Bad/Kind"},
	}
	for _, c := range cases {
		id, err := NewOccurrenceID(c.file, c.start, c.end, c.kind)
		if err == nil {
			t.Errorf("NewOccurrenceID(%q,%d,%d,%q) accepted invalid input as %q", c.file, c.start, c.end, c.kind, id)
			continue
		}
		var typed *Error
		if !errors.As(err, &typed) {
			t.Errorf("NewOccurrenceID(%q,%d,%d,%q) error = %T, want *identity.Error", c.file, c.start, c.end, c.kind, err)
		}
	}
}
