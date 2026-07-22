package compiler

import (
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/source"
)

// withManifest runs the manifest-producing audit for the request, writes the
// artifact, and returns the request with the artifact selected — the ordinary
// produce-then-consume pipeline every closure containing provider-owned files
// requires.
func withManifest(t *testing.T, req source.Request) source.Request {
	t.Helper()
	artifact, err := AuditCatalog(req)
	if err != nil {
		t.Fatalf("AuditCatalog: %v", err)
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := analyze.WriteAuditArtifact(artifact, path); err != nil {
		t.Fatal(err)
	}
	req.AuditArtifact = path
	return req
}
