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

// ConstShapeSignature is the canonical join string of a constant's
// declaration shape — its canonical type and exact value. It is the ONE
// definition of the constant-parity contract: the census gate hashes it
// over its recorded shapes and the translator hashes it over the constant
// it lowered, so a divergence in either the type or the value is a detected
// defect. The NUL separator cannot appear in a type spelling or an exact
// constant value, so distinct (type, value) pairs never alias.
func ConstShapeSignature(s ConstShape) string {
	return s.Type + "\x00" + s.Value
}

// TypeShapeSignature is the canonical join string of a named type's
// complete declaration shape: kind, underlying, type parameters, struct
// fields (name/type/tag/embedded), interface methods and embeds, and
// declared methods. It is the ONE definition of the named-type parity
// contract: the census gate hashes it over its recorded shapes and the
// translator hashes it over the type it lowered, so any drift in a
// field, tag, method signature, or embed is a detected defect. The NUL
// and unit separators cannot appear in canonical type spellings, names,
// or tags rendered by strconv-quoted framing below.
func TypeShapeSignature(s TypeShape) string {
	var b []byte
	app := func(parts ...string) {
		for _, p := range parts {
			b = append(b, p...)
			b = append(b, 0x00)
		}
		b = append(b, 0x1f)
	}
	app("kind", s.Kind)
	app("underlying", s.Underlying)
	for _, tp := range s.TypeParams {
		app("typeparam", tp.Name, tp.Constraint)
	}
	for _, f := range s.Fields {
		embedded, exported := "0", "0"
		if f.Embedded {
			embedded = "1"
		}
		if f.Exported {
			exported = "1"
		}
		app("field", f.Name, f.Type, f.Tag, embedded, exported)
	}
	for _, m := range s.InterfaceMethods {
		app("ifacemethod", m.Name, m.Signature)
	}
	for _, e := range s.InterfaceEmbeds {
		app("ifaceembed", e)
	}
	for _, m := range s.Methods {
		ptr := "0"
		if m.PointerReceiver {
			ptr = "1"
		}
		app("method", m.Name, m.Signature, ptr)
	}
	return string(b)
}

// AliasShapeSignature is the canonical join string of a type alias's
// declaration shape: its transparent target and its own type parameters.
func AliasShapeSignature(s AliasShape) string {
	var b []byte
	app := func(parts ...string) {
		for _, p := range parts {
			b = append(b, p...)
			b = append(b, 0x00)
		}
		b = append(b, 0x1f)
	}
	app("target", s.Target)
	for _, tp := range s.TypeParams {
		app("typeparam", tp.Name, tp.Constraint)
	}
	return string(b)
}

// VarShape is one package-level variable.
type VarShape struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// InitializerHash is the sha256 of the initializer expression's exact
	// source bytes when an initializer exists — the identity-bearing
	// evidence the specification requires for initializer bodies.
	InitializerHash string `json:"initializerHash,omitempty"`
}

// FuncLitShape is one function literal: an independent implementation
// unit with canonical identity (position-qualified inside its parent
// declaration) and exact body hash.
type FuncLitShape struct {
	ID       string `json:"id"`
	Parent   string `json:"parent"` // the enclosing declaration's ID
	BodyHash string `json:"bodyHash"`
}

// AliasShape is one type alias and its target.
// AliasShape is one type alias's DECLARATION evidence. ID is the alias's
// own declaration identity (owner package + name). Target is the
// transparent type it denotes (the use-site domain: two aliases with equal
// targets share it). TypeParams carries the alias's own type parameters and
// their constraints — the DECLARATION domain — so two generic aliases with
// the same target but different parameters/constraints stay distinct even
// though their instantiated use-site targets coincide.
type AliasShape struct {
	ID         string           `json:"id"`
	Target     string           `json:"target"`
	TypeParams []TypeParamShape `json:"typeParams,omitempty"`
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
	// FunctionLiterals are the independent function-literal units of
	// production scope, each with canonical identity and body hash.
	FunctionLiterals []FuncLitShape `json:"functionLiterals"`
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
