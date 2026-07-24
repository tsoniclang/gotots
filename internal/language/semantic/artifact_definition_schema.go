package semantic

type wireDefinitionRecord struct {
	ID       wireDefinitionReference                  `json:"id"`
	Package  wirePackageReference                     `json:"package"`
	Form     uint8                                    `json:"form"`
	Name     string                                   `json:"name,omitempty"`
	Bindings wireReferenceRange[wireBindingReference] `json:"bindings"`
	Payload  wireDefinitionPayload                    `json:"payload"`
}

type wireDefinitionPayload struct {
	Tag         uint8                      `json:"tag"`
	Callable    *wireCallableDefinition    `json:"callable,omitempty"`
	Initializer *wireInitializerDefinition `json:"initializer,omitempty"`
	Bodyless    *wireBodylessDefinition    `json:"bodyless,omitempty"`
	Implicit    *wireImplicitDefinition    `json:"implicit,omitempty"`
	Synthetic   *wireSyntheticDefinition   `json:"synthetic,omitempty"`
}

type wireCallableDefinition struct {
	Declarations wireReferenceRange[wireDeclarationReference] `json:"declarations"`
	Signature    wireTypeReference                            `json:"signature"`
	Receiver     wireBindingReference                         `json:"receiver,omitempty"`
}

type wireInitializerDefinition struct {
	Declarations wireReferenceRange[wireDeclarationReference] `json:"declarations"`
	Entries      wireReferenceRange[wireOccurrenceReference]  `json:"entries"`
}

type wireBodylessDefinition struct {
	Declaration wireDeclarationReference `json:"declaration"`
	Signature   wireTypeReference        `json:"signature"`
	Receiver    wireBindingReference     `json:"receiver,omitempty"`
}

type wireImplicitDefinition struct {
	Operation uint8 `json:"operation"`
}

type wireSyntheticDefinition struct {
	Declaration wireDeclarationReference `json:"declaration"`
	Signature   wireTypeReference        `json:"signature,omitempty"`
}

type wireResolutionRecord struct {
	Occurrence wireOccurrenceReference `json:"occurrence"`
	Owner      wireDefinitionReference `json:"owner,omitempty"`
	Syntax     uint16                  `json:"syntax"`
	Role       uint16                  `json:"role"`
	Variant    uint16                  `json:"variant"`
	Domain     uint8                   `json:"domain"`
	Kind       uint8                   `json:"kind"`
	Payload    wireResolutionPayload   `json:"payload"`
}

type wireResolutionPayload struct {
	Tag                 uint8                              `json:"tag"`
	Structural          *wireStructuralResolution          `json:"structural,omitempty"`
	DefinitionComponent *wireDefinitionComponentResolution `json:"definitionComponent,omitempty"`
	Declaration         *wireDeclarationReferencePayload   `json:"declaration,omitempty"`
	Binding             *wireBindingReferencePayload       `json:"binding,omitempty"`
	Type                *wireTypeReferencePayload          `json:"type,omitempty"`
	Operation           *wireOperationReferencePayload     `json:"operation,omitempty"`
	Unsupported         *wireUnsupportedReferencePayload   `json:"unsupported,omitempty"`
}

type wireStructuralResolution struct {
	Disposition uint8                    `json:"disposition"`
	Declaration wireDeclarationReference `json:"declaration,omitempty"`
	Type        wireTypeReference        `json:"type,omitempty"`
}

type wireDefinitionComponentResolution struct {
	Component  uint8                   `json:"component"`
	Definition wireDefinitionReference `json:"definition"`
}

type wireDeclarationReferencePayload struct {
	Reference wireDeclarationReference `json:"reference"`
}

type wireBindingReferencePayload struct {
	Reference wireBindingReference `json:"reference"`
}

type wireTypeReferencePayload struct {
	Reference wireTypeReference `json:"reference"`
}

type wireOperationReferencePayload struct {
	Reference wireOperationReference `json:"reference"`
}

type wireUnsupportedReferencePayload struct {
	Reference wireUnsupportedReference `json:"reference"`
}
