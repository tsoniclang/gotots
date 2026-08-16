package api

import (
	representationcontract "github.com/tsoniclang/gotots/internal/contracts/representation"
	runtimecontract "github.com/tsoniclang/gotots/internal/emit/api/runtimecontract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/types"
	"slices"
)

type RuntimeSymbol = runtimecontract.RuntimeSymbol

const (
	RuntimeInvalid                    = runtimecontract.RuntimeInvalid
	RuntimeStringIndex                = runtimecontract.RuntimeStringIndex
	RuntimeStringSlice                = runtimecontract.RuntimeStringSlice
	RuntimeStringMax                  = runtimecontract.RuntimeStringMax
	RuntimeStringMin                  = runtimecontract.RuntimeStringMin
	RuntimeStringEncodeRune           = runtimecontract.RuntimeStringEncodeRune
	RuntimeStringDecodeRune           = runtimecontract.RuntimeStringDecodeRune
	RuntimeArray                      = runtimecontract.RuntimeArray
	RuntimeArrayAllocate              = runtimecontract.RuntimeArrayAllocate
	RuntimeArrayView                  = runtimecontract.RuntimeArrayView
	RuntimeArrayLocation              = runtimecontract.RuntimeArrayLocation
	RuntimeStorageTypeToken           = runtimecontract.RuntimeStorageTypeToken
	RuntimeStoredValue                = runtimecontract.RuntimeStoredValue
	RuntimeStorageType                = runtimecontract.RuntimeStorageType
	RuntimeContainerStorageToken      = runtimecontract.RuntimeContainerStorageToken
	RuntimeContainerStoredValue       = runtimecontract.RuntimeContainerStoredValue
	RuntimeContainerStorageType       = runtimecontract.RuntimeContainerStorageType
	RuntimeSlice                      = runtimecontract.RuntimeSlice
	RuntimeSliceAddress               = runtimecontract.RuntimeSliceAddress
	RuntimeSliceStorage               = runtimecontract.RuntimeSliceStorage
	RuntimeSliceProjection            = runtimecontract.RuntimeSliceProjection
	RuntimeSliceArrayPointer          = runtimecontract.RuntimeSliceArrayPointer
	RuntimeArraySlice                 = runtimecontract.RuntimeArraySlice
	RuntimeSliceAppendSlice           = runtimecontract.RuntimeSliceAppendSlice
	RuntimeSliceClear                 = runtimecontract.RuntimeSliceClear
	RuntimeSliceRegion                = runtimecontract.RuntimeSliceRegion
	RuntimeMap                        = runtimecontract.RuntimeMap
	RuntimeMapHash                    = runtimecontract.RuntimeMapHash
	RuntimeMapClear                   = runtimecontract.RuntimeMapClear
	RuntimeMapKeys                    = runtimecontract.RuntimeMapKeys
	RuntimeMapValue                   = runtimecontract.RuntimeMapValue
	RuntimePanic                      = runtimecontract.RuntimePanic
	RuntimePanicValue                 = runtimecontract.RuntimePanicValue
	RuntimeRecovery                   = runtimecontract.RuntimeRecovery
	RuntimePanicNilError              = runtimecontract.RuntimePanicNilError
	RuntimePanicNilValue              = runtimecontract.RuntimePanicNilValue
	RuntimeDeferPop                   = runtimecontract.RuntimeDeferPop
	RuntimeDeferredRegistry           = runtimecontract.RuntimeDeferredRegistry
	RuntimeIntegerDivide              = runtimecontract.RuntimeIntegerDivide
	RuntimeIntegerRemainder           = runtimecontract.RuntimeIntegerRemainder
	RuntimeIntegerMax                 = runtimecontract.RuntimeIntegerMax
	RuntimeIntegerMin                 = runtimecontract.RuntimeIntegerMin
	RuntimeNumberIntDivide            = runtimecontract.RuntimeNumberIntDivide
	RuntimeNumberIntRemainder         = runtimecontract.RuntimeNumberIntRemainder
	RuntimeIntegerNormalizeSigned64   = runtimecontract.RuntimeIntegerNormalizeSigned64
	RuntimeIntegerNormalizeUnsigned64 = runtimecontract.RuntimeIntegerNormalizeUnsigned64
	RuntimeFloat32Round               = runtimecontract.RuntimeFloat32Round
	RuntimeComplex64                  = runtimecontract.RuntimeComplex64
	RuntimeComplex128                 = runtimecontract.RuntimeComplex128
	RuntimeComplexDivide              = runtimecontract.RuntimeComplexDivide
	RuntimeComplex64Add               = runtimecontract.RuntimeComplex64Add
	RuntimeComplex64Sub               = runtimecontract.RuntimeComplex64Sub
	RuntimeComplex64Mul               = runtimecontract.RuntimeComplex64Mul
	RuntimeComplex64Div               = runtimecontract.RuntimeComplex64Div
	RuntimeComplex64Neg               = runtimecontract.RuntimeComplex64Neg
	RuntimeComplex64Equal             = runtimecontract.RuntimeComplex64Equal
	RuntimeComplex128Add              = runtimecontract.RuntimeComplex128Add
	RuntimeComplex128Sub              = runtimecontract.RuntimeComplex128Sub
	RuntimeComplex128Mul              = runtimecontract.RuntimeComplex128Mul
	RuntimeComplex128Div              = runtimecontract.RuntimeComplex128Div
	RuntimeComplex128Neg              = runtimecontract.RuntimeComplex128Neg
	RuntimeComplex128Equal            = runtimecontract.RuntimeComplex128Equal
	RuntimeNumberToBigInt             = runtimecontract.RuntimeNumberToBigInt
	RuntimeInterfaceValue             = runtimecontract.RuntimeInterfaceValue
	RuntimeInterfaceNonNil            = runtimecontract.RuntimeInterfaceNonNil
	RuntimeInterfaceEqual             = runtimecontract.RuntimeInterfaceEqual
	RuntimeErrorMethodToken           = runtimecontract.RuntimeErrorMethodToken
	RuntimeRuntimeErrorToken          = runtimecontract.RuntimeRuntimeErrorToken
	RuntimeBuiltinErrorType           = runtimecontract.RuntimeBuiltinErrorType
	RuntimeBuiltinErrorContract       = runtimecontract.RuntimeBuiltinErrorContract
	RuntimeBuiltinErrorGuard          = runtimecontract.RuntimeBuiltinErrorGuard
	RuntimeErrorType                  = runtimecontract.RuntimeErrorType
	RuntimeErrorContract              = runtimecontract.RuntimeErrorContract
	RuntimeErrorGuard                 = runtimecontract.RuntimeErrorGuard
	RuntimeInterfaceFormat            = runtimecontract.RuntimeInterfaceFormat
	RuntimeProviderInterfaceBridge    = runtimecontract.RuntimeProviderInterfaceBridge
	RuntimeEmptyStruct                = runtimecontract.RuntimeEmptyStruct
	RuntimeChannel                    = runtimecontract.RuntimeChannel
	RuntimeReceiveChannel             = runtimecontract.RuntimeReceiveChannel
	RuntimeSendChannel                = runtimecontract.RuntimeSendChannel
	RuntimeSelectCase                 = runtimecontract.RuntimeSelectCase
	RuntimeSelect                     = runtimecontract.RuntimeSelect
	RuntimeScheduler                  = runtimecontract.RuntimeScheduler
	RuntimeSelectReady                = runtimecontract.RuntimeSelectReady
	RuntimeSelectAttempt              = runtimecontract.RuntimeSelectAttempt
	RuntimeUnsafeString               = runtimecontract.RuntimeUnsafeString
	RuntimeAwaitable                  = runtimecontract.RuntimeAwaitable
)

