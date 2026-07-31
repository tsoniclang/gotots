package api

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type IntegerRepresentation uint8

const (
	IntegerRepresentationInvalid IntegerRepresentation = iota
	IntegerRepresentationNumber
	IntegerRepresentationBigInt
)

func (r IntegerRepresentation) Valid() bool {
	return r == IntegerRepresentationNumber ||
		r == IntegerRepresentationBigInt
}

func (r IntegerRepresentation) String() string {
	switch r {
	case IntegerRepresentationNumber:
		return "number"
	case IntegerRepresentationBigInt:
		return "bigint"
	default:
		return fmt.Sprintf("integer-representation(%d)", r)
	}
}

type EvaluationOrder uint8

const (
	EvaluationOrderInvalid EvaluationOrder = iota
	EvaluationOrderDirect
	EvaluationOrderPreserveGo
)

func (o EvaluationOrder) Valid() bool {
	return o == EvaluationOrderDirect ||
		o == EvaluationOrderPreserveGo
}

func (o EvaluationOrder) String() string {
	switch o {
	case EvaluationOrderDirect:
		return "direct"
	case EvaluationOrderPreserveGo:
		return "preserve-go"
	default:
		return fmt.Sprintf("evaluation-order(%d)", o)
	}
}

type MethodReceiverABI uint8

const (
	MethodReceiverABIInvalid MethodReceiverABI = iota
	MethodReceiverABISourceRepresentation
	MethodReceiverABIContractDirect
)

func (a MethodReceiverABI) Valid() bool {
	return a == MethodReceiverABISourceRepresentation ||
		a == MethodReceiverABIContractDirect
}

type DefinedValueRepresentationKind uint8

const (
	DefinedValueRepresentationInvalid DefinedValueRepresentationKind = iota
	DefinedValueRepresentationGeneratedWrapper
	DefinedValueRepresentationProviderIdentity
	DefinedValueRepresentationProviderOperations
)

func (k DefinedValueRepresentationKind) Valid() bool {
	return k >= DefinedValueRepresentationGeneratedWrapper &&
		k <= DefinedValueRepresentationProviderOperations
}

type DefinedValueRepresentation struct {
	kind        DefinedValueRepresentationKind
	operations  NameReference
	cooperative bool
}

func NewProviderIdentityDefinedValueRepresentation(
	cooperative bool,
) (DefinedValueRepresentation, error) {
	return DefinedValueRepresentation{
		kind:        DefinedValueRepresentationProviderIdentity,
		cooperative: cooperative,
	}, nil
}

func NewDefinedValueRepresentation(
	kind DefinedValueRepresentationKind,
	operations NameReference,
) (DefinedValueRepresentation, error) {
	hasOperations := operations.Name() != ""
	if !kind.Valid() ||
		kind == DefinedValueRepresentationProviderIdentity ||
		(kind == DefinedValueRepresentationProviderOperations) != hasOperations {
		return DefinedValueRepresentation{}, &NameError{
			Name:   operations.Name(),
			Reason: "defined-value representation is invalid",
		}
	}
	return DefinedValueRepresentation{kind: kind, operations: operations}, nil
}

func (r DefinedValueRepresentation) Kind() DefinedValueRepresentationKind {
	return r.kind
}

func (r DefinedValueRepresentation) ProviderCallableEffect() (bool, bool) {
	return r.cooperative,
		r.kind == DefinedValueRepresentationProviderIdentity
}

func (r DefinedValueRepresentation) Operations() (NameReference, bool) {
	return r.operations, r.operations.Name() != ""
}

type ConcurrencySemantics uint8

const (
	ConcurrencySemanticsDisabled    ConcurrencySemantics = 0
	ConcurrencySemanticsCooperative ConcurrencySemantics = 1
	ConcurrencySemanticsInvalid     ConcurrencySemantics = 255
)

func (s ConcurrencySemantics) Valid() bool {
	return s == ConcurrencySemanticsDisabled ||
		s == ConcurrencySemanticsCooperative
}

func (s ConcurrencySemantics) String() string {
	switch s {
	case ConcurrencySemanticsDisabled:
		return "disabled"
	case ConcurrencySemanticsCooperative:
		return "cooperative"
	default:
		return fmt.Sprintf("concurrency-semantics(%d)", s)
	}
}

func IntegerLiteral(
	factory tsgo.Factory,
	representation IntegerRepresentation,
	decimal string,
) (tsgo.Expression, error) {
	if decimal == "" {
		return nil, &IntegerRepresentationError{
			Representation: representation,
		}
	}
	switch representation {
	case IntegerRepresentationNumber:
		return factory.NumericLiteral(decimal, tsgo.TokenFlagsNone), nil
	case IntegerRepresentationBigInt:
		return factory.BigIntLiteral(decimal+"n", tsgo.TokenFlagsNone), nil
	default:
		return nil, &IntegerRepresentationError{
			Representation: representation,
		}
	}
}

type PrimitiveAlias uint8

