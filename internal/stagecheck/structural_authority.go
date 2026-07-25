package stagecheck

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

func verifyStructuralAuthorityProjections(
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	selectedPackages map[identity.PackageID]bool,
) (
	map[identity.PackageID]structure.AuthorityProjection,
	error,
) {
	expected := map[identity.PackageID]*source.LoadedPackage{}
	for _, pkg := range universe.Packages() {
		if selectedPackages != nil && !selectedPackages[pkg.ID()] {
			continue
		}
		if pkg.Disposition() != source.DispositionOrdinarySource &&
			pkg.Disposition() != source.DispositionUnsafeIntrinsic {
			continue
		}
		expected[pkg.ID()] = pkg
	}
	projections := graph.AuthorityProjections()
	if len(projections) != len(expected) {
		return nil, &VerificationError{
			Stage:  "definition-graph-authority",
			Reason: "structural authority package count differs",
		}
	}
	out := make(
		map[identity.PackageID]structure.AuthorityProjection,
		len(projections),
	)
	for _, projection := range projections {
		pkg := expected[projection.ID()]
		if pkg == nil {
			return nil, &VerificationError{
				Stage: "definition-graph-authority",
				Reason: "unexpected structural authority package " +
					projection.ID().String(),
			}
		}
		if _, duplicate := out[projection.ID()]; duplicate {
			return nil, &VerificationError{
				Stage: "definition-graph-authority",
				Reason: "duplicate structural authority package " +
					projection.ID().String(),
			}
		}
		local, certified, synthetic, err :=
			expectedStructuralAuthorities(plan, pkg)
		if err != nil {
			return nil, err
		}
		if err := exactFileAuthorityJoin(
			projection.ID(),
			"local",
			projection.LocalFiles(),
			local,
		); err != nil {
			return nil, err
		}
		if err := exactFileAuthorityJoin(
			projection.ID(),
			"certified",
			projection.CertifiedFiles(),
			certified,
		); err != nil {
			return nil, err
		}
		if projection.HasCertifiedSynthetic() != synthetic {
			return nil, &VerificationError{
				Stage: "definition-graph-authority",
				Reason: "synthetic authority differs for " +
					projection.ID().String(),
			}
		}
		out[projection.ID()] = projection
	}
	return out, nil
}

func expectedStructuralAuthorities(
	plan *sourceplan.Plan,
	pkg *source.LoadedPackage,
) (
	map[identity.FileID]bool,
	map[identity.FileID]bool,
	bool,
	error,
) {
	local := map[identity.FileID]bool{}
	certified := map[identity.FileID]bool{}
	for _, file := range pkg.Files() {
		kind := sourceplan.KindLocalSyntax
		if plan != nil {
			decision, present := plan.For(file.ID())
			if !present {
				return nil, nil, false, &VerificationError{
					Stage: "definition-graph-authority",
					Reason: "source plan omits file " +
						file.ID().String(),
				}
			}
			kind = decision.Kind()
		}
		switch kind {
		case sourceplan.KindLocalSyntax:
			local[file.ID()] = true
		case sourceplan.KindCertifiedGraph:
			certified[file.ID()] = true
		default:
			return nil, nil, false, &VerificationError{
				Stage: "definition-graph-authority",
				Reason: "invalid source authority for " +
					file.ID().String(),
			}
		}
	}
	certifiedSynthetic := false
	if plan != nil {
		if synthetic, present := plan.SyntheticFor(pkg.ID()); present {
			certifiedSynthetic =
				synthetic.Kind() == sourceplan.KindCertifiedGraph
		}
	}
	return local, certified, certifiedSynthetic, nil
}

func exactFileAuthorityJoin(
	packageID identity.PackageID,
	class string,
	actual []identity.FileID,
	expected map[identity.FileID]bool,
) error {
	seen := map[identity.FileID]bool{}
	for _, file := range actual {
		if file.IsZero() || seen[file] || !expected[file] {
			return &VerificationError{
				Stage: "definition-graph-authority",
				Reason: class + " file authority differs for " +
					packageID.String() + ": " + file.String(),
			}
		}
		seen[file] = true
	}
	if len(seen) != len(expected) {
		for file := range expected {
			if !seen[file] {
				return &VerificationError{
					Stage: "definition-graph-authority",
					Reason: class + " file authority is absent for " +
						packageID.String() + ": " + file.String(),
				}
			}
		}
	}
	return nil
}