type RuntimeModule = runtimecontract.RuntimeModule

const (
	RuntimeModuleInvalid          = runtimecontract.RuntimeModuleInvalid
	RuntimeModuleString           = runtimecontract.RuntimeModuleString
	RuntimeModuleArray            = runtimecontract.RuntimeModuleArray
	RuntimeModuleSlice            = runtimecontract.RuntimeModuleSlice
	RuntimeModuleMap              = runtimecontract.RuntimeModuleMap
	RuntimeModulePanic            = runtimecontract.RuntimeModulePanic
	RuntimeModuleInteger          = runtimecontract.RuntimeModuleInteger
	RuntimeModuleFloat            = runtimecontract.RuntimeModuleFloat
	RuntimeModuleComplex          = runtimecontract.RuntimeModuleComplex
	RuntimeModuleConversion       = runtimecontract.RuntimeModuleConversion
	RuntimeModuleInterface        = runtimecontract.RuntimeModuleInterface
	RuntimeModuleInterfaceValue   = runtimecontract.RuntimeModuleInterfaceValue
	RuntimeModulePanicNil         = runtimecontract.RuntimeModulePanicNil
	RuntimeModuleChannel          = runtimecontract.RuntimeModuleChannel
	RuntimeModuleUnsafe           = runtimecontract.RuntimeModuleUnsafe
	RuntimeModuleStruct           = runtimecontract.RuntimeModuleStruct
	RuntimeModuleStorage          = runtimecontract.RuntimeModuleStorage
	RuntimeModuleDeferredRegistry = runtimecontract.RuntimeModuleDeferredRegistry
	RuntimeModuleScalar           = runtimecontract.RuntimeModuleScalar
)

