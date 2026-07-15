// Package pinning defines the immutable source pin contract and verifies
// that a checkout and toolchain exactly match it before any extraction runs.
//
// A pin identifies both the source revision and the complete toolchain
// identity: version string, target platform, the go executable digest, a
// digest of the GOROOT source tree, and a digest of the GOROOT tool
// binaries actually executed during loading. Two different toolchains
// reporting the same version string do not pass. The candidate go
// executable's digest is checked before the binary is ever executed, and
// the Go frontend compiled into the gotots binary itself must match the
// pinned version, because go/parser and go/types run in-process.
//
// Source identity is stronger than cleanliness: every build input is
// reconciled against the pinned commit tree, so an ignored-but-present
// file cannot be admitted while git status stays clean.
//
// File layout: pin.go owns the pin schema; toolchain.go owns toolchain
// attestation; source.go owns checkout verification; tree.go owns the
// tracked-tree manifest.
package pinning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// Toolchain identifies the exact Go toolchain a pin was created with.
type Toolchain struct {
	Version string `json:"version"`
	GOOS    string `json:"goos"`
	GOARCH  string `json:"goarch"`
	// GoExecutableSha256 pins the exact go command binary.
	GoExecutableSha256 string `json:"goExecutableSha256"`
	// GorootSrcDigest pins the toolchain's GOROOT/src tree plus its VERSION
	// file: a running sha256 over sorted relative paths and file contents.
	GorootSrcDigest string `json:"gorootSrcDigest"`
	// GorootToolDigest pins the GOOS_GOARCH tool binaries (compile, link,
	// asm, ...) under GOROOT/pkg/tool that loading executes indirectly.
	GorootToolDigest string `json:"gorootToolDigest"`
}

// Pin is the immutable identity of one upstream source revision.
type Pin struct {
	SchemaVersion int       `json:"schemaVersion"`
	Upstream      string    `json:"upstream"`
	GoModule      string    `json:"goModule"`
	Revision      string    `json:"revision"`
	Toolchain     Toolchain `json:"toolchain"`
}

func isLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, c := range value {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// decodeStrict parses exactly one JSON document with no unknown fields and
// no trailing content.
func decodeStrict(data []byte, value any, what string) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("parse %s: %w", what, err)
	}
	if decoder.More() {
		return fmt.Errorf("parse %s: trailing content after JSON document", what)
	}
	return nil
}

// Load reads and validates a pin file.
func Load(path string) (*Pin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pin: %w", err)
	}
	var pin Pin
	if err := decodeStrict(data, &pin, "pin "+path); err != nil {
		return nil, err
	}
	if pin.SchemaVersion != 3 {
		return nil, fmt.Errorf("pin %s: unsupported schemaVersion %d", path, pin.SchemaVersion)
	}
	if pin.GoModule == "" || !isLowerHex(pin.Revision, 40) {
		return nil, fmt.Errorf("pin %s: goModule and a 40-hex-digit revision are required", path)
	}
	t := pin.Toolchain
	if t.Version == "" || t.GOOS == "" || t.GOARCH == "" ||
		!isLowerHex(t.GoExecutableSha256, 64) || !isLowerHex(t.GorootSrcDigest, 64) || !isLowerHex(t.GorootToolDigest, 64) {
		return nil, fmt.Errorf("pin %s: complete toolchain identity (version, goos, goarch, hex digests for executable, GOROOT src, and GOROOT tools) is required", path)
	}
	return &pin, nil
}
