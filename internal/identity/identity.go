// Package identity owns canonical, machine-independent identities. Every
// identity is an opaque structure with an unexported representation: it can be
// obtained only through its validating constructor, so a non-zero identity is
// valid by construction and direct fabrication is a compile error. Zero values
// are detectably invalid via IsZero. Machine-local paths and //line display
// adjustments never enter an identity.
package identity

import (
	"fmt"
	"strings"
)

// Error is the typed validation failure of an identity constructor.
type Error struct {
	Identity string // which identity kind failed, e.g. "module", "file"
	Value    string
	Reason   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("GOTOTS_INVALID_IDENTITY: %s identity %q: %s", e.Identity, e.Value, e.Reason)
}

// reservedSeparators are the characters the canonical serializations reserve;
// no identity component may contain them.
const reservedSeparators = "#@\\\"'` \t\n\r"

func hasReserved(s string) bool { return strings.ContainsAny(s, reservedSeparators) }

// ModuleID is the semantic identity of one Go module: its declared module path
// plus, for versioned dependencies, its version. It is independent of checkout
// location, so relocated workspaces preserve identity, while two modules that
// contain identical relative file paths remain distinct.
type ModuleID struct {
	path    string
	version string
}

// NewModuleID validates a module path (and optional version) into a ModuleID.
func NewModuleID(path, version string) (ModuleID, error) {
	fail := func(reason string) (ModuleID, error) {
		return ModuleID{}, &Error{Identity: "module", Value: path + "@" + version, Reason: reason}
	}
	switch {
	case path == "":
		return fail("module path must not be empty")
	case hasReserved(path) || strings.Contains(path, "::"):
		return fail("module path contains a reserved character")
	case strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/"):
		return fail("module path must not begin or end with '/'")
	case strings.Contains(path, "//"):
		return fail("module path must not contain empty segments")
	case pathHasDotSegment(path):
		return fail("module path must not contain '.' or '..' segments")
	}
	if version != "" && (hasReserved(version) || strings.Contains(version, "/")) {
		return fail("module version contains a reserved character")
	}
	return ModuleID{path: path, version: version}, nil
}

func pathHasDotSegment(p string) bool {
	for start := 0; start <= len(p); {
		end := strings.IndexByte(p[start:], '/')
		if end < 0 {
			end = len(p)
		} else {
			end += start
		}
		segment := p[start:end]
		if segment == "." || segment == ".." {
			return true
		}
		if end == len(p) {
			return false
		}
		start = end + 1
	}
	return false
}

// IsZero reports whether m is the invalid zero identity.
func (m ModuleID) IsZero() bool { return m == ModuleID{} }

// Path is the declared module path.
func (m ModuleID) Path() string { return m.path }

// Version is the module version; empty for workspace (main) modules.
func (m ModuleID) Version() string { return m.version }

// String is the canonical serialization: path, or path@version.
func (m ModuleID) String() string {
	if m.version == "" {
		return m.path
	}
	return m.path + "@" + m.version
}

// PackageID is the semantic identity of one package: its owner plus its full
// import path. A module-owned import path must lie inside its module; the
// reserved owners admit any valid toolchain import path.
type PackageID struct {
	owner      Owner
	importPath string
}

// NewPackageID validates a package import path against its owner.
func NewPackageID(owner Owner, importPath string) (PackageID, error) {
	fail := func(reason string) (PackageID, error) {
		return PackageID{}, &Error{Identity: "package", Value: importPath, Reason: reason}
	}
	switch {
	case owner.IsZero():
		return fail("owner identity must not be zero")
	case importPath == "":
		return fail("import path must not be empty")
	case hasReserved(importPath) || strings.Contains(importPath, "::"):
		return fail("import path contains a reserved character")
	}
	if owner.Class() == OwnerModule {
		module := owner.Module()
		if importPath != module.path && !strings.HasPrefix(importPath, module.path+"/") {
			return fail("import path lies outside module " + module.path)
		}
	}
	return PackageID{owner: owner, importPath: importPath}, nil
}

// IsZero reports whether p is the invalid zero identity.
func (p PackageID) IsZero() bool { return p == PackageID{} }

// Owner is the owning identity.
func (p PackageID) Owner() Owner { return p.owner }

// ImportPath is the package's full import path.
func (p PackageID) ImportPath() string { return p.importPath }

// String is the canonical serialization: owner::importPath.
func (p PackageID) String() string {
	if p.IsZero() {
		return ""
	}
	return p.owner.String() + "::" + p.importPath
}

