package tsgo

type SyntaxKind uint32

type NodeFlags uint32

type TokenFlags uint32

type LanguageVariant uint32

type ScriptKind uint32

type Factory struct{}

func NewFactory() Factory {
	return Factory{}
}

type Node interface {
	Kind() SyntaxKind
	Pos() int32
	End() int32
	Flags() NodeFlags
	targetEncoding() nodeEncoding
}

type nodeCore struct {
	kind  SyntaxKind
	pos   int32
	end   int32
	flags NodeFlags
}

func newNodeCore(kind SyntaxKind, flags NodeFlags) nodeCore {
	return nodeCore{kind: kind, pos: -1, end: -1, flags: flags}
}

func (n nodeCore) Kind() SyntaxKind {
	return n.kind
}

func (n nodeCore) Pos() int32 {
	return n.pos
}

func (n nodeCore) End() int32 {
	return n.end
}

func (n nodeCore) Flags() NodeFlags {
	return n.flags
}

type nodeDataType uint8

const (
	nodeDataChildren nodeDataType = iota
	nodeDataString
	nodeDataExtended
)

type childEncoding struct {
	name     string
	present  bool
	required bool
	raw      bool
	node     Node
	nodes    []Node
}

type extendedKind uint8

const (
	extendedNone extendedKind = iota
	extendedLiteral
	extendedTemplate
	extendedSourceFile
	extendedUnsupported
)

type nodeEncoding struct {
	dataType   nodeDataType
	commonData uint32
	children   []childEncoding
	text       string
	extended   extendedKind
	tokenFlags TokenFlags
	rawText    string
	sourceFile SourceFileData
}

type FileReference struct {
	Pos            uint32
	End            uint32
	FileName       string
	ResolutionMode uint32
	Preserve       bool
}

func cloneSourceFileData(value SourceFileData) SourceFileData {
	value.ReferencedFiles = cloneSlice(value.ReferencedFiles)
	value.TypeReferenceDirectives = cloneSlice(value.TypeReferenceDirectives)
	value.LibReferenceDirectives = cloneSlice(value.LibReferenceDirectives)
	value.Imports = cloneSlice(value.Imports)
	value.ModuleAugmentations = cloneSlice(value.ModuleAugmentations)
	value.AmbientModuleNames = cloneSlice(value.AmbientModuleNames)
	return value
}

func nodesOf[Value Node](values []Value) []Node {
	if values == nil {
		return nil
	}
	result := make([]Node, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

func cloneSlice[Value any](values []Value) []Value {
	if values == nil {
		return nil
	}
	result := make([]Value, len(values))
	copy(result, values)
	return result
}

func boolBit(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}
