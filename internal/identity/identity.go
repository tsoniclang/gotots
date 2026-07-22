// Package identity owns canonical, machine-independent identities. An
// identity is constructed only through a validating constructor here; no other
// package assembles identity strings. Machine-local paths never enter an
// identity.
package identity

import (
	"fmt"
	"path"
	"strings"
)

// Error is the typed validation failure of an identity constructor.
type Error struct {
	Identity string // which identity kind failed, e.g. "file", "occurrence"
	Value    string
	Reason   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("GOTOTS_INVALID_IDENTITY: %s identity %q: %s", e.Identity, e.Value, e.Reason)
}

// FileID is the canonical identity of one source file: its workspace-relative
// slash-separated path. It is independent of the machine, checkout location,
// and //line display adjustments.
type FileID string

// NewFileID validates a workspace-relative path into a FileID. The path must
// be non-empty, slash-separated, cleaned, relative, inside the workspace, and
// free of the '#' separator the occurrence encoding reserves.
func NewFileID(workspaceRel string) (FileID, error) {
	fail := func(reason string) (FileID, error) {
		return "", &Error{Identity: "file", Value: workspaceRel, Reason: reason}
	}
	switch {
	case workspaceRel == "":
		return fail("must not be empty")
	case workspaceRel == ".":
		return fail("names the workspace root, not a file")
	case strings.Contains(workspaceRel, `\`):
		return fail("must be slash-separated")
	case path.IsAbs(workspaceRel):
		return fail("must be workspace-relative, not absolute")
	case path.Clean(workspaceRel) != workspaceRel:
		return fail("must be a cleaned path")
	case workspaceRel == ".." || strings.HasPrefix(workspaceRel, "../"):
		return fail("must not escape the workspace root")
	case strings.Contains(workspaceRel, "#"):
		return fail("must not contain the reserved separator '#'")
	}
	return FileID(workspaceRel), nil
}

// OccurrenceID is the canonical identity of one construct occurrence: file
// identity plus physical byte span plus construct kind name. Equal source
// bytes at equal workspace-relative locations yield equal IDs on any machine.
type OccurrenceID string

// NewOccurrenceID validates and encodes one occurrence identity. Offsets are
// physical byte offsets (//line-independent); the kind name comes from the
// closed catalog and must not contain the encoding separators.
func NewOccurrenceID(file FileID, startOffset, endOffset int, kindName string) (OccurrenceID, error) {
	fail := func(value, reason string) (OccurrenceID, error) {
		return "", &Error{Identity: "occurrence", Value: value, Reason: reason}
	}
	if file == "" {
		return fail("", "file identity must not be empty")
	}
	if startOffset < 0 {
		return fail(fmt.Sprintf("%d", startOffset), "start offset must be non-negative")
	}
	if endOffset < startOffset {
		return fail(fmt.Sprintf("%d-%d", startOffset, endOffset), "end offset must not precede start")
	}
	if kindName == "" || strings.ContainsAny(kindName, "#/") {
		return fail(kindName, "kind name must be non-empty and free of '#' and '/'")
	}
	return OccurrenceID(fmt.Sprintf("%s#%d-%d/%s", file, startOffset, endOffset, kindName)), nil
}
