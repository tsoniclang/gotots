package census

import (
	"fmt"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/inventory"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/typedload"
)

// Run orchestrates both census passes: toolchain attestation, source
// verification, the whole-universe inventory, the typed owned-source load,
// and the post-load mutation check.
func Run(prof *profile.Profile, sourceDir string, buildProfileName string) (*Result, error) {
	build, err := prof.BuildProfileByName(buildProfileName)
	if err != nil {
		return nil, err
	}

	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, err
	}
	// Resolve symlinks once so every path comparison — go list package
	// directories, file attestation, publication containment — happens in
	// one canonical path space.
	resolvedSource, err := filepath.EvalSymlinks(absSource)
	if err != nil {
		return nil, err
	}
	sourceDir = resolvedSource

	// Toolchain first: the go executable's digest is verified before it is
	// ever executed, and the in-process Go frontend must match the pin.
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		return nil, err
	}
	env := resolved.Environ(goenv.EnvOptions{
		GOOS:       build.GOOS,
		GOARCH:     build.GOARCH,
		GOAMD64:    build.GOAMD64,
		GOARM64:    build.GOARM64,
		CgoEnabled: build.CgoEnabled,
	})

	checkout, err := pinning.VerifySource(prof.Pin, sourceDir)
	if err != nil {
		return nil, err
	}
	verified := checkout.Source
	verified.ToolchainVersion = prof.Pin.Toolchain.Version
	verified.GoExecutableSha256 = prof.Pin.Toolchain.GoExecutableSha256
	verified.GorootSrcDigest = prof.Pin.Toolchain.GorootSrcDigest

	inv, err := inventory.Run(prof, resolved, env, sourceDir, checkout.Tree)
	if err != nil {
		return nil, err
	}

	report := &Report{
		SchemaVersion: 4,
		Product:       prof.Product,
		Profile: ProfileAttestation{
			Hash:                 prof.Hash,
			OwnedRoots:           prof.OwnedRoots,
			TestOnlyRoots:        prof.TestOnlyRoots,
			OutsideUniverseRoots: prof.OutsideUniverseRoots,
			ToolingRoots:         prof.ToolingRoots,
		},
		BuildProfile: *build,
		Pin:          *prof.Pin,
		Source:       verified,
		Production:   newScopeReport(),
		Test:         newScopeReport(),
	}
	fillPartition(inv, report)

	// The predeclared universe of the toolchain actually running in this
	// process must match the reviewed contract before any body evidence is
	// trusted.
	universeNames, err := checkPredeclaredUniverse()
	if err != nil {
		return nil, err
	}
	report.PredeclaredUniverse = universeNames

	loaded, err := typedload.Load(prof, env, sourceDir)
	if err != nil {
		return nil, err
	}
	shapes := &DeclarationShapes{SchemaVersion: 1}
	externals, err := analyze(prof, inv, checkout.Tree, loaded, sourceDir, report, shapes)
	if err != nil {
		return nil, err
	}
	report.Blockers = collectBlockers(report)

	// Extraction must not have mutated the pinned source.
	clean, err := pinning.CheckClean(sourceDir)
	if err != nil {
		return nil, err
	}
	if !clean {
		return nil, fmt.Errorf("source checkout %s is dirty after loading; extraction mutated pinned input", sourceDir)
	}
	report.Source.CleanAfterLoad = true

	return &Result{
		Inventory:   inv,
		Report:      report,
		Shapes:      shapes,
		Externals:   externals,
		Environment: &Environment{SourceDir: sourceDir, Toolchain: resolved, ChildEnv: env},
		sourceDir:   sourceDir,
	}, nil
}

// collectBlockers names every condition that prevents this census from
// being an authoritative-success result. Blockers do not invalidate the
// evidence; they gate downstream generation.
func collectBlockers(report *Report) []string {
	var blockers []string
	for _, edge := range report.Contradictions {
		label := edge.Class
		if edge.Category != "" {
			label += ":" + edge.Category
		}
		blockers = append(blockers, fmt.Sprintf("contradiction [%s] %s -> %s (%s)", edge.Scope, edge.From, edge.To, label))
	}
	for _, directive := range report.Directives {
		if !directive.Known {
			blockers = append(blockers, fmt.Sprintf("unknown directive %s at %s:%d", directive.Directive, directive.File, directive.Line))
		}
	}
	return blockers
}

func fillPartition(inv *inventory.Inventory, report *Report) {
	for _, pkg := range inv.Module {
		switch profile.PackageClass(pkg.Class) {
		case profile.ClassOwned:
			report.Partition.Owned++
		case profile.ClassTestOnly:
			report.Partition.TestOnly++
		case profile.ClassUnselected:
			report.Partition.Unselected++
		}
	}
	for _, pkg := range inv.External {
		if pkg.Standard {
			report.Partition.ExternalStd++
		} else {
			report.Partition.ExternalMod++
		}
		if pkg.ReachableFromProduction || pkg.ReachableFromTest {
			report.Partition.ExternalProductClosure++
		}
	}
}
