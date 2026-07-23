package identity

import "fmt"

// OwnerClass is the closed class of a package owner. Standard-library and
// toolchain packages have reserved owner identities — never a fabricated
// module and never a machine path.
type OwnerClass uint8

const (
	OwnerInvalid OwnerClass = iota
	// OwnerModule: a Go module identified by declared path and version.
	OwnerModule
	// OwnerStandardLibrary: the selected toolchain's `go list std` set.
	OwnerStandardLibrary
	// OwnerToolchain: selected-toolchain command packages (`go list cmd`).
	OwnerToolchain
	// OwnerLanguagePseudo: language pseudo-packages (builtin, import "C").
	OwnerLanguagePseudo

	numOwnerClasses
)

var ownerClassNames = [numOwnerClasses]string{
	OwnerModule:          "module",
	OwnerStandardLibrary: "standard-library",
	OwnerToolchain:       "toolchain",
	OwnerLanguagePseudo:  "language-pseudo",
}

// Valid reports whether c names an owner class.
func (c OwnerClass) Valid() bool { return c > OwnerInvalid && c < numOwnerClasses }

// String renders c for reports.
func (c OwnerClass) String() string {
	if c.Valid() {
		return ownerClassNames[c]
	}
	return fmt.Sprintf("identity.OwnerClass(%d)", uint8(c))
}

// Owner is the semantic owner of a package or file: a module identity, or one
// of the reserved standard-library / toolchain / language-pseudo owners.
type Owner struct {
	class  OwnerClass
	module ModuleID
}

// NewModuleOwner validates a module identity into an owner.
func NewModuleOwner(module ModuleID) (Owner, error) {
	if module.IsZero() {
		return Owner{}, &Error{Identity: "owner", Value: "", Reason: "module owner requires a non-zero module identity"}
	}
	return Owner{class: OwnerModule, module: module}, nil
}

// StandardLibraryOwner is the reserved owner of `go list std` packages.
func StandardLibraryOwner() Owner { return Owner{class: OwnerStandardLibrary} }

// ToolchainOwner is the reserved owner of selected-toolchain command packages.
func ToolchainOwner() Owner { return Owner{class: OwnerToolchain} }

// LanguagePseudoOwner is the reserved owner of language pseudo-packages.
func LanguagePseudoOwner() Owner { return Owner{class: OwnerLanguagePseudo} }

// IsZero reports whether o is the invalid zero owner.
func (o Owner) IsZero() bool { return o == Owner{} }

// Class is the owner class.
func (o Owner) Class() OwnerClass { return o.class }

// Module is the owning module identity; zero unless Class is OwnerModule.
func (o Owner) Module() ModuleID { return o.module }

// String is the canonical serialization. The "mod=" prefix keeps module paths
// (including a module literally named "std") unambiguous against the reserved
// owners.
func (o Owner) String() string {
	switch o.class {
	case OwnerModule:
		return "mod=" + o.module.String()
	case OwnerStandardLibrary:
		return "std"
	case OwnerToolchain:
		return "toolchain"
	case OwnerLanguagePseudo:
		return "lang"
	}
	return "invalid-owner"
}
