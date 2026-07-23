// Package sourceplan owns the request-bound per-file choice between local
// selected syntax and a certified provider structural graph. It runs before
// definition construction and never chooses provider or evidence depth.
package sourceplan

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

// Kind is the closed structural-source domain.
type Kind uint8

const (
	KindInvalid Kind = iota
	KindLocalSyntax
	KindCertifiedGraph
)

func (k Kind) Valid() bool { return k == KindLocalSyntax || k == KindCertifiedGraph }

func (k Kind) String() string {
	switch k {
	case KindLocalSyntax:
		return "local-syntax"
	case KindCertifiedGraph:
		return "certified-graph"
	default:
		return fmt.Sprintf("sourceplan.Kind(%d)", uint8(k))
	}
}

// Purpose is the closed lifecycle class of a structural-source plan.
type Purpose uint8

const (
	PurposeInvalid Purpose = iota
	PurposeCompilation
	PurposeProviderProduction
)

func (p Purpose) Valid() bool {
	return p == PurposeCompilation || p == PurposeProviderProduction
}

func (p Purpose) String() string {
	switch p {
	case PurposeCompilation:
		return "compilation"
	case PurposeProviderProduction:
		return "provider-production"
	default:
		return fmt.Sprintf("sourceplan.Purpose(%d)", uint8(p))
	}
}

// File is one validated per-file structural source decision.
type File struct {
	id             identity.FileID
	kind           Kind
	contractDigest string
	artifactDigest string
}

func (f File) ID() identity.FileID    { return f.id }
func (f File) Kind() Kind             { return f.kind }
func (f File) ContractDigest() string { return f.contractDigest }
func (f File) ArtifactDigest() string { return f.artifactDigest }

// Plan is the immutable complete per-file structural-source plan.
type Plan struct {
	purpose     Purpose
	files       []File
	byID        map[identity.FileID]*File
	synthetic   []SyntheticOwner
	syntheticBy map[identity.PackageID]*SyntheticOwner
	fingerprint string
}

func (p *Plan) Files() []File       { return append([]File(nil), p.files...) }
func (p *Plan) Purpose() Purpose    { return p.purpose }
func (p *Plan) Fingerprint() string { return p.fingerprint }
func (p *Plan) For(id identity.FileID) (File, bool) {
	file, ok := p.byID[id]
	if !ok {
		return File{}, false
	}
	return *file, true
}

// SyntheticOwner is the structural-source decision for package-generated
// semantic owners such as cgo checked-view declarations.
type SyntheticOwner struct {
	pkg            identity.PackageID
	kind           Kind
	contractDigest string
	artifactDigest string
}

func (o SyntheticOwner) Package() identity.PackageID { return o.pkg }
func (o SyntheticOwner) Kind() Kind                  { return o.kind }
func (o SyntheticOwner) ContractDigest() string      { return o.contractDigest }
func (o SyntheticOwner) ArtifactDigest() string      { return o.artifactDigest }

func (p *Plan) SyntheticOwners() []SyntheticOwner {
	return append([]SyntheticOwner(nil), p.synthetic...)
}

// LocalFileIDs returns the exact physical files requiring local semantic
// hydration. It is a projection of the sealed source plan, not a second
// selection policy.
func (p *Plan) LocalFileIDs() []identity.FileID {
	var out []identity.FileID
	for _, file := range p.files {
		if file.kind == KindLocalSyntax {
			out = append(out, file.id)
		}
	}
	return out
}

// LocalSyntheticPackages returns checked-view owners requiring local semantic
// hydration.
func (p *Plan) LocalSyntheticPackages() []identity.PackageID {
	var out []identity.PackageID
	for _, owner := range p.synthetic {
		if owner.kind == KindLocalSyntax {
			out = append(out, owner.pkg)
		}
	}
	return out
}

func (p *Plan) SyntheticFor(
	pkg identity.PackageID,
) (SyntheticOwner, bool) {
	owner, ok := p.syntheticBy[pkg]
	if !ok {
		return SyntheticOwner{}, false
	}
	return *owner, true
}

// CertifiedInput is the request-bound availability evidence of one certified
// provider graph artifact.
type CertifiedInput struct {
	Digest   string
	Files    map[string]bool
	Packages map[string]bool
}

// Build conservatively chooses local syntax whenever any declared rule can
// require automatic translation for a definition in the file. Every other
// ordinary source file must be present in the certified input.
func Build(
	universe *source.Universe,
	selected contract.Contract,
	certified CertifiedInput,
) (*Plan, error) {
	return build(
		universe, selected, certified, PurposeCompilation,
	)
}

// BuildForAudit derives the same intended local/certified partition while
// authorizing local extraction of the certified side solely for artifact
// production.
func BuildForAudit(
	universe *source.Universe,
	selected contract.Contract,
) (*Plan, error) {
	return build(
		universe,
		selected,
		CertifiedInput{},
		PurposeProviderProduction,
	)
}