type RuntimeSymbolContract = runtimecontract.RuntimeSymbolContract

type RuntimeSymbolError = runtimecontract.RuntimeSymbolError

func RuntimeContract(
	symbol RuntimeSymbol,
) (RuntimeSymbolContract, error) {
	return runtimecontract.RuntimeContract(symbol)
}

type IntegerRepresentation = representationcontract.IntegerRepresentation

const (
	IntegerRepresentationInvalid       = representationcontract.IntegerRepresentationInvalid
	IntegerRepresentationNumber        = representationcontract.IntegerRepresentationNumber
	IntegerRepresentationBigInt        = representationcontract.IntegerRepresentationBigInt
	IntegerRepresentationFixed64BigInt = representationcontract.IntegerRepresentationFixed64BigInt
)

type EvaluationOrder = representationcontract.EvaluationOrder

const (
	EvaluationOrderInvalid    = representationcontract.EvaluationOrderInvalid
	EvaluationOrderDirect     = representationcontract.EvaluationOrderDirect
	EvaluationOrderPreserveGo = representationcontract.EvaluationOrderPreserveGo
)

type MethodReceiverABI = representationcontract.MethodReceiverABI

const (
	MethodReceiverABIInvalid              = representationcontract.MethodReceiverABIInvalid
	MethodReceiverABISourceRepresentation = representationcontract.MethodReceiverABISourceRepresentation
	MethodReceiverABIContractDirect       = representationcontract.MethodReceiverABIContractDirect
)

type ConcurrencySemantics = representationcontract.ConcurrencySemantics

const (
	ConcurrencySemanticsDisabled    = representationcontract.ConcurrencySemanticsDisabled
	ConcurrencySemanticsCooperative = representationcontract.ConcurrencySemanticsCooperative
	ConcurrencySemanticsInvalid     = representationcontract.ConcurrencySemanticsInvalid
)

type PrimitiveAlias = representationcontract.PrimitiveAlias

const (
	PrimitiveInvalid = representationcontract.PrimitiveInvalid
	PrimitiveBool    = representationcontract.PrimitiveBool
	PrimitiveInt8    = representationcontract.PrimitiveInt8
	PrimitiveInt16   = representationcontract.PrimitiveInt16
	PrimitiveInt32   = representationcontract.PrimitiveInt32
	PrimitiveInt64   = representationcontract.PrimitiveInt64
	PrimitiveUint8   = representationcontract.PrimitiveUint8
	PrimitiveUint16  = representationcontract.PrimitiveUint16
	PrimitiveUint32  = representationcontract.PrimitiveUint32
	PrimitiveUint64  = representationcontract.PrimitiveUint64
	PrimitiveString  = representationcontract.PrimitiveString
	PrimitiveFloat32 = representationcontract.PrimitiveFloat32
	PrimitiveFloat64 = representationcontract.PrimitiveFloat64
	PrimitiveInt     = representationcontract.PrimitiveInt
	PrimitiveUint    = representationcontract.PrimitiveUint
	PrimitiveUintptr = representationcontract.PrimitiveUintptr
)

type PrimitiveAliasError = representationcontract.PrimitiveAliasError

type IntegerRepresentationError = representationcontract.IntegerRepresentationError

type IntegerCarrier = representationcontract.IntegerCarrier

const (
	IntegerCarrierInvalid = representationcontract.IntegerCarrierInvalid
	IntegerCarrierNumber  = representationcontract.IntegerCarrierNumber
	IntegerCarrierBigInt  = representationcontract.IntegerCarrierBigInt
)

type IntegerCarrierError = representationcontract.IntegerCarrierError

type NativeIntegerWidth = representationcontract.NativeIntegerWidth

const (
	NativeIntegerWidthInvalid = representationcontract.NativeIntegerWidthInvalid
	NativeIntegerWidth32      = representationcontract.NativeIntegerWidth32
	NativeIntegerWidth64      = representationcontract.NativeIntegerWidth64
)

type NativeIntegerWidthError = representationcontract.NativeIntegerWidthError

type ScalarABI = representationcontract.ScalarABI

func NewScalarABI(
	integer IntegerRepresentation,
	nativeWidth NativeIntegerWidth,
) (ScalarABI, error) {
	return representationcontract.NewScalarABI(integer, nativeWidth)
}