// FileID is the canonical identity of one source file: its owner plus its
// owner-relative slash path (module-relative for module owners, GOROOT/src-
// relative for the reserved owners). Workspace layout and machine paths never
// appear.
type FileID struct {
	owner Owner
	rel   string
}

// NewFileID validates an owner-relative path into a FileID.
func NewFileID(owner Owner, rel string) (FileID, error) {
	fail := func(reason string) (FileID, error) {
		return FileID{}, &Error{Identity: "file", Value: rel, Reason: reason}
	}
	switch {
	case owner.IsZero():
		return fail("owner identity must not be zero")
	case rel == "" || rel == ".":
		return fail("owner-relative path must name a file")
	case strings.Contains(rel, `\`):
		return fail("owner-relative path must be slash-separated")
	case strings.HasPrefix(rel, "/"):
		return fail("owner-relative path must be relative")
	case hasReserved(rel) || strings.Contains(rel, "::"):
		return fail("owner-relative path contains a reserved character")
	case strings.HasSuffix(rel, "/") || strings.Contains(rel, "//"):
		return fail("owner-relative path must be a cleaned file path")
	case pathHasDotSegment(rel):
		return fail("owner-relative path must not contain '.' or '..' segments")
	}
	return FileID{owner: owner, rel: rel}, nil
}

// IsZero reports whether f is the invalid zero identity.
func (f FileID) IsZero() bool { return f == FileID{} }

// Owner is the owning identity.
func (f FileID) Owner() Owner { return f.owner }

// Rel is the owner-relative slash path.
func (f FileID) Rel() string { return f.rel }

// String is the canonical serialization: owner::rel.
func (f FileID) String() string {
	if f.IsZero() {
		return ""
	}
	return f.owner.String() + "::" + f.rel
}

// SpanID is the identity of one physical byte range in a file, measured with
// //line directives ignored.
type SpanID struct {
	file  FileID
	start int
	end   int
}

// NewSpanID validates a physical byte range into a SpanID.
func NewSpanID(file FileID, startOffset, endOffset int) (SpanID, error) {
	fail := func(value, reason string) (SpanID, error) {
		return SpanID{}, &Error{Identity: "span", Value: value, Reason: reason}
	}
	if file.IsZero() {
		return fail("", "file identity must not be zero")
	}
	if startOffset < 0 {
		return fail(fmt.Sprintf("%d", startOffset), "start offset must be non-negative")
	}
	if endOffset < startOffset {
		return fail(fmt.Sprintf("%d-%d", startOffset, endOffset), "end offset must not precede start")
	}
	return SpanID{file: file, start: startOffset, end: endOffset}, nil
}

// IsZero reports whether s is the invalid zero identity.
func (s SpanID) IsZero() bool { return s == SpanID{} }

// File is the owning file identity.
func (s SpanID) File() FileID { return s.file }

// Start is the physical start byte offset.
func (s SpanID) Start() int { return s.start }

// End is the physical end byte offset.
func (s SpanID) End() int { return s.end }

// String is the canonical serialization: file#start-end.
func (s SpanID) String() string {
	if s.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s#%d-%d", s.file.String(), s.start, s.end)
}

// OccurrenceID is the canonical identity of one construct occurrence: its
// physical span plus the pinned numeric catalog kind identity. Equal source
// bytes at equal module-relative locations yield equal IDs on any machine.
type OccurrenceID struct {
	span SpanID
	kind uint16
}

// NewOccurrenceID validates one occurrence identity. The kind identity is the
// catalog's pinned numeric value, never a spelling.
func NewOccurrenceID(span SpanID, kindID uint16) (OccurrenceID, error) {
	if span.IsZero() {
		return OccurrenceID{}, &Error{Identity: "occurrence", Value: "", Reason: "span identity must not be zero"}
	}
	if kindID == 0 {
		return OccurrenceID{}, &Error{Identity: "occurrence", Value: span.String(), Reason: "kind identity must not be zero"}
	}
	return OccurrenceID{span: span, kind: kindID}, nil
}

// IsZero reports whether o is the invalid zero identity.
func (o OccurrenceID) IsZero() bool { return o == OccurrenceID{} }

// Span is the occurrence's physical span identity.
func (o OccurrenceID) Span() SpanID { return o.span }

// KindID is the pinned numeric catalog kind identity.
func (o OccurrenceID) KindID() uint16 { return o.kind }

// String is the canonical serialization: file#start-end/K<kindID>.
func (o OccurrenceID) String() string {
	if o.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s/K%d", o.span.String(), o.kind)
}