func build(
	universe *source.Universe,
	selected contract.Contract,
	certified CertifiedInput,
	purpose Purpose,
) (*Plan, error) {
	out := &Plan{
		purpose:     purpose,
		byID:        map[identity.FileID]*File{},
		syntheticBy: map[identity.PackageID]*SyntheticOwner{},
	}
	for _, pkg := range universe.Packages() {
		if pkg.Disposition() != source.DispositionOrdinarySource &&
			pkg.Disposition() != source.DispositionUnsafeIntrinsic {
			continue
		}
		packageLocal := packageMayTranslate(pkg, selected)
		localCgo := false
		for _, loadedFile := range pkg.Files() {
			local := pkg.Disposition() == source.DispositionUnsafeIntrinsic ||
				packageLocal ||
				exactDefinitionMayTranslate(
					loadedFile.ID(), pkg, selected,
				)
			kind := KindCertifiedGraph
			artifactDigest := certified.Digest
			if local {
				kind = KindLocalSyntax
				artifactDigest = ""
			} else if purpose == PurposeCompilation &&
				(certified.Digest == "" || !certified.Files[loadedFile.ID().String()]) {
				return nil, &Error{Reason: loadedFile.ID().String() +
					" requires a certified structural graph and none is bound"}
			}
			record := File{
				id: loadedFile.ID(), kind: kind, contractDigest: selected.Fingerprint(),
				artifactDigest: artifactDigest,
			}
			out.files = append(out.files, record)
			localCgo = localCgo ||
				(loadedFile.CgoOriginal() &&
					kind == KindLocalSyntax)
		}
		if pkg.HasCheckedView() {
			kind := KindCertifiedGraph
			artifactDigest := certified.Digest
			if localCgo {
				kind = KindLocalSyntax
				artifactDigest = ""
			} else if purpose == PurposeCompilation &&
				(certified.Digest == "" ||
					!certified.Packages[pkg.ID().String()]) {
				return nil, &Error{Reason: pkg.ID().String() +
					" requires a certified synthetic graph and none is bound"}
			}
			owner := SyntheticOwner{
				pkg: pkg.ID(), kind: kind,
				contractDigest: selected.Fingerprint(),
				artifactDigest: artifactDigest,
			}
			out.synthetic = append(out.synthetic, owner)
		}
	}
	if err := finishPlan(out); err != nil {
		return nil, err
	}
	return out, nil
}

func finishPlan(out *Plan) error {
	if out == nil || !out.purpose.Valid() {
		return &Error{Reason: "source plan has no valid purpose"}
	}
	sort.Slice(out.files, func(i, j int) bool {
		return out.files[i].id.String() < out.files[j].id.String()
	})
	sort.Slice(out.synthetic, func(i, j int) bool {
		return out.synthetic[i].pkg.String() < out.synthetic[j].pkg.String()
	})
	for index := range out.files {
		file := &out.files[index]
		if file.id.IsZero() || !file.kind.Valid() ||
			file.contractDigest == "" ||
			(file.kind == KindLocalSyntax && file.artifactDigest != "") ||
			(file.kind == KindCertifiedGraph &&
				out.purpose == PurposeCompilation &&
				file.artifactDigest == "") {
			return &Error{
				Reason: "invalid source-plan file " + file.id.String(),
			}
		}
		if _, duplicate := out.byID[file.id]; duplicate {
			return &Error{
				Reason: "duplicate source-plan file " + file.id.String(),
			}
		}
		out.byID[file.id] = file
	}
	for index := range out.synthetic {
		owner := &out.synthetic[index]
		if owner.pkg.IsZero() || !owner.kind.Valid() ||
			owner.contractDigest == "" ||
			(owner.kind == KindLocalSyntax && owner.artifactDigest != "") ||
			(owner.kind == KindCertifiedGraph &&
				out.purpose == PurposeCompilation &&
				owner.artifactDigest == "") {
			return &Error{
				Reason: "invalid synthetic source-plan owner " +
					owner.pkg.String(),
			}
		}
		if _, duplicate := out.syntheticBy[owner.pkg]; duplicate {
			return &Error{
				Reason: "duplicate synthetic owner " + owner.pkg.String(),
			}
		}
		out.syntheticBy[owner.pkg] = owner
	}
	hash := sha256.New()
	fmt.Fprintf(hash, "purpose:%s\n", out.purpose)
	for _, file := range out.files {
		fmt.Fprintf(hash, "%s|%s|%s|%s\n",
			file.id, file.kind, file.contractDigest, file.artifactDigest)
	}
	for _, owner := range out.synthetic {
		fmt.Fprintf(hash, "synthetic:%s|%s|%s|%s\n",
			owner.pkg, owner.kind, owner.contractDigest, owner.artifactDigest)
	}
	out.fingerprint = fmt.Sprintf("%x", hash.Sum(nil))
	return nil
}

// Error is a typed source-planning failure.
type Error struct{ Reason string }

func (e *Error) Error() string { return "GOTOTS_SOURCE_PLAN: " + e.Reason }