const (
	PrimitiveInvalid PrimitiveAlias = iota
	PrimitiveBool
	PrimitiveInt8
	PrimitiveInt16
	PrimitiveInt32
	PrimitiveInt64
	PrimitiveUint8
	PrimitiveUint16
	PrimitiveUint32
	PrimitiveUint64
	PrimitiveString
	PrimitiveFloat32
	PrimitiveFloat64
)

func PrimitiveAliasRepresentation(
	alias PrimitiveAlias,
	integer IntegerRepresentation,
) (string, tsgo.KeywordTypeSyntaxKind, error) {
	name, err := PrimitiveAliasName(alias)
	if err != nil {
		return "", 0, err
	}
	switch alias {
	case PrimitiveBool:
		return name, tsgo.KeywordTypeSyntaxKindBooleanKeyword, nil
	case PrimitiveString:
		return name, tsgo.KeywordTypeSyntaxKindStringKeyword, nil
	case PrimitiveFloat32, PrimitiveFloat64:
		return name, tsgo.KeywordTypeSyntaxKindNumberKeyword, nil
	case PrimitiveInt8,
		PrimitiveInt16,
		PrimitiveInt32,
		PrimitiveInt64,
		PrimitiveUint8,
		PrimitiveUint16,
		PrimitiveUint32,
		PrimitiveUint64:
		keyword, err := integerKeyword(integer)
		return name, keyword, err
	default:
		return "", 0, &PrimitiveAliasError{Alias: alias}
	}
}

func PrimitiveAliasName(alias PrimitiveAlias) (string, error) {
	switch alias {
	case PrimitiveBool:
		return "bool", nil
	case PrimitiveInt8:
		return "int8", nil
	case PrimitiveInt16:
		return "int16", nil
	case PrimitiveInt32:
		return "int32", nil
	case PrimitiveInt64:
		return "int64", nil
	case PrimitiveUint8:
		return "uint8", nil
	case PrimitiveUint16:
		return "uint16", nil
	case PrimitiveUint32:
		return "uint32", nil
	case PrimitiveUint64:
		return "uint64", nil
	case PrimitiveString:
		return "gostring", nil
	case PrimitiveFloat32:
		return "float32", nil
	case PrimitiveFloat64:
		return "float64", nil
	default:
		return "", &PrimitiveAliasError{Alias: alias}
	}
}

func integerKeyword(
	representation IntegerRepresentation,
) (tsgo.KeywordTypeSyntaxKind, error) {
	switch representation {
	case IntegerRepresentationNumber:
		return tsgo.KeywordTypeSyntaxKindNumberKeyword, nil
	case IntegerRepresentationBigInt:
		return tsgo.KeywordTypeSyntaxKindBigIntKeyword, nil
	default:
		return 0, &IntegerRepresentationError{Representation: representation}
	}
}

type PrimitiveAliasError struct {
	Alias PrimitiveAlias
}

func (e *PrimitiveAliasError) Error() string {
	return fmt.Sprintf("primitive alias %d is invalid", e.Alias)
}

type IntegerRepresentationError struct {
	Representation IntegerRepresentation
}

func (e *IntegerRepresentationError) Error() string {
	return fmt.Sprintf(
		"integer representation %d is invalid",
		e.Representation,
	)
}

type GoRuntimeType uint8

const (
	GoRuntimeTypeInvalid GoRuntimeType = iota
	GoRuntimeTypeBuiltinError
	GoRuntimeTypeError
	GoRuntimeTypePanicNilError
	GoRuntimeTypePanicNilPointer
)

func (k GoRuntimeType) Valid() bool {
	return k == GoRuntimeTypeBuiltinError ||
		k == GoRuntimeTypeError ||
		k == GoRuntimeTypePanicNilError ||
		k == GoRuntimeTypePanicNilPointer
}

type GoRuntimeContract interface {
	Owns(*types.Package) bool
	Classify(types.Type) GoRuntimeType
}

func (c Context) WithGoRuntimeContract(contract GoRuntimeContract) Context {
	if contract == nil {
		panic("Go runtime contract is nil")
	}
	c.goRuntime = contract
	return c
}

func (c Context) GoRuntimeType(sourceType types.Type) GoRuntimeType {
	if c.goRuntime == nil {
		return GoRuntimeTypeInvalid
	}
	return c.goRuntime.Classify(sourceType)
}

type RuntimeModule uint8

const (
	RuntimeModuleInvalid RuntimeModule = iota
	RuntimeModuleString
	RuntimeModulePointer
	RuntimeModuleArray
	RuntimeModuleSlice
	RuntimeModuleMap
	RuntimeModulePanic
	RuntimeModuleInteger
	RuntimeModuleFloat
	RuntimeModuleComplex
	RuntimeModuleConversion
	RuntimeModuleInterface
	RuntimeModuleInterfaceValue
	RuntimeModulePanicNil
	RuntimeModuleChannel
	RuntimeModuleUnsafePointer
	RuntimeModuleDenseIndex
	RuntimeModuleUnsafe
)

