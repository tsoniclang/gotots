package source

import (
	"crypto/sha256"
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// ManifestUnitRecord is one provider-owned unit's content-addressed manifest
// record: the canonical identity components plus the selected-span hash. The
// consumer reconstructs the identity through the validating constructors and
// recomputes the hash from its own selected bytes — nothing is trusted as a
// bare count.
type ManifestUnitRecord struct {
	Unit       string // canonical SourceUnitID serialization (cross-checked)
	Kind       uint8  // pinned identity.UnitKind value
	Start      int    // byte offsets within the selected bytes
	End        int
	Name       string // display only, never identity
	Hash       string // SourceSpanHash hex of the selected span bytes
	CDependent bool   // typed cgo evidence recorded at manifest production
}

// ManifestFileRecord is one file's manifest entry: the selected-byte digest
// that binds it plus the file's complete unit ledger (top-level and interior).
type ManifestFileRecord struct {
	Digest string // sha256 hex of the file's selected bytes
	Units  []ManifestUnitRecord
}

// UnitManifest is the request-supplied provider unit manifest: verified
// content-addressed unit evidence for provider-owned files, keyed by
// canonical file identity. The zero manifest supplies nothing, so any
// manifest-mode file fails closed until the request carries a produced
// artifact.
type UnitManifest struct {
	files map[string]ManifestFileRecord
}

// NewUnitManifest validates one manifest.
func NewUnitManifest(files map[string]ManifestFileRecord) (UnitManifest, error) {
	out := UnitManifest{files: make(map[string]ManifestFileRecord, len(files))}
	for id, record := range files {
		if id == "" {
			return UnitManifest{}, &LoadError{Reason: "unit manifest names an empty file identity"}
		}
		if record.Digest == "" {
			return UnitManifest{}, &LoadError{Reason: "unit manifest record " + id + " carries no selected-byte digest"}
		}
		out.files[id] = record
	}
	return out, nil
}

// fileRecord answers one file's manifest record.
func (m UnitManifest) fileRecord(id identity.FileID) (ManifestFileRecord, bool) {
	record, ok := m.files[id.String()]
	return record, ok
}

// joinManifestUnits exact-joins one manifest-mode file's manifest record
// against the local bounded census and returns the interior units the
// manifest supplies:
//
//   - the record's digest must equal the file's selected-byte digest (a stale
//     manifest fails closed);
//   - every locally censused top-level unit must appear in the manifest with
//     an identical hash, and every non-interior manifest unit must appear in
//     the local census (both one-sided lists are reported);
//   - interior units (function literals) are reconstructed through the
//     validating identity constructors, their canonical serialization must
//     equal the record's, and their hashes are recomputed from the local
//     selected bytes.
func joinManifestUnits(u *Universe, file *LoadedFile, record ManifestFileRecord, raw []byte) ([]SourceUnit, error) {
	fail := func(reason string) ([]SourceUnit, error) {
		return nil, &LoadError{Dir: u.request.Dir, Reason: "provider manifest for " + file.id.String() + ": " + reason}
	}
	if record.Digest != file.byteDigest.String() {
		return fail("stale: selected-byte digest " + file.byteDigest.String() + " vs manifest " + record.Digest)
	}
	local := map[string]SourceUnit{}
	for _, unit := range file.units {
		local[unit.ID().String()] = unit
	}
	var interiors []SourceUnit
	var missingLocal, extraManifest []string
	seenLocal := map[string]bool{}
	for _, mu := range record.Units {
		kind := identity.UnitKind(mu.Kind)
		if kind == identity.UnitFuncLitBody {
			unit, err := manifestInterior(u, file, mu, raw)
			if err != nil {
				return nil, err
			}
			interiors = append(interiors, unit)
			continue
		}
		unit, censused := local[mu.Unit]
		if !censused {
			extraManifest = append(extraManifest, mu.Unit)
			continue
		}
		seenLocal[mu.Unit] = true
		if unit.Hash().String() != mu.Hash {
			return fail("unit " + mu.Unit + " hash diverges from the local census")
		}
	}
	for id := range local {
		if !seenLocal[id] {
			missingLocal = append(missingLocal, id)
		}
	}
	if len(missingLocal)+len(extraManifest) > 0 {
		return fail(fmt.Sprintf("top-level unit join failed; manifest-missing=%v manifest-extra=%v", missingLocal, extraManifest))
	}
	return interiors, nil
}

// manifestInterior reconstructs one manifest-supplied interior unit through
// the validating constructors and content addressing.
func manifestInterior(u *Universe, file *LoadedFile, mu ManifestUnitRecord, raw []byte) (SourceUnit, error) {
	fail := func(reason string) (SourceUnit, error) {
		return SourceUnit{}, &LoadError{Dir: u.request.Dir, Reason: "provider manifest for " + file.id.String() + ": " + reason}
	}
	spanID, err := identity.NewSpanID(file.id, mu.Start, mu.End)
	if err != nil {
		return SourceUnit{}, err
	}
	unitID, err := identity.NewSourceUnitID(spanID, identity.UnitKind(mu.Kind))
	if err != nil {
		return SourceUnit{}, err
	}
	if unitID.String() != mu.Unit {
		return fail("interior unit identity " + mu.Unit + " is not canonical (reconstructed " + unitID.String() + ")")
	}
	if mu.End > len(raw) {
		return fail("interior unit " + mu.Unit + " span exceeds the selected bytes")
	}
	hash := sha256.Sum256(raw[mu.Start:mu.End])
	if fmt.Sprintf("%x", hash) != mu.Hash {
		return fail("interior unit " + mu.Unit + " hash diverges from the selected bytes")
	}
	start := positionAt(raw, mu.Start)
	end := positionAt(raw, mu.End)
	return SourceUnit{
		id:         unitID,
		name:       mu.Name,
		span:       Span{Start: start, End: end},
		hash:       hash,
		cDependent: mu.CDependent,
	}, nil
}

// positionAt computes the 1-based line/column of a byte offset from the
// selected bytes themselves — display positions are recomputed locally, never
// trusted from the artifact.
func positionAt(raw []byte, offset int) Position {
	line, lastNL := 1, -1
	for i := 0; i < offset && i < len(raw); i++ {
		if raw[i] == '\n' {
			line++
			lastNL = i
		}
	}
	return Position{Line: line, Column: offset - lastNL, Offset: offset}
}
