package source

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// CensusMode is the closed per-file unit-acquisition mode. It is derived from
// the request-selected provider contract BEFORE the universe loads — owner
// class is an input to the contract's rules, never the policy itself.
type CensusMode uint8

const (
	CensusModeInvalid CensusMode = iota
	// CensusRecursive: the file's interior units (nested function literals)
	// derive from a local recursive syntax walk.
	CensusRecursive
	// CensusManifest: the file receives a bounded top-level census; interior
	// units join from the request's verified content-addressed unit manifest,
	// and the manifest's top-level records must exact-join the local census.
	CensusManifest

	numCensusModes
)

var censusModeNames = [numCensusModes]string{
	CensusRecursive: "recursive", CensusManifest: "manifest",
}

// Valid reports whether m names a census mode.
func (m CensusMode) Valid() bool { return m > CensusModeInvalid && m < numCensusModes }

// String renders m for reports.
func (m CensusMode) String() string {
	if m.Valid() {
		return censusModeNames[m]
	}
	return fmt.Sprintf("source.CensusMode(%d)", uint8(m))
}

// AcquisitionPolicy is the contract-derived unit-acquisition policy: census
// modes by declared owner namespace with exact-package overrides. It is pure
// data resolved before any census decision; a package no rule covers fails
// closed.
type AcquisitionPolicy struct {
	byClass   map[identity.OwnerClass]CensusMode
	byPackage map[string]CensusMode // canonical PackageID serialization
}

// NewAcquisitionPolicy validates one acquisition policy.
func NewAcquisitionPolicy(byClass map[identity.OwnerClass]CensusMode, byPackage map[string]CensusMode) (AcquisitionPolicy, error) {
	out := AcquisitionPolicy{
		byClass:   make(map[identity.OwnerClass]CensusMode, len(byClass)),
		byPackage: make(map[string]CensusMode, len(byPackage)),
	}
	for class, mode := range byClass {
		if !class.Valid() {
			return AcquisitionPolicy{}, &LoadError{Reason: "acquisition policy names an invalid owner class"}
		}
		if !mode.Valid() {
			return AcquisitionPolicy{}, &LoadError{Reason: "acquisition policy for class " + class.String() + " has an invalid census mode"}
		}
		out.byClass[class] = mode
	}
	for pkg, mode := range byPackage {
		if pkg == "" {
			return AcquisitionPolicy{}, &LoadError{Reason: "acquisition policy names an empty package identity"}
		}
		if !mode.Valid() {
			return AcquisitionPolicy{}, &LoadError{Reason: "acquisition policy for package " + pkg + " has an invalid census mode"}
		}
		out.byPackage[pkg] = mode
	}
	return out, nil
}

// ModeFor answers the census mode of one package: the exact-package override
// when declared, otherwise the owner-namespace mode; a package the policy
// does not cover is a typed failure, never a default.
func (p AcquisitionPolicy) ModeFor(pkg identity.PackageID) (CensusMode, error) {
	if mode, exact := p.byPackage[pkg.String()]; exact {
		return mode, nil
	}
	if mode, declared := p.byClass[pkg.Owner().Class()]; declared {
		return mode, nil
	}
	return CensusModeInvalid, &LoadError{Reason: "acquisition policy covers no rule for package " + pkg.String() +
		" (owner class " + pkg.Owner().Class().String() + ")"}
}
