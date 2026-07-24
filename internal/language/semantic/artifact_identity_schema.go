package semantic

type wireModuleIdentity struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

type wireOwnerIdentity struct {
	Class  uint8               `json:"class"`
	Module wireModuleReference `json:"module,omitempty"`
}

type wirePackageIdentity struct {
	Owner      wireOwnerReference `json:"owner"`
	ImportPath string             `json:"importPath"`
}

type wireFileIdentity struct {
	Owner wireOwnerReference `json:"owner"`
	Rel   string             `json:"relativePath"`
}

type wireSpanIdentity struct {
	File  wireFileReference `json:"file"`
	Start int               `json:"start"`
	End   int               `json:"end"`
}

type wireOccurrenceIdentity struct {
	Span wireSpanReference `json:"span"`
	Kind uint16            `json:"kind"`
}

type wireDefinitionIdentity struct {
	Kind      uint8                   `json:"kind"`
	Root      wireOccurrenceReference `json:"root,omitempty"`
	Package   wirePackageReference    `json:"package,omitempty"`
	Implicit  uint8                   `json:"implicit,omitempty"`
	Synthetic uint8                   `json:"synthetic,omitempty"`
	Name      string                  `json:"name,omitempty"`
}

type wireTypeIdentity struct {
	Digest string `json:"digest"`
}

type wireDeclarationIdentity struct {
	Form        uint8                   `json:"form"`
	Package     wirePackageReference    `json:"package,omitempty"`
	OwnerType   wireTypeReference       `json:"ownerType,omitempty"`
	MemberPkg   wirePackageReference    `json:"memberPackage,omitempty"`
	Class       uint8                   `json:"class"`
	Name        string                  `json:"name,omitempty"`
	Ordinal     int                     `json:"ordinal,omitempty"`
	Predeclared uint16                  `json:"predeclared,omitempty"`
	Owner       wireOccurrenceReference `json:"owner,omitempty"`
	Occurrence  wireOccurrenceReference `json:"occurrence,omitempty"`
}

type wireBindingIdentity struct {
	Owner       wireOccurrenceReference `json:"owner"`
	Declaration wireOccurrenceReference `json:"declaration,omitempty"`
	Role        uint8                   `json:"role"`
	Ordinal     int                     `json:"ordinal"`
}

type wireOperationIdentity struct {
	Definition wireDefinitionReference `json:"definition"`
	Occurrence wireOccurrenceReference `json:"occurrence,omitempty"`
	Implicit   uint8                   `json:"implicit,omitempty"`
	Ordinal    int                     `json:"ordinal,omitempty"`
}

type wireUnsupportedIdentity struct {
	Definition wireDefinitionReference `json:"definition"`
	Occurrence wireOccurrenceReference `json:"occurrence"`
}

type wireIdentityCounts struct {
	modules      int
	owners       int
	packages     int
	files        int
	spans        int
	occurrences  int
	definitions  int
	types        int
	declarations int
	bindings     int
	operations   int
	unsupported  int
}

func (table packageIdentityTable) wireCounts() wireIdentityCounts {
	return wireIdentityCounts{
		modules:      len(table.modules),
		owners:       len(table.owners),
		packages:     len(table.packages),
		files:        len(table.files),
		spans:        len(table.spans),
		occurrences:  len(table.occurrences),
		definitions:  len(table.definitions),
		types:        len(table.types),
		declarations: len(table.declarations),
		bindings:     len(table.bindings),
		operations:   len(table.operations),
		unsupported:  len(table.unsupported),
	}
}