func runtimeContract(
	module RuntimeModule,
	outputPath string,
	exportedName string,
	typeUsable bool,
	dependencies ...RuntimeSymbol,
) RuntimeSymbolContract {
	return RuntimeSymbolContract{
		module:       module,
		outputPath:   outputPath,
		exportedName: exportedName,
		typeUsable:   typeUsable,
		dependencies: slices.Clone(dependencies),
	}
}

func concurrencyRuntimeContract(
	symbol RuntimeSymbol,
) (RuntimeSymbolContract, error) {
	switch symbol {
	case RuntimeChannel:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoChannel",
			true,
			RuntimeReceiveChannel,
			RuntimeSendChannel,
			RuntimeSelectCase,
			RuntimePanic,
		), nil
	case RuntimeReceiveChannel:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoReceiveChannel",
			true,
			RuntimeSelectCase,
		), nil
	case RuntimeSendChannel:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoSendChannel",
			true,
			RuntimeSelectCase,
		), nil
	case RuntimeSelectCase:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoSelectCase",
			true,
		), nil
	case RuntimeSelect:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"goSelect",
			false,
			RuntimeSelectReady,
			RuntimeSelectAttempt,
		), nil
	case RuntimeScheduler:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoScheduler",
			true,
			RuntimePanic,
		), nil
	case RuntimeSelectReady:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"goSelectReady",
			false,
			RuntimeSelectAttempt,
		), nil
	case RuntimeSelectAttempt:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"goSelectAttempt",
			false,
			RuntimeSelectCase,
		), nil
	default:
		return RuntimeSymbolContract{}, &RuntimeSymbolError{Symbol: symbol}
	}
}

func complexOperationContract(
	exportedName string,
	dependencies ...RuntimeSymbol,
) (RuntimeSymbolContract, error) {
	return runtimeContract(
		RuntimeModuleComplex,
		"runtime/complex.ts",
		exportedName,
		false,
		dependencies...,
	), nil
}

func (c RuntimeSymbolContract) Module() RuntimeModule {
	return c.module
}

func (c RuntimeSymbolContract) OutputPath() string {
	return c.outputPath
}

func (c RuntimeSymbolContract) ExportedName() string {
	return c.exportedName
}

func (c RuntimeSymbolContract) Dependencies() []RuntimeSymbol {
	return slices.Clone(c.dependencies)
}

func (c RuntimeSymbolContract) AllowsImportPhase(phase ImportPhase) bool {
	return phase == ImportPhaseValue ||
		(phase == ImportPhaseType && c.typeUsable)
}

type RuntimeSymbolError struct {
	Symbol RuntimeSymbol
}

func (e *RuntimeSymbolError) Error() string {
	return fmt.Sprintf("runtime symbol %d is invalid", e.Symbol)
}

func interfaceRuntimeContract(
	symbol RuntimeSymbol,
) (RuntimeSymbolContract, bool) {
	var contract RuntimeSymbolContract
	switch symbol {
	case RuntimeInterfaceValue:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoInterfaceValue",
			true,
		)
	case RuntimeInterfaceNonNil:
		contract = runtimeContract(
			RuntimeModuleInterface,
			"runtime/interface.ts",
			"goInterfaceNonNil",
			false,
			RuntimeInterfaceValue,
			RuntimePanic,
		)
	case RuntimeInterfaceEqual:
		contract = runtimeContract(
			RuntimeModuleInterface,
			"runtime/interface.ts",
			"goInterfaceEqual",
			false,
			RuntimeInterfaceValue,
		)
	case RuntimeErrorMethodToken:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoErrorMethodToken",
			false,
		)
	case RuntimeRuntimeErrorToken:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoRuntimeErrorMethodToken",
			false,
		)
	case RuntimeBuiltinErrorType:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoError",
			true,
			RuntimeInterfaceValue,
			RuntimeErrorMethodToken,
		)
	case RuntimeBuiltinErrorContract:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoError$contract",
			false,
			RuntimeErrorMethodToken,
		)
	case RuntimeBuiltinErrorGuard:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoError$is",
			false,
			RuntimeBuiltinErrorType,
			RuntimeBuiltinErrorContract,
		)
	case RuntimeErrorType:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoRuntimeError",
			true,
			RuntimeInterfaceValue,
			RuntimeErrorMethodToken,
			RuntimeRuntimeErrorToken,
		)
	case RuntimeErrorContract:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoRuntimeError$contract",
			false,
			RuntimeErrorMethodToken,
			RuntimeRuntimeErrorToken,
		)
	case RuntimeErrorGuard:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoRuntimeError$is",
			false,
			RuntimeErrorType,
			RuntimeErrorContract,
		)
	case RuntimeInterfaceFormat:
		contract = runtimeContract(
			RuntimeModuleInterface,
			"runtime/interface.ts",
			"GoInterfaceFormat",
			false,
			RuntimePanic,
		)
	default:
		return RuntimeSymbolContract{}, false
	}
	return contract, true
}
