package stagecheck

import (
	"crypto/sha256"
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

type expectedSourceDecision struct {
	kind           sourceplan.Kind
	contractDigest string
	artifactDigest string
}

func verifySourcePlan(
	req source.Request,
	universe *source.Universe,
	plan *sourceplan.Plan,
	selected contract.Contract,
	certified *structure.ProviderArtifact,
) error {
	if plan == nil ||
		plan.Purpose() != sourceplan.PurposeCompilation {
		return &VerificationError{
			Stage:  "structural-source-plan",
			Reason: "ordinary compilation lacks a compilation source plan",
		}
	}
	expectedFiles := map[identity.FileID]expectedSourceDecision{}
	expectedSynthetic := map[identity.PackageID]expectedSourceDecision{}
	certifiedFiles := map[string]bool{}
	certifiedPackages := map[string]bool{}
	if certified != nil {
		certifiedFiles = certified.FileIDs()
		certifiedPackages = certified.PackageIDs()
	}
	for _, pkg := range universe.Packages() {
		if pkg.Disposition() != source.DispositionOrdinarySource &&
			pkg.Disposition() != source.DispositionUnsafeIntrinsic {
			continue
		}
		packageLocal := independentPackageMayTranslate(pkg, selected)
		localCgo := false
		for _, file := range pkg.Files() {
			local := pkg.Disposition() ==
				source.DispositionUnsafeIntrinsic ||
				packageLocal ||
				independentExactMayTranslate(
					file.ID(), pkg, selected,
				)
			decision := expectedSourceDecision{
				kind:           sourceplan.KindLocalSyntax,
				contractDigest: selected.Fingerprint(),
			}
			if !local {
				decision.kind = sourceplan.KindCertifiedGraph
				decision.artifactDigest = req.AuditArtifactDigest
				if certified == nil ||
					!certifiedFiles[file.ID().String()] ||
					req.AuditArtifactDigest == "" {
					return &VerificationError{
						Stage: "structural-source-plan",
						Reason: "expected certified file is unavailable " +
							file.ID().String(),
					}
				}
			}
			expectedFiles[file.ID()] = decision
			localCgo = localCgo ||
				(file.CgoOriginal() && local)
		}
		if pkg.HasCheckedView() {
			decision := expectedSourceDecision{
				kind:           sourceplan.KindLocalSyntax,
				contractDigest: selected.Fingerprint(),
			}
			if !localCgo {
				decision.kind = sourceplan.KindCertifiedGraph
				decision.artifactDigest = req.AuditArtifactDigest
				if certified == nil ||
					!certifiedPackages[pkg.ID().String()] ||
					req.AuditArtifactDigest == "" {
					return &VerificationError{
						Stage: "structural-source-plan",
						Reason: "expected certified synthetic owner is unavailable " +
							pkg.ID().String(),
					}
				}
			}
			expectedSynthetic[pkg.ID()] = decision
		}
	}
	actualFiles := map[identity.FileID]sourceplan.File{}
	for _, file := range plan.Files() {
		if _, duplicate := actualFiles[file.ID()]; duplicate {
			return sourcePlanError("duplicate file " + file.ID().String())
		}
		actualFiles[file.ID()] = file
		indexed, present := plan.For(file.ID())
		if !present || indexed != file {
			return sourcePlanError(
				"file index disagrees at " + file.ID().String(),
			)
		}
	}
	actualSynthetic := map[identity.PackageID]sourceplan.SyntheticOwner{}
	for _, owner := range plan.SyntheticOwners() {
		if _, duplicate := actualSynthetic[owner.Package()]; duplicate {
			return sourcePlanError(
				"duplicate synthetic owner " + owner.Package().String(),
			)
		}
		actualSynthetic[owner.Package()] = owner
		indexed, present := plan.SyntheticFor(owner.Package())
		if !present || indexed != owner {
			return sourcePlanError(
				"synthetic index disagrees at " + owner.Package().String(),
			)
		}
	}
	problems := newProblemSet()
	for id, expected := range expectedFiles {
		actual, present := actualFiles[id]
		if !present {
			problems.add("missing file " + id.String())
			continue
		}
		if actual.Kind() != expected.kind ||
			actual.ContractDigest() != expected.contractDigest ||
			actual.ArtifactDigest() != expected.artifactDigest {
			problems.add("file decision mismatch " + id.String())
		}
	}
	for id := range actualFiles {
		if _, present := expectedFiles[id]; !present {
			problems.add("unexpected file " + id.String())
		}
	}
	for id, expected := range expectedSynthetic {
		actual, present := actualSynthetic[id]
		if !present {
			problems.add("missing synthetic " + id.String())
			continue
		}
		if actual.Kind() != expected.kind ||
			actual.ContractDigest() != expected.contractDigest ||
			actual.ArtifactDigest() != expected.artifactDigest {
			problems.add(
				"synthetic decision mismatch " + id.String(),
			)
		}
	}
	for id := range actualSynthetic {
		if _, present := expectedSynthetic[id]; !present {
			problems.add("unexpected synthetic " + id.String())
		}
	}
	if plan.Fingerprint() != independentPlanFingerprint(plan) {
		problems.add("plan fingerprint mismatch")
	}
	if !problems.empty() {
		return sourcePlanError(
			problems.summary("plan exact join failed"),
		)
	}
	return nil
}

func sourcePlanError(reason string) error {
	return &VerificationError{
		Stage: "structural-source-plan", Reason: reason,
	}
}

func independentPlanFingerprint(plan *sourceplan.Plan) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "purpose:%s\n", plan.Purpose())
	for _, file := range plan.Files() {
		fmt.Fprintf(
			hash,
			"%s|%s|%s|%s\n",
			file.ID(),
			file.Kind(),
			file.ContractDigest(),
			file.ArtifactDigest(),
		)
	}
	for _, owner := range plan.SyntheticOwners() {
		fmt.Fprintf(
			hash,
			"synthetic:%s|%s|%s|%s\n",
			owner.Package(),
			owner.Kind(),
			owner.ContractDigest(),
			owner.ArtifactDigest(),
		)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
