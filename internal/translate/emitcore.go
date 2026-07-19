// Core-module emission for one retained package: the module environment
// is built after the whole-unit withholding closure is known, so no
// retained output references a withheld class or module.
package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/ir"
)

// emitCorePackage renders one retained package's module.
func emitCorePackage(out *Generated, p *packages.Package, sourceDir string, unit ir.Scope, options Options,
	corePath string, files []fileSource, functions []*ir.Func, structs map[string]*ir.Struct, structOrder []string,
	carrierMethods []emit.Method, carrierTypes []emit.CarrierType, packageVars []emit.PackageVar, initCalls []string) error {
	_ = files
	structList := make([]*ir.Struct, 0, len(structOrder))
	for _, name := range structOrder {
		structList = append(structList, structs[name])
	}
	// Synthesized anonymous-struct classes the package's bodies use.
	structList = append(structList, unit.AnonStructs(p.PkgPath)...)
	// The module's import environment is built after the declaration
	// passes so external obligations discovered while building bodies
	// resolve to their stub modules.
	module, err := newModule(corePath, p.PkgPath, p.Types.Name(), unit, p, sourceDir, out.NotMaterialized)
	if err != nil {
		return err
	}
	// Every co-generated package this package imports is an
	// initialization edge: its module must evaluate even when constant
	// folding or type-only use erased every symbol reference.
	for importPath := range p.Imports {
		if unit.Owns(importPath) {
			module.RequireInitEdge(importPath)
		}
	}
	body, outcomes, artifacts, err := emit.Package(module, emit.Decls{
		InitCalls:    initCalls,
		Structs:      structList,
		Methods:      carrierMethods,
		CarrierTypes: carrierTypes,
		Vars:         packageVars,
		Functions:    functions,
	})
	if err != nil {
		// A declaration-level emission failure (struct/carrier/var/alias):
		// the package cannot produce analyzable TypeScript. Record the
		// defect — the acceptance gates hard-fail on it — and withhold the
		// package honestly instead of aborting the whole corpus.
		out.EmitterDefects = append(out.EmitterDefects, EmitterDefect{Package: p.PkgPath, ID: p.PkgPath, Err: err.Error()})
		reason := "emitter defect (declaration-level)"
		out.NotMaterialized[p.PkgPath] = reason
		out.Withheld[p.PkgPath] = reason
		return nil
	}
	applyBodyOutcomes(out, p.PkgPath, outcomes)
	// Retain every lowered body's canonical analysis artifact (spec 01):
	// the exact fragment and its hash, withheld packages included. The
	// .ts.txt extension keeps it out of every module and typecheck path.
	hashes := make(map[string]string, len(artifacts))
	for _, artifact := range artifacts {
		digest := sha256.Sum256([]byte(artifact.Text))
		hash := hex.EncodeToString(digest[:])
		hashes[artifact.ID] = hash
		artifactPath := "analysis/bodies/" + p.PkgPath + "/" + sanitizeArtifactName(artifact.ID) + ".ts.txt"
		out.Files[artifactPath] = artifact.Text
		out.Ownership[artifactPath] = "analysis-body"
	}
	for i := range out.Proofs {
		if out.Proofs[i].Package != p.PkgPath {
			continue
		}
		if hash, has := hashes[out.Proofs[i].ID]; has {
			out.Proofs[i].LoweredHash = hash
		}
	}
	coreContent, err := emit.FileWithProvenance(emit.Provenance{
		SchemaVersion:  1,
		SourceRevision: options.SourceRevision,
		ProfileHash:    options.ProfileHash,
		AbiVersion:     abi.Version,
		Path:           corePath,
	}, body)
	if err != nil {
		return err
	}
	out.Files[corePath] = coreContent
	out.Ownership[corePath] = "generated-core"
	if out.ModuleImports == nil {
		out.ModuleImports = map[string][]string{}
	}
	if out.ModuleTypeImports == nil {
		out.ModuleTypeImports = map[string][]string{}
	}
	out.ModuleImports[p.PkgPath] = module.RuntimeImports()
	out.ModuleTypeImports[p.PkgPath] = module.TypeOnlyImports()
	return nil
}

// applyBodyOutcomes folds the emitter's typed per-body outcomes into the
// evidence artifacts. A KnownUnsupported body reclassifies honestly from
// IR-admitted to unimplemented (its typed site on record) and withholds
// the package's runnable module — it materialized as a typed throwing
// placeholder. An EmitterDefect is a compiler defect: the body carries a
// diagnostic placeholder so siblings stay analyzable, the exact identity
// is recorded, and the acceptance gates hard-fail while any defect exists.
func applyBodyOutcomes(out *Generated, pkgPath string, outcomes []emit.BodyOutcome) {
	for _, outcome := range outcomes {
		switch outcome.Kind {
		case emit.OutcomeKnownUnsupported:
			for i := range out.Support {
				if out.Support[i].ID != outcome.ID {
					continue
				}
				out.Support[i].State = ir.SupportUnimplemented
				if outcome.Site != nil {
					out.Support[i].Sites = append(out.Support[i].Sites, *outcome.Site)
				}
				break
			}
			if _, withheld := out.Withheld[pkgPath]; !withheld {
				out.Withheld[pkgPath] = "unimplemented body reclassified at emission (materialized as placeholder)"
			}
		case emit.OutcomeEmitterDefect:
			out.EmitterDefects = append(out.EmitterDefects, EmitterDefect{Package: pkgPath, ID: outcome.ID, Err: outcome.Err})
			if _, withheld := out.Withheld[pkgPath]; !withheld {
				out.Withheld[pkgPath] = "emitter defect (diagnostic placeholder; compiler must be fixed)"
			}
		}
	}
}

// sanitizeArtifactName spells one body identity as a file name: every
// path-hostile rune folds to "_" (identities keep uniqueness through the
// containing package directory plus the sanitized tail).
func sanitizeArtifactName(id string) string {
	out := make([]rune, 0, len(id))
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}
