package semantic

const ProviderArtifactVersion = 1

type providerContext struct {
	Version                  int    `json:"version"`
	ToolchainDigest          string `json:"toolchainDigest"`
	ConfigurationDigest      string `json:"configurationDigest"`
	ContractID               string `json:"contractId"`
	ContractFingerprint      string `json:"contractFingerprint"`
	StructuralArtifactDigest string `json:"structuralArtifactDigest"`
}

type providerManifest struct {
	Context  providerContext           `json:"context"`
	Packages []providerManifestPackage `json:"packages"`
}

type providerManifestPackage struct {
	Package          string   `json:"package"`
	Provenance       uint8    `json:"provenance"`
	PackageInput     string   `json:"packageInputDigest"`
	Structure        string   `json:"structureDigest"`
	Selection        string   `json:"selectionDigest"`
	Definitions      []string `json:"definitions"`
	Declarations     []string `json:"declarations"`
	DefinitionCount  int      `json:"definitionCount"`
	ResolutionCount  int      `json:"resolutionCount"`
	DeclarationCount int      `json:"declarationCount"`
	BindingCount     int      `json:"bindingCount"`
	TypeCount        int      `json:"typeCount"`
	OperationCount   int      `json:"operationCount"`
	UnsupportedCount int      `json:"unsupportedCount"`
	ShardOffset      int64    `json:"shardOffset"`
	ShardBytes       int64    `json:"shardBytes"`
	ShardDigest      string   `json:"shardDigest"`
}

type providerShard struct {
	Version      int               `json:"version"`
	Package      string            `json:"package"`
	Provenance   uint8             `json:"provenance"`
	Definitions  []wireDefinition  `json:"definitions"`
	Resolutions  []wireResolution  `json:"resolutions"`
	Declarations []wireDeclaration `json:"declarations"`
	Bindings     []wireBinding     `json:"bindings"`
	Types        []wireType        `json:"types"`
	Operations   []wireOperation   `json:"operations"`
	Unsupported  []wireUnsupported `json:"unsupported"`
}

type wireDefinition struct {
	Definition         string   `json:"definition"`
	Package            string   `json:"package"`
	Form               uint8    `json:"form"`
	Name               string   `json:"name,omitempty"`
	Declarations       []string `json:"declarations,omitempty"`
	Signature          string   `json:"signature,omitempty"`
	Receiver           string   `json:"receiver,omitempty"`
	Bindings           []string `json:"bindings,omitempty"`
	InitializerEntries []string `json:"initializerEntries,omitempty"`
	Implicit           uint8    `json:"implicit,omitempty"`
}

type wireResolution struct {
	Occurrence  string         `json:"occurrence"`
	Owner       string         `json:"owner"`
	Syntax      uint16         `json:"syntax"`
	Role        uint16         `json:"role"`
	Variant     uint16         `json:"variant"`
	Domain      uint8          `json:"domain"`
	Kind        uint8          `json:"kind"`
	Structural  wireStructural `json:"structural,omitempty"`
	Component   uint8          `json:"component,omitempty"`
	Definition  string         `json:"definition,omitempty"`
	Declaration string         `json:"declaration,omitempty"`
	Binding     string         `json:"binding,omitempty"`
	Type        string         `json:"type,omitempty"`
	Operation   string         `json:"operation,omitempty"`
	Unsupported string         `json:"unsupported,omitempty"`
}

type wireStructural struct {
	Disposition uint8  `json:"disposition,omitempty"`
	Declaration string `json:"declaration,omitempty"`
	Type        string `json:"type,omitempty"`
}

type wireDeclaration struct {
	ID       string       `json:"id"`
	Package  string       `json:"package"`
	Class    uint8        `json:"class"`
	Name     string       `json:"name"`
	Type     string       `json:"type"`
	Source   string       `json:"source,omitempty"`
	Exported bool         `json:"exported,omitempty"`
	Constant wireConstant `json:"constant,omitempty"`
}

type wireConstant struct {
	Kind  uint8  `json:"kind,omitempty"`
	Exact string `json:"exact,omitempty"`
}

type wireBinding struct {
	ID         string   `json:"id"`
	Package    string   `json:"package"`
	Definition string   `json:"definition,omitempty"`
	Role       uint8    `json:"role"`
	Name       string   `json:"name,omitempty"`
	Type       string   `json:"type,omitempty"`
	Source     string   `json:"source,omitempty"`
	CapturedBy []string `json:"capturedBy,omitempty"`
}

