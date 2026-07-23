package semantic

import "fmt"

// PackageProvenance is the target-independent source class resolved by the Go
// toolchain. It is evidence only and never selects lowering behavior.
type PackageProvenance uint8

const (
	ProvenanceInvalid PackageProvenance = iota
	ProvenanceWorkspaceModule
	ProvenanceModuleDependency
	ProvenanceStandardLibrary
	ProvenanceToolchainPackage
	ProvenanceLanguagePseudo
)

func (provenance PackageProvenance) Valid() bool {
	return provenance >= ProvenanceWorkspaceModule &&
		provenance <= ProvenanceLanguagePseudo
}

func (provenance PackageProvenance) String() string {
	switch provenance {
	case ProvenanceWorkspaceModule:
		return "workspace-module"
	case ProvenanceModuleDependency:
		return "module-dependency"
	case ProvenanceStandardLibrary:
		return "standard-library"
	case ProvenanceToolchainPackage:
		return "toolchain-package"
	case ProvenanceLanguagePseudo:
		return "language-pseudo"
	default:
		return fmt.Sprintf(
			"semantic.PackageProvenance(%d)", uint8(provenance),
		)
	}
}
