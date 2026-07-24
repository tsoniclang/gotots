package semantic

type wireConstantValue struct {
	Kind  uint8  `json:"kind"`
	Exact string `json:"exact"`
}

type wireDeclarationRecord struct {
	ID       wireDeclarationReference `json:"id"`
	Package  wirePackageReference     `json:"package"`
	Class    uint8                    `json:"class"`
	Name     string                   `json:"name"`
	Type     wireTypeReference        `json:"type,omitempty"`
	Exported bool                     `json:"exported,omitempty"`
	Constant *wireConstantValue       `json:"constant,omitempty"`
}

type wireBindingRecord struct {
	ID         wireBindingReference                        `json:"id"`
	Package    wirePackageReference                        `json:"package"`
	Definition wireDefinitionReference                     `json:"definition,omitempty"`
	Role       uint8                                       `json:"role"`
	Name       string                                      `json:"name,omitempty"`
	Type       wireTypeReference                           `json:"type,omitempty"`
	Source     wireOccurrenceReference                     `json:"source,omitempty"`
	CapturedBy wireReferenceRange[wireDefinitionReference] `json:"capturedBy"`
}

type wireUnsupportedRecord struct {
	ID       wireUnsupportedReference `json:"id"`
	Reason   uint8                    `json:"reason"`
	Evidence string                   `json:"evidence"`
}