type wireType struct {
	ID                   string           `json:"id"`
	Kind                 uint8            `json:"kind"`
	Basic                uint8            `json:"basic,omitempty"`
	Declaration          string           `json:"declaration,omitempty"`
	ParameterDeclaration string           `json:"parameterDeclaration,omitempty"`
	ParameterDefinition  string           `json:"parameterDefinition,omitempty"`
	ParameterRole        uint8            `json:"parameterRole,omitempty"`
	ParameterOrdinal     int              `json:"parameterOrdinal,omitempty"`
	Arguments            []string         `json:"arguments,omitempty"`
	Underlying           string           `json:"underlying,omitempty"`
	Target               string           `json:"target,omitempty"`
	Constraint           string           `json:"constraint,omitempty"`
	Element              string           `json:"element,omitempty"`
	Key                  string           `json:"key,omitempty"`
	Length               int64            `json:"length,omitempty"`
	Direction            uint8            `json:"direction,omitempty"`
	Signature            wireSignature    `json:"signature,omitempty"`
	Fields               []wireTypeField  `json:"fields,omitempty"`
	Methods              []wireTypeMethod `json:"methods,omitempty"`
	Embeddeds            []string         `json:"embeddeds,omitempty"`
	Terms                []wireTypeTerm   `json:"terms,omitempty"`
	TypeSet              uint8            `json:"typeSet,omitempty"`
	Comparable           bool             `json:"comparable,omitempty"`
	Elements             []string         `json:"elements,omitempty"`
}

type wireSignature struct {
	Receiver               string   `json:"receiver,omitempty"`
	ReceiverTypeParameters []string `json:"receiverTypeParameters,omitempty"`
	TypeParameters         []string `json:"typeParameters,omitempty"`
	Parameters             []string `json:"parameters,omitempty"`
	Results                []string `json:"results,omitempty"`
	Variadic               bool     `json:"variadic,omitempty"`
}

type wireTypeField struct {
	Name     string `json:"name"`
	Package  string `json:"package,omitempty"`
	Type     string `json:"type"`
	Embedded bool   `json:"embedded,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Ordinal  int    `json:"ordinal"`
}

type wireTypeMethod struct {
	Name      string `json:"name"`
	Package   string `json:"package,omitempty"`
	Signature string `json:"signature"`
	Ordinal   int    `json:"ordinal"`
}

type wireTypeTerm struct {
	Tilde bool   `json:"tilde,omitempty"`
	Type  string `json:"type"`
}

type wireOperation struct {
	ID            string              `json:"id"`
	Kind          uint16              `json:"kind"`
	Syntax        uint16              `json:"syntax,omitempty"`
	Variant       uint16              `json:"variant,omitempty"`
	Role          uint16              `json:"role,omitempty"`
	Token         uint16              `json:"token,omitempty"`
	Mode          uint8               `json:"mode"`
	Arity         uint8               `json:"arity"`
	Place         uint8               `json:"place"`
	ResultType    string              `json:"resultType,omitempty"`
	ExpectedType  string              `json:"expectedType,omitempty"`
	Addressable   bool                `json:"addressable,omitempty"`
	Assignable    bool                `json:"assignable,omitempty"`
	HasOk         bool                `json:"hasOk,omitempty"`
	Constant      wireConstant        `json:"constant,omitempty"`
	Object        wireObjectReference `json:"object"`
	Selection     wireSelection       `json:"selection,omitempty"`
	Instance      wireInstance        `json:"instance,omitempty"`
	Operands      []string            `json:"operands,omitempty"`
	Definitions   []string            `json:"definitions,omitempty"`
	Implicit      []wireImplicit      `json:"implicit,omitempty"`
	ControlTarget string              `json:"controlTarget,omitempty"`
	Label         string              `json:"label,omitempty"`
}

type wireObjectReference struct {
	Kind        uint8  `json:"kind"`
	Declaration string `json:"declaration,omitempty"`
	Binding     string `json:"binding,omitempty"`
}

type wireSelection struct {
	Kind     uint8  `json:"kind,omitempty"`
	Receiver string `json:"receiver,omitempty"`
	Object   string `json:"object,omitempty"`
	Index    []int  `json:"index,omitempty"`
	Indirect bool   `json:"indirect,omitempty"`
}

type wireInstance struct {
	Target    wireObjectReference `json:"target"`
	Types     []string            `json:"types,omitempty"`
	Signature string              `json:"signature,omitempty"`
}

type wireImplicit struct {
	Kind    uint8  `json:"kind"`
	Site    string `json:"site"`
	Ordinal int    `json:"ordinal"`
	Source  string `json:"source,omitempty"`
	Target  string `json:"target,omitempty"`
}

type wireUnsupported struct {
	ID       string `json:"id"`
	Reason   uint8  `json:"reason"`
	Evidence string `json:"evidence"`
}
