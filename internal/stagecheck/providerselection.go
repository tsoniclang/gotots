package stagecheck

import (
	"crypto/sha256"
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

// VerifyProviderSelection independently proves that every certified package
// selected by an ordinary source plan was produced from the exact current
// package-local inputs. Provider artifacts may contain a superset, so
// unrelated application inputs neither authorize nor invalidate a reusable
// provider package.
func VerifyProviderSelection(
	universe *source.Universe,
	plan *sourceplan.Plan,
	artifact *structure.ProviderArtifact,
) error {
	if universe == nil || plan == nil {
		return providerSelectionError("source universe or plan is absent")
	}
	if plan.Purpose() != sourceplan.PurposeCompilation {
		return providerSelectionError("source plan is not for compilation")
	}
	packages := map[identity.PackageID]*source.LoadedPackage{}
	filePackages := map[identity.FileID]*source.LoadedPackage{}
	for _, pkg := range universe.Packages() {
		packages[pkg.ID()] = pkg
		for _, file := range pkg.Files() {
			filePackages[file.ID()] = pkg
		}
	}
	required := map[identity.PackageID]*source.LoadedPackage{}
	for _, decision := range plan.Files() {
		if decision.Kind() != sourceplan.KindCertifiedGraph {
			continue
		}
		pkg := filePackages[decision.ID()]
		if pkg == nil {
			return providerSelectionError(
				"certified file has no resolved package " +
					decision.ID().String(),
			)
		}
		required[pkg.ID()] = pkg
	}
	for _, decision := range plan.SyntheticOwners() {
		if decision.Kind() != sourceplan.KindCertifiedGraph {
			continue
		}
		pkg := packages[decision.Package()]
		if pkg == nil {
			return providerSelectionError(
				"certified synthetic owner has no resolved package " +
					decision.Package().String(),
			)
		}
		required[pkg.ID()] = pkg
	}
	if len(required) == 0 {
		if artifact != nil {
			return nil
		}
		return nil
	}
	if artifact == nil {
		return providerSelectionError(
			"certified source decisions have no provider artifact",
		)
	}
	for packageID, pkg := range required {
		actual, present := artifact.PackageInputDigest(packageID)
		if !present {
			return providerSelectionError(
				"artifact has no input context for " + packageID.String(),
			)
		}
		expected := independentProviderInputFingerprint(pkg)
		if actual != expected {
			return providerSelectionError(
				"package input drift for " + packageID.String(),
			)
		}
	}
	return nil
}

func independentProviderInputFingerprint(
	pkg *source.LoadedPackage,
) string {
	hash := sha256.New()
	fmt.Fprintln(hash, "gotots-provider-package/v2")
	fmt.Fprintf(
		hash,
		"package|%s|%d:%s|%d|%d|%s|%t\n",
		pkg.ID(),
		len(pkg.DeclaredName()),
		pkg.DeclaredName(),
		pkg.Provenance(),
		pkg.Disposition(),
		pkg.ModuleGoVersion(),
		pkg.HasCheckedView(),
	)
	for _, imported := range pkg.Imports() {
		fmt.Fprintf(
			hash, "import|%s|%s\n",
			imported.Importer(), imported.Imported(),
		)
	}
	for _, file := range pkg.Files() {
		fmt.Fprintf(
			hash,
			"go-file|%s|%s|%s|%t\n",
			file.ID(),
			file.ByteDigest(),
			file.EffectiveGoVersion(),
			file.CgoOriginal(),
		)
	}
	for _, input := range pkg.Inputs() {
		fmt.Fprintf(
			hash,
			"input|%s|%d|%s|%t\n",
			input.ID(),
			input.Kind(),
			input.ByteDigest(),
			input.Overlaid(),
		)
	}
	for _, pattern := range pkg.EmbedPatterns() {
		fmt.Fprintf(
			hash, "embed-pattern|%d:%s\n",
			len(pattern), pattern,
		)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func providerSelectionError(reason string) error {
	return &VerificationError{
		Stage: "provider-selection", Reason: reason,
	}
}
