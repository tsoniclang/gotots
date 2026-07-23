// Package semantic owns the immutable, target-independent Go semantic schema.
// It contains no go/ast, go/types, source-loader, compiler, or target imports.
package semantic

import "fmt"

type AuthorityKind uint8

const (
	AuthorityInvalid AuthorityKind = iota
	AuthorityChecker
	AuthorityCertifiedProvider
)

func (kind AuthorityKind) Valid() bool {
	return kind == AuthorityChecker ||
		kind == AuthorityCertifiedProvider
}

type ResolutionKind uint8

const (
	ResolutionInvalid ResolutionKind = iota
	ResolutionStructuralOnly
	ResolutionDefinitionComponent
	ResolutionDeclaration
	ResolutionBinding
	ResolutionType
	ResolutionOperation
	ResolutionUnsupported
)

func (kind ResolutionKind) Valid() bool {
	return kind >= ResolutionStructuralOnly &&
		kind <= ResolutionUnsupported
}

func (kind ResolutionKind) String() string {
	switch kind {
	case ResolutionStructuralOnly:
		return "structural-only"
	case ResolutionDefinitionComponent:
		return "definition-component"
	case ResolutionDeclaration:
		return "declaration"
	case ResolutionBinding:
		return "binding"
	case ResolutionType:
		return "type"
	case ResolutionOperation:
		return "operation"
	case ResolutionUnsupported:
		return "unsupported"
	default:
		return fmt.Sprintf(
			"semantic.ResolutionKind(%d)", uint8(kind),
		)
	}
}

type StructuralDisposition uint8

const (
	StructuralInvalid StructuralDisposition = iota
	StructuralDocumentation
	StructuralContainer
	StructuralPackageClause
	StructuralImportPath
	StructuralBlankIdentifier
	StructuralDeclarationEnvelope
	StructuralDefinitionReference
)

func (kind StructuralDisposition) Valid() bool {
	return kind >= StructuralDocumentation &&
		kind <= StructuralDefinitionReference
}

type DefinitionComponentKind uint8

const (
	DefinitionComponentInvalid DefinitionComponentKind = iota
	DefinitionComponentRoot
	DefinitionComponentName
	DefinitionComponentReceiver
	DefinitionComponentSignature
	DefinitionComponentBoundary
	DefinitionComponentInitializer
	DefinitionComponentBodyless
	DefinitionComponentImplicit
)

func (kind DefinitionComponentKind) Valid() bool {
	return kind >= DefinitionComponentRoot &&
		kind <= DefinitionComponentImplicit
}

type DefinitionForm uint8

const (
	DefinitionFormInvalid DefinitionForm = iota
	DefinitionFormCallable
	DefinitionFormInitializer
	DefinitionFormBodyless
	DefinitionFormImplicit
	DefinitionFormExternal
	DefinitionFormIntrinsic
)

func (kind DefinitionForm) Valid() bool {
	return kind >= DefinitionFormCallable &&
		kind <= DefinitionFormIntrinsic
}

type ValueMode uint8

const (
	ValueModeInvalid ValueMode = iota
	ValueModeNone
	ValueModeValue
	ValueModeType
	ValueModeBuiltin
	ValueModeNil
	ValueModeVoid
	ValueModeTuple
	ValueModePackage
	ValueModeLabel
	ValueModePlace
)

func (mode ValueMode) Valid() bool {
	return mode >= ValueModeNone && mode <= ValueModePlace
}

type ResultArity uint8

const (
	ResultArityInvalid ResultArity = iota
	ResultArityZero
	ResultArityOne
	ResultArityTuple
	ResultArityCommaOk
)

func (arity ResultArity) Valid() bool {
	return arity >= ResultArityZero &&
		arity <= ResultArityCommaOk
}

type PlaceKind uint8

const (
	PlaceInvalid PlaceKind = iota
	PlaceNone
	PlaceBinding
	PlaceField
	PlaceArrayElement
	PlaceSliceElement
	PlacePointerDereference
	PlaceBlank
)

func (kind PlaceKind) Valid() bool {
	return kind >= PlaceNone &&
		kind <= PlaceBlank
}

type OperationKind uint16