func NewScalarABIFromSizes(
	integer IntegerRepresentation,
	sizes types.Sizes,
) (ScalarABI, error) {
	return representationcontract.NewScalarABIFromSizes(integer, sizes)
}

func IntegerLiteral(
	factory tsgo.Factory,
	abi ScalarABI,
	alias PrimitiveAlias,
	decimal string,
) (tsgo.Expression, error) {
	return representationcontract.IntegerLiteral(
		factory,
		abi,
		alias,
		decimal,
	)
}

func IntegerCarrierRepresentation(
	alias PrimitiveAlias,
	abi ScalarABI,
) (IntegerCarrier, error) {
	return representationcontract.IntegerCarrierRepresentation(
		alias,
		abi,
	)
}

func PrimitiveAliasRepresentation(
	alias PrimitiveAlias,
	abi ScalarABI,
) (string, tsgo.KeywordTypeSyntaxKind, error) {
	return representationcontract.PrimitiveAliasRepresentation(alias, abi)
}

func PrimitiveAliasName(alias PrimitiveAlias) (string, error) {
	return representationcontract.PrimitiveAliasName(alias)
}

type GenericTypeArgumentFacet uint8

const (
	GenericTypeArgumentInvalid GenericTypeArgumentFacet = iota
	GenericTypeArgumentLogical
	GenericTypeArgumentStorage
	GenericTypeArgumentContainerStorage
	GenericTypeArgumentPointer
)

func (f GenericTypeArgumentFacet) Valid() bool {
	return f >= GenericTypeArgumentLogical &&
		f <= GenericTypeArgumentPointer
}

type GenericTypeArgumentProjection struct {
	parameter int
	facet     GenericTypeArgumentFacet
}

func NewGenericTypeArgumentProjection(
	parameter int,
	facet GenericTypeArgumentFacet,
) (GenericTypeArgumentProjection, error) {
	if parameter < 0 || !facet.Valid() {
		return GenericTypeArgumentProjection{}, &InvariantError{
			Reason: "generic type-argument projection is invalid",
		}
	}
	return GenericTypeArgumentProjection{
		parameter: parameter,
		facet:     facet,
	}, nil
}

func (p GenericTypeArgumentProjection) Parameter() int {
	return p.parameter
}

func (p GenericTypeArgumentProjection) Facet() GenericTypeArgumentFacet {
	return p.facet
}

type GenericRepresentationSelection struct {
	parameter *types.TypeParam
	facet     GenericRepresentationFacet
}

func (s GenericRepresentationSelection) Parameter() *types.TypeParam {
	return s.parameter
}

func (s GenericRepresentationSelection) Facet() GenericRepresentationFacet {
	return s.facet
}

type GenericRepresentationProfile struct {
	owner      types.Object
	parameters []*types.TypeParam
	masks      []uint8
}

func SelectGenericRepresentationProfile(
	owner types.Object,
	requirements []DeclarationRequirement,
) (GenericRepresentationProfile, error) {
	owner = GenericDeclarationOrigin(owner)
	parameters := GenericDeclarationParameters(owner)
	if owner == nil || len(parameters) == 0 {
		return GenericRepresentationProfile{}, &InvariantError{
			Reason: "generic representation owner is invalid",
		}
	}
	profile := GenericRepresentationProfile{
		owner:      owner,
		parameters: slices.Clone(parameters),
		masks:      make([]uint8, len(parameters)),
	}
	for _, requirement := range requirements {
		if requirement.Kind() != DeclarationRequirementGenericRepresentation {
			continue
		}
		selectedOwner, parameter, facet, ok :=
			requirement.GenericRepresentation()
		index, indexed := GenericDeclarationParameterIndex(owner, parameter)
		if !ok || selectedOwner != owner || !indexed {
			return GenericRepresentationProfile{}, &InvariantError{
				Reason: "generic representation requirement has foreign ownership",
			}
		}
		mask := uint8(1) << facet
		if profile.masks[index]&mask != 0 {
			return GenericRepresentationProfile{}, &InvariantError{
				Reason: "generic representation requirement is duplicated",
			}
		}
		profile.masks[index] |= mask
	}
	return profile, nil
}

func (p GenericRepresentationProfile) Valid() bool {
	if p.owner == nil ||
		GenericDeclarationOrigin(p.owner) != p.owner ||
		len(p.parameters) == 0 ||
		len(p.parameters) != len(p.masks) {
		return false
	}
	expected := GenericDeclarationParameters(p.owner)
	if len(expected) != len(p.parameters) {
		return false
	}
	var validMask uint8
	for _, facet := range GenericRepresentationFacetOrder() {
		validMask |= uint8(1) << facet
	}
	for index, parameter := range p.parameters {
		if parameter != expected[index] || p.masks[index]&^validMask != 0 {
			return false
		}
	}
	return true
}

