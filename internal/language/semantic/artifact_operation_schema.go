package semantic

type wireOperationRecord struct {
	ID            wireOperationReference                      `json:"id"`
	Kind          uint16                                      `json:"kind"`
	Syntax        uint16                                      `json:"syntax,omitempty"`
	Variant       uint16                                      `json:"variant,omitempty"`
	Role          uint16                                      `json:"role,omitempty"`
	Token         uint16                                      `json:"token,omitempty"`
	Mode          uint8                                       `json:"mode"`
	Arity         uint8                                       `json:"arity"`
	Place         uint8                                       `json:"place"`
	ResultType    wireTypeReference                           `json:"resultType,omitempty"`
	ExpectedType  wireTypeReference                           `json:"expectedType,omitempty"`
	Addressable   bool                                        `json:"addressable,omitempty"`
	Assignable    bool                                        `json:"assignable,omitempty"`
	HasOk         bool                                        `json:"hasOk,omitempty"`
	Constant      *wireConstantValue                          `json:"constant,omitempty"`
	Object        *wireObjectReference                        `json:"object,omitempty"`
	Selection     *wireSelection                              `json:"selection,omitempty"`
	Instance      *wireInstance                               `json:"instance,omitempty"`
	Operands      wireReferenceRange[wireOccurrenceReference] `json:"operands"`
	Definitions   wireReferenceRange[wireDefinitionReference] `json:"definitions"`
	Implicit      wireImplicitOperationRange                  `json:"implicit"`
	ControlTarget wireOperationReference                      `json:"controlTarget,omitempty"`
	Label         wireBindingReference                        `json:"label,omitempty"`
}

type wireObjectReference struct {
	Kind        uint8                    `json:"kind"`
	Declaration wireDeclarationReference `json:"declaration,omitempty"`
	Binding     wireBindingReference     `json:"binding,omitempty"`
}

type wireSelection struct {
	Kind     uint8                    `json:"kind"`
	Receiver wireTypeReference        `json:"receiver"`
	Object   wireDeclarationReference `json:"object"`
	Index    wireIntegerRange         `json:"index"`
	Indirect bool                     `json:"indirect,omitempty"`
}

type wireInstance struct {
	Target    wireObjectReference                   `json:"target"`
	Types     wireReferenceRange[wireTypeReference] `json:"types"`
	Signature wireTypeReference                     `json:"signature"`
}

type wireImplicitOperation struct {
	Kind    uint8                   `json:"kind"`
	Site    wireOccurrenceReference `json:"site"`
	Ordinal int                     `json:"ordinal"`
	Source  wireTypeReference       `json:"source,omitempty"`
	Target  wireTypeReference       `json:"target,omitempty"`
}

type wireImplicitOperationRange struct {
	Start  uint64                  `json:"start"`
	Count  uint64                  `json:"count"`
	Values []wireImplicitOperation `json:"values,omitempty"`
}