const (
	OperationInvalid OperationKind = iota
	OperationLiteral
	OperationLoad
	OperationStore
	OperationDeclare
	OperationFunctionValue
	OperationComposite
	OperationFieldSelect
	OperationMethodValue
	OperationMethodExpression
	OperationPackageSelect
	OperationIndex
	OperationMapLookup
	OperationSlice
	OperationTypeAssert
	OperationCall
	OperationBuiltinCall
	OperationConvert
	OperationGenericInstantiate
	OperationDereference
	OperationAddress
	OperationReceive
	OperationUnary
	OperationBinary
	OperationKeyedElement
	OperationSend
	OperationIncrement
	OperationAssign
	OperationSpawn
	OperationDefer
	OperationReturn
	OperationBranch
	OperationBlock
	OperationIf
	OperationCase
	OperationSwitch
	OperationTypeSwitch
	OperationCommClause
	OperationSelect
	OperationFor
	OperationRange
	OperationExpressionStatement
	OperationEmpty
	OperationLabel
	OperationDeclarationStatement
	OperationPackageInitialization

	operationKindCount = OperationPackageInitialization
)

func (kind OperationKind) Valid() bool {
	return kind > OperationInvalid &&
		kind <= operationKindCount
}

func (kind OperationKind) String() string {
	if !kind.Valid() {
		return fmt.Sprintf(
			"semantic.OperationKind(%d)", uint16(kind),
		)
	}
	return operationKindNames[kind]
}

var operationKindNames = [operationKindCount + 1]string{
	OperationLiteral:               "literal",
	OperationLoad:                  "load",
	OperationStore:                 "store",
	OperationDeclare:               "declare",
	OperationFunctionValue:         "function-value",
	OperationComposite:             "composite",
	OperationFieldSelect:           "field-select",
	OperationMethodValue:           "method-value",
	OperationMethodExpression:      "method-expression",
	OperationPackageSelect:         "package-select",
	OperationIndex:                 "index",
	OperationMapLookup:             "map-lookup",
	OperationSlice:                 "slice",
	OperationTypeAssert:            "type-assert",
	OperationCall:                  "call",
	OperationBuiltinCall:           "builtin-call",
	OperationConvert:               "convert",
	OperationGenericInstantiate:    "generic-instantiate",
	OperationDereference:           "dereference",
	OperationAddress:               "address",
	OperationReceive:               "receive",
	OperationUnary:                 "unary",
	OperationBinary:                "binary",
	OperationKeyedElement:          "keyed-element",
	OperationSend:                  "send",
	OperationIncrement:             "increment",
	OperationAssign:                "assign",
	OperationSpawn:                 "spawn",
	OperationDefer:                 "defer",
	OperationReturn:                "return",
	OperationBranch:                "branch",
	OperationBlock:                 "block",
	OperationIf:                    "if",
	OperationCase:                  "case",
	OperationSwitch:                "switch",
	OperationTypeSwitch:            "type-switch",
	OperationCommClause:            "comm-clause",
	OperationSelect:                "select",
	OperationFor:                   "for",
	OperationRange:                 "range",
	OperationExpressionStatement:   "expression-statement",
	OperationEmpty:                 "empty",
	OperationLabel:                 "label",
	OperationDeclarationStatement:  "declaration-statement",
	OperationPackageInitialization: "package-initialization",
}

type SelectionKind uint8

const (
	SelectionInvalid SelectionKind = iota
	SelectionField
	SelectionMethodValue
	SelectionMethodExpression
)

func (kind SelectionKind) Valid() bool {
	return kind >= SelectionField &&
		kind <= SelectionMethodExpression
}

type ObjectReferenceKind uint8

const (
	ObjectReferenceInvalid ObjectReferenceKind = iota
	ObjectReferenceNone
	ObjectReferenceDeclaration
	ObjectReferenceBinding
)

func (kind ObjectReferenceKind) Valid() bool {
	return kind >= ObjectReferenceNone &&
		kind <= ObjectReferenceBinding
}

type UnsupportedReason uint8

const (
	UnsupportedInvalid UnsupportedReason = iota
	UnsupportedExternalBoundary
	UnsupportedIntrinsicBoundary
	UnsupportedExplicitContract
)

func (reason UnsupportedReason) Valid() bool {
	return reason >= UnsupportedExternalBoundary &&
		reason <= UnsupportedExplicitContract
}
