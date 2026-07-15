package census

// Declaration shapes are the exact typed contract of every declaration in
// the owned corpus: the seed of the declaration parity gate. All type
// spellings are canonical types.TypeString renderings with package-path
// qualifiers, so they are deterministic and machine-comparable across
// pins.

// Param is one parameter or result.
type Param struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

// TypeParamShape is one type parameter with its constraint.
type TypeParamShape struct {
	Name       string `json:"name"`
	Constraint string `json:"constraint"`
}

// FunctionShape is the exact signature of a function or method.
type FunctionShape struct {
	ID         string           `json:"id"` // matches DeclarationRecord.ID
	Receiver   string           `json:"receiver,omitempty"`
	TypeParams []TypeParamShape `json:"typeParams,omitempty"`
	Params     []Param          `json:"params,omitempty"`
	Results    []Param          `json:"results,omitempty"`
	Variadic   bool             `json:"variadic,omitempty"`
	Signature  string           `json:"signature"`
}

// FieldShape is one struct field.
type FieldShape struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Tag      string `json:"tag,omitempty"`
	Embedded bool   `json:"embedded,omitempty"`
	Exported bool   `json:"exported"`
}

// MethodShape is one declared or interface method.
type MethodShape struct {
	Name            string `json:"name"`
	Signature       string `json:"signature"`
	PointerReceiver bool   `json:"pointerReceiver,omitempty"`
}

// TypeShape is the exact contract of a named type.
type TypeShape struct {
	ID               string           `json:"id"`
	Kind             string           `json:"kind"`
	Underlying       string           `json:"underlying"`
	TypeParams       []TypeParamShape `json:"typeParams,omitempty"`
	Fields           []FieldShape     `json:"fields,omitempty"`           // structs
	InterfaceMethods []MethodShape    `json:"interfaceMethods,omitempty"` // interfaces
	InterfaceEmbeds  []string         `json:"interfaceEmbeds,omitempty"`
	Methods          []MethodShape    `json:"methods,omitempty"` // explicit methods on the named type
}

// ConstShape carries the exact go/constant value.
type ConstShape struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// VarShape is one package-level variable.
type VarShape struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// AliasShape is one type alias and its target.
type AliasShape struct {
	ID     string `json:"id"`
	Target string `json:"target"`
}

// DeclarationShapes is the complete typed declaration contract, published
// as declarations.json in the census bundle.
type DeclarationShapes struct {
	SchemaVersion int             `json:"schemaVersion"`
	Functions     []FunctionShape `json:"functions"`
	Types         []TypeShape     `json:"types"`
	Constants     []ConstShape    `json:"constants"`
	Variables     []VarShape      `json:"variables"`
	Aliases       []AliasShape    `json:"aliases"`
}

// ExternalObligation is one external object referenced by owned source: a
// deterministic declaration/stub obligation for the emulation layer.
// Production and test uses stay distinct so product stub obligations are
// never inflated by test-only demand. Field obligations are qualified by
// the receiver type they were selected through (Owner.Field), so
// same-named fields of different types keep distinct identities.
type ExternalObligation struct {
	Package        string `json:"package"`
	Name           string `json:"name"`
	Kind           string `json:"kind"` // func|method|var|const|type|field
	ProductionUses int    `json:"productionUses"`
	TestUses       int    `json:"testUses"`
}

// TestFunctionRecord is one test-discovery function in owned test scope,
// classified by the toolchain's discovery contract (name pattern plus
// exact signature evidence).
type TestFunctionRecord struct {
	ID   string `json:"id"`
	Kind string `json:"kind"` // test|benchmark|fuzz|example
	File string `json:"file"`
	Line int    `json:"line"`
}