func (p GenericRepresentationProfile) Owner() types.Object {
	if !p.Valid() {
		return nil
	}
	return p.owner
}

func (p GenericRepresentationProfile) Parameters() []*types.TypeParam {
	if !p.Valid() {
		return nil
	}
	return slices.Clone(p.parameters)
}

func (p GenericRepresentationProfile) Requires(
	parameter *types.TypeParam,
	facet GenericRepresentationFacet,
) bool {
	if !p.Valid() || !facet.Valid() {
		return false
	}
	index, ok := GenericDeclarationParameterIndex(p.owner, parameter)
	return ok && p.masks[index]&(uint8(1)<<facet) != 0
}

func (p GenericRepresentationProfile) OrderedFacets() []GenericRepresentationSelection {
	if !p.Valid() {
		return nil
	}
	var result []GenericRepresentationSelection
	for index, parameter := range p.parameters {
		for _, facet := range GenericRepresentationFacetOrder() {
			if p.masks[index]&(uint8(1)<<facet) == 0 {
				continue
			}
			result = append(result, GenericRepresentationSelection{
				parameter: parameter,
				facet:     facet,
			})
		}
	}
	return result
}

func NewGenericRepresentationRequirement(
	owner types.Object,
	parameter *types.TypeParam,
	facet GenericRepresentationFacet,
) (DeclarationRequirement, error) {
	owner, parameter, ok := GenericRepresentationParameter(owner, parameter)
	if !ok || !facet.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "generic representation requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:            MustSourceArtifactOwner(owner),
		kind:             DeclarationRequirementGenericRepresentation,
		genericParameter: parameter,
		genericFacet:     facet,
	}, nil
}

func NewGenericRepresentationRequest(
	owner types.Object,
	parameter *types.TypeParam,
	facet GenericRepresentationFacet,
) (RootRequest, error) {
	requirement, err := NewGenericRepresentationRequirement(
		owner,
		parameter,
		facet,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) GenericRepresentation() (
	types.Object,
	*types.TypeParam,
	GenericRepresentationFacet,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementGenericRepresentation {
		return nil, nil, GenericRepresentationInvalid, false
	}
	owner, ok := r.owner.Source()
	return owner, r.genericParameter, r.genericFacet, ok
}

func GenericDeclarationParameterIndex(
	owner types.Object,
	parameter *types.TypeParam,
) (int, bool) {
	owner = GenericDeclarationOrigin(owner)
	for index, selected := range GenericDeclarationParameters(owner) {
		if selected == parameter {
			return index, true
		}
	}
	return -1, false
}

func GenericRepresentationParameter(
	owner types.Object,
	parameter *types.TypeParam,
) (types.Object, *types.TypeParam, bool) {
	origin := GenericDeclarationOrigin(owner)
	index, ok := GenericDeclarationParameterIndex(origin, parameter)
	if !ok {
		return nil, nil, false
	}
	function, callable := origin.(*types.Func)
	if !callable {
		return origin, parameter, true
	}
	typeName := ValueReceiverTypeName(function)
	if typeName == nil {
		return origin, parameter, true
	}
	typeOwner := GenericDeclarationOrigin(typeName)
	parameters := GenericDeclarationParameters(typeOwner)
	if index >= len(parameters) {
		return nil, nil, false
	}
	return typeOwner, parameters[index], true
}

func (c Context) RequireGenericParameterRepresentation(
	parameter *types.TypeParam,
	facet GenericRepresentationFacet,
) ([]RootRequest, error) {
	_, ok := c.GenericParameterName(parameter)
	owner, selected, owned := GenericRepresentationParameter(
		c.genericParameterOwner,
		parameter,
	)
	if !ok || !owned || !facet.Valid() {
		return nil, &ContextError{
			Reason: "generic parameter representation is unavailable",
		}
	}
	request, err := NewGenericRepresentationRequest(owner, selected, facet)
	if err != nil {
		return nil, err
	}
	return []RootRequest{request}, nil
}

func (c Context) ResolveGenericRepresentationProfile(
	owner types.Object,
) (GenericRepresentationProfile, bool, error) {
	if c.genericResolver == nil {
		return GenericRepresentationProfile{}, false, &ContextError{
			Reason: "generic representation resolver is unavailable",
		}
	}
	return c.genericResolver.ResolveGenericRepresentationProfile(owner)
}
