package stagecheck

import (
	"crypto/sha256"
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

// verifyHydration exact-joins the source plan to the exposed transient
// semantic surface. It proves certified files have no retained bytes or AST
// even when their package had to be type-imported by a local package.
func verifyHydration(
	universe *source.Universe,
	plan *sourceplan.Plan,
) error {
	expectedFiles := map[identity.FileID]bool{}
	expectedSynthetic := map[identity.PackageID]bool{}
	if plan == nil {
		for _, pkg := range universe.Packages() {
			if pkg.Disposition() != source.DispositionOrdinarySource &&
				pkg.Disposition() != source.DispositionUnsafeIntrinsic {
				continue
			}
			for _, file := range pkg.Files() {
				expectedFiles[file.ID()] = true
			}
			if pkg.HasCheckedView() {
				expectedSynthetic[pkg.ID()] = true
			}
		}
	} else {
		for _, file := range plan.Files() {
			if file.Kind() == sourceplan.KindLocalSyntax {
				expectedFiles[file.ID()] = true
			}
		}
		for _, owner := range plan.SyntheticOwners() {
			if owner.Kind() == sourceplan.KindLocalSyntax {
				expectedSynthetic[owner.Package()] = true
			}
		}
	}
	return verifyHydrationExpected(
		universe, expectedFiles, expectedSynthetic,
	)
}

func verifyHydrationPackages(
	universe *source.Universe,
	packageIDs map[identity.PackageID]bool,
) error {
	expectedFiles := map[identity.FileID]bool{}
	expectedSynthetic := map[identity.PackageID]bool{}
	for _, pkg := range universe.Packages() {
		if !packageIDs[pkg.ID()] {
			continue
		}
		for _, file := range pkg.Files() {
			expectedFiles[file.ID()] = true
		}
		if pkg.HasCheckedView() {
			expectedSynthetic[pkg.ID()] = true
		}
	}
	return verifyHydrationExpected(
		universe, expectedFiles, expectedSynthetic,
	)
}

func verifyHydrationExpected(
	universe *source.Universe,
	expectedFiles map[identity.FileID]bool,
	expectedSynthetic map[identity.PackageID]bool,
) error {
	if universe == nil || !universe.Hydrated() {
		return hydrationError("universe is not selectively hydrated")
	}
	if (len(expectedFiles) != 0 || len(expectedSynthetic) != 0) &&
		universe.Fset() == nil {
		return hydrationError(
			"non-empty semantic hydration has no file set",
		)
	}
	expectedChecker := map[identity.PackageID]bool{}
	problems := newProblemSet()
	for _, pkg := range universe.Packages() {
		localFileCount := 0
		for _, file := range pkg.Files() {
			local := expectedFiles[file.ID()]
			if local {
				localFileCount++
				expectedChecker[pkg.ID()] = true
				raw := file.SelectedBytes()
				if len(raw) == 0 {
					problems.add(
						file.ID().String() + " local bytes are absent",
					)
				} else if sha256.Sum256(raw) != file.ByteDigest() {
					problems.add(
						file.ID().String() + " local bytes changed",
					)
				}
				if file.PhysicalSyntax() == nil ||
					file.PhysicalFileSet() == nil {
					problems.add(
						file.ID().String() + " local syntax is absent",
					)
				}
				continue
			}
			if len(file.SelectedBytes()) != 0 ||
				file.PhysicalSyntax() != nil ||
				file.PhysicalFileSet() != nil ||
				file.CheckedSyntax() != nil {
				problems.add(
					file.ID().String() +
						" certified syntax or source bytes remain reachable",
				)
			}
		}
		if expectedSynthetic[pkg.ID()] {
			expectedChecker[pkg.ID()] = true
			if len(pkg.CheckedDeclarations()) == 0 {
				problems.add(
					pkg.ID().String() +
						" local checked view has no declarations",
				)
			}
		} else if len(pkg.CheckedDeclarations()) != 0 {
			problems.add(
				pkg.ID().String() +
					" certified checked-view declarations remain reachable",
			)
		}
		hasChecker := pkg.CheckerView() != nil
		if hasChecker != expectedChecker[pkg.ID()] {
			problems.addf(
				"%s checker-view=%t want %t (local files=%d synthetic=%t)",
				pkg.ID(),
				hasChecker,
				expectedChecker[pkg.ID()],
				localFileCount,
				expectedSynthetic[pkg.ID()],
			)
		}
		if expectedChecker[pkg.ID()] && pkg.Types() == nil {
			problems.add(
				pkg.ID().String() + " local package lacks type graph",
			)
		}
	}
	if err := problems.verificationError(
		"semantic-hydration",
		"plan-to-hydration exact join failed",
	); err != nil {
		return err
	}
	stats := universe.HydrationStats()
	if stats.LocalFiles != len(expectedFiles) {
		return hydrationError(
			"hydration reports %d local files, want %d",
			stats.LocalFiles, len(expectedFiles),
		)
	}
	if stats.CheckedPackages != len(expectedSynthetic) {
		return hydrationError(
			"hydration reports %d checked packages, want %d",
			stats.CheckedPackages, len(expectedSynthetic),
		)
	}
	return nil
}

func hydrationError(format string, arguments ...any) error {
	return &VerificationError{
		Stage:  "semantic-hydration",
		Reason: fmt.Sprintf(format, arguments...),
	}
}
