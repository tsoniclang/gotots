package api

import (
	"fmt"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/types"
	"slices"
)

type PackageVariableReference struct {
	qualifier string
	stateName string
	fieldName string
	requests  []RootRequest
	provider  bool
}

func NewQualifiedPackageVariableReference(
	qualifier string,
	stateName string,
	fieldName string,
	requests ...RootRequest,
) (PackageVariableReference, error) {
	if qualifier == "" {
		return PackageVariableReference{},
			&NameError{Reason: "package state qualifier is empty"}
	}
	reference, err := NewPackageVariableReference(
		stateName,
		fieldName,
		requests...,
	)
	if err != nil {
		return PackageVariableReference{}, err
	}
	reference.qualifier = qualifier
	return reference, nil
}

func NewProviderQualifiedPackageVariableReference(
	qualifier string,
	stateName string,
	fieldName string,
	requests ...RootRequest,
) (PackageVariableReference, error) {
	reference, err := NewQualifiedPackageVariableReference(
		qualifier,
		stateName,
		fieldName,
		requests...,
	)
	if err != nil {
		return PackageVariableReference{}, err
	}
	reference.provider = true
	return reference, nil
}

func NewPackageVariableReference(
	stateName string,
	fieldName string,
	requests ...RootRequest,
) (PackageVariableReference, error) {
	switch {
	case stateName == "":
		return PackageVariableReference{},
			&NameError{Reason: "package state name is empty"}
	case fieldName == "":
		return PackageVariableReference{},
			&NameError{Reason: "package variable field name is empty"}
	}
	return PackageVariableReference{
		stateName: stateName,
		fieldName: fieldName,
		requests:  slices.Clone(requests),
	}, nil
}

func (r PackageVariableReference) StateName() string {
	return r.stateName
}

func (r PackageVariableReference) FieldName() string {
	return r.fieldName
}

func (r PackageVariableReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

func (r PackageVariableReference) ProviderBoundary() bool {
	return r.provider
}

func (r PackageVariableReference) Expression(
	factory tsgo.Factory,
) tsgo.PropertyAccessExpression {
	var state tsgo.Expression = factory.Identifier(r.stateName)
	if r.qualifier != "" {
		state = factory.PropertyAccessExpression(
			factory.Identifier(r.qualifier),
			nil,
			factory.Identifier(r.stateName),
			tsgo.NodeFlagsNone,
		)
	}
	return factory.PropertyAccessExpression(
		state,
		nil,
		factory.Identifier(r.fieldName),
		tsgo.NodeFlagsNone,
	)
}

type DeclarationOwnershipNames interface {
	ProviderOwnedDeclaration(types.Object) (bool, error)
	EnvironmentOwnedDeclaration(types.Object) (bool, error)
}

type ProviderCallableEffectNames interface {
	ProviderCallableEffect(*types.Func) (gostdlib.EffectKind, bool, error)
}

type ProviderDefinedCallableEffectNames interface {
	ProviderDefinedCallableEffect(*types.TypeName) (
		gostdlib.EffectKind,
		bool,
		error,
	)
}

type Names interface {
	DeclarationOwnershipNames
	Declare(types.Object) (string, error)
	Parameter(*types.Var, int) (string, error)
	Result(*types.Var, int) (string, error)
	Reference(types.Object) (NameReference, error)
	ProviderGenericTypeArguments(*types.Func) (
		[]GenericTypeArgumentProjection,
		bool,
		error,
	)
	TypeReference(types.Object) (NameReference, error)
	PackageVariable(*types.Var) (PackageVariableReference, error)
	NamedStructConstructor(*types.TypeName) (NameReference, error)
	NamedStructOperation(*types.TypeName, NamedStructOperation) (NameReference, error)
	NamedStructStorage(*types.TypeName) (NameReference, error)
	ProviderStructField(*types.TypeName, *types.Var) (
		gostdlib.ProviderStructField,
		bool,
		error,
	)
	DefinedValueRepresentation(*types.TypeName) (DefinedValueRepresentation, error)
	AnonymousStruct(
		*types.Struct,
		AnonymousStructDemand,
		ImportPhase,
	) (NameReference, error)
	AnonymousStructStorage(*types.Struct) (NameReference, error)
	MapSpecialization(
		types.Type,
		MapSpecializationDemand,
	) (NameReference, error)
	InterfaceAdapter(types.Type, types.Type) (NameReference, error)
	InterfaceContractDemand(types.Type, types.Type) ([]RootRequest, error)
	InterfaceDynamicType(types.Type) (NameReference, error)
	ProviderInterface(types.Type) (gostdlib.ProviderInterface, bool, error)
	ProviderInterfaceBridge(types.Type) (NameReference, bool, error)
	ProviderProfileInterfaceBridge(
		types.Type,
		[]gostdlib.ProviderCallableProfileInterface,
	) (ProviderProfileBridgeReference, bool, error)
	ProviderInterfaceCapability(types.Type, types.Type, string) (
		ProviderInterfaceCapabilityReference,
		bool,
		error,
	)
	ProviderCallableProfile(*types.Func, string) (
		ProviderCallableProfileReference,
		bool,
		error,
	)
	ProviderCallableProfileCandidates(*types.Func) (
		[]ProviderCallableProfileCandidate,
		bool,
		error,
	)
	ProviderCallableParameters(*types.Func) (
		[]gostdlib.ProviderCallableParameterDocument,
		bool,
		error,
	)
	ProviderStatefulProfileCandidates(*types.TypeName) (
		[]ProviderStatefulProfileCandidate,
		bool,
		error,
	)
	ProviderStatefulProfileTarget(
		*types.TypeName,
		string,
		ImportPhase,
	) (NameReference, error)
	ProviderRepresentationOwnsMethod(types.Type, *types.Func) (bool, error)
	InterfaceType(types.Type) (NameReference, error)
	InterfaceContract(types.Type) (InterfaceContractReference, error)
	RecoveryCallable(*types.Func) (RecoveryCallableReference, bool, error)
	DeferredCallable(*types.Func, string) (NameReference, error)
	DeferredCallableRegistry(*types.Signature) (NameReference, error)
	MethodTarget(*types.Func) (MethodTarget, error)
	InterfaceMethodName(*types.Func) (string, error)
	InterfaceMethodCallable(*types.Func) (
		InterfaceMethodCallableReference,
		error,
	)
	InterfaceMethodToken(*types.Func) (NameReference, error)
	GenericCapability(
		GenericOperationSelection,
		*types.Signature,
	) (GenericCapabilityReference, error)
	CallableABI(*types.Signature) (CallableABIReference, error)
	SourceCallableABI(
		types.Object,
		*types.Signature,
	) (CallableABIReference, error)
	ConstantValue(*types.Const) (NameReference, bool, error)
	ConstantProjection(*types.Const, types.BasicKind) (NameReference, error)
	Member(*types.Var) (string, error)
	Primitive(PrimitiveAlias) (NameReference, error)
	ProviderPrimitive(PrimitiveAlias) (NameReference, error)
	TsonicCore(tsoniccore.Symbol) (NameReference, error)
	Runtime(RuntimeSymbol, ImportPhase) (NameReference, error)
	ExternalProviderFunction(string, string) (NameReference, error)
	Temporary(TemporaryKind) (string, error)
	ModuleExport(types.Object) (bool, error)
}

type TypeRepresentationNames interface {
	TypeRepresentation(
		*types.TypeName,
		TypeRepresentationFacet,
	) ([]RootRequest, error)
	AnonymousStructTypeRepresentation(
		*types.Struct,
		TypeRepresentationFacet,
	) ([]RootRequest, error)
}

func TemporaryPrefix(kind TemporaryKind) (string, error) {
	switch kind {
	case TemporaryAssignmentValue:
		return "assignmentValue", nil
	case TemporaryMultipleResults:
		return "results", nil
	case TemporaryCompositeField:
		return "fieldValue", nil
	case TemporaryStructSource:
		return "structValue", nil
	case TemporaryReceiverValue:
		return "receiver", nil
	case TemporaryCallArgument:
		return "argument", nil
	case TemporaryCallCallee:
		return "callee", nil
	case TemporaryArrayReceiver:
		return "arrayValue", nil
	case TemporarySliceElement:
		return "sliceElement", nil
	case TemporarySliceReceiver:
		return "sliceValue", nil
	case TemporarySliceOperand:
		return "sliceOperand", nil
	case TemporaryStoreOperand:
		return "storeTarget", nil
	case TemporaryMapOperand:
		return "mapValue", nil
	case TemporaryAddressOperand:
		return "addressValue", nil
	case TemporaryConversionOperand:
		return "convertedValue", nil
	case TemporaryArrayComparison:
		return "arrayComparison", nil
	case TemporaryEqualityOperand:
		return "equalityOperand", nil
	case TemporaryBinaryOperand:
		return "binaryOperand", nil
	case TemporaryLogicalResult:
		return "logicalResult", nil
	case TemporaryArrayHash:
		return "arrayHash", nil
	case TemporaryArrayConstruction:
		return "arrayBuilder", nil
	case TemporarySliceConstruction:
		return "sliceBuilder", nil
	case TemporaryRangeOperand:
		return "rangeSource", nil
	case TemporaryRangeIndex:
		return "rangeIndex", nil
	case TemporaryRangeValue:
		return "rangeValue", nil
	case TemporaryRangeKeys:
		return "rangeKeys", nil
	case TemporaryRangeDecode:
		return "rangeDecoded", nil
	case TemporarySwitchTag:
		return "switchTag", nil
	case TemporarySwitchSelection:
		return "switchSelection", nil
	case TemporarySwitchMatch:
		return "switchMatch", nil
	case TemporaryTypeSwitchValue:
		return "typeSwitchValue", nil
	case TemporaryForFirstIteration:
		return "firstIteration", nil
	case TemporaryRangeState:
		return "rangeState", nil
	case TemporaryDeferStack:
		return "deferredCalls", nil
	case TemporaryDeferredCall:
		return "deferredCall", nil
	case TemporaryRecoveryAuthority:
		return "recovery", nil
	case TemporaryActivePanic:
		return "activePanic", nil
	case TemporaryCaughtPanic:
		return "caughtPanic", nil
	case TemporaryReturnResult:
		return "returnValue", nil
	case TemporaryReturnLabel:
		return "returnBlock", nil
	case TemporaryControlTarget:
		return "controlTarget", nil
	case TemporaryGotoTarget:
		return "gotoTarget", nil
	case TemporaryGotoState:
		return "gotoState", nil
	case TemporaryGotoDispatch:
		return "gotoDispatch", nil
	case TemporaryChannelOperand:
		return "channelValue", nil
	case TemporaryChannelResult:
		return "receivedValue", nil
	case TemporarySelectCase:
		return "selectCase", nil
	case TemporaryRangeReturn:
		return "rangeResult", nil
	default:
		return "", &NameError{
			Reason: fmt.Sprintf("temporary kind %d is invalid", kind),
		}
	}
}

type ArtifactOwner struct {
	source             types.Object
	initializerPackage *types.Package
	initializer        *types.Initializer
	assemblyPackage    *types.Package
	generated          *GeneratedArtifact
}

func PackageAssemblyArtifactOwner(
	sourcePackage *types.Package,
) (ArtifactOwner, error) {
	if sourcePackage == nil || sourcePackage.Path() == "" {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "package assembly artifact owner is invalid",
		}
	}
	return ArtifactOwner{assemblyPackage: sourcePackage}, nil
}

func PackageInitializerArtifactOwner(
	sourcePackage *types.Package,
	initializer *types.Initializer,
) (ArtifactOwner, error) {
	if sourcePackage == nil ||
		initializer == nil ||
		initializer.Rhs == nil ||
		len(initializer.Lhs) == 0 {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "package initializer artifact owner is invalid",
		}
	}
	for _, variable := range initializer.Lhs {
		if !validPackageInitializerTarget(sourcePackage, variable) {
			return ArtifactOwner{}, &RootRequestError{
				Reason: "package initializer artifact owner has a foreign target",
			}
		}
	}
	return ArtifactOwner{
		initializerPackage: sourcePackage,
		initializer:        initializer,
	}, nil
}

func SourceArtifactOwner(source types.Object) (ArtifactOwner, error) {
	if source == nil {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "source artifact owner is nil",
		}
	}
	return ArtifactOwner{source: source}, nil
}

func (c Context) WithSourceArtifactOwner(
	owner ArtifactOwner,
) (Context, error) {
	source, ok := owner.Source()
	if !ok || source == nil {
		return Context{}, &ContextError{
			Reason: "source artifact owner is invalid",
		}
	}
	if existing, bound := c.artifactOwner.Source(); bound &&
		existing != source {
		return Context{}, &ContextError{
			Reason: "source artifact owner is already bound",
		}
	}
	c.artifactOwner = owner
	return c, nil
}

func GeneratedArtifactOwner(
	generated *GeneratedArtifact,
) (ArtifactOwner, error) {
	if !generated.Valid() {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "generated artifact owner is invalid",
		}
	}
	return ArtifactOwner{generated: generated}, nil
}

func MustSourceArtifactOwner(source types.Object) ArtifactOwner {
	owner, err := SourceArtifactOwner(source)
	if err != nil {
		panic(err)
	}
	return owner
}

func MustPackageInitializerArtifactOwner(
	sourcePackage *types.Package,
	initializer *types.Initializer,
) ArtifactOwner {
	owner, err := PackageInitializerArtifactOwner(sourcePackage, initializer)
	if err != nil {
		panic(err)
	}
	return owner
}

func MustPackageAssemblyArtifactOwner(
	sourcePackage *types.Package,
) ArtifactOwner {
	owner, err := PackageAssemblyArtifactOwner(sourcePackage)
	if err != nil {
		panic(err)
	}
	return owner
}

func MustGeneratedArtifactOwner(generated *GeneratedArtifact) ArtifactOwner {
	owner, err := GeneratedArtifactOwner(generated)
	if err != nil {
		panic(err)
	}
	return owner
}

func (o ArtifactOwner) Valid() bool {
	variants := 0
	if o.source != nil {
		variants++
	}
	if o.initializerPackage != nil || o.initializer != nil {
		if o.initializerPackage == nil ||
			o.initializer == nil ||
			o.initializer.Rhs == nil ||
			len(o.initializer.Lhs) == 0 {
			return false
		}
		for _, variable := range o.initializer.Lhs {
			if !validPackageInitializerTarget(
				o.initializerPackage,
				variable,
			) {
				return false
			}
		}
		variants++
	}
	if o.generated != nil {
		variants++
	}
	if o.assemblyPackage != nil {
		if o.assemblyPackage.Path() == "" {
			return false
		}
		variants++
	}
	return variants == 1 &&
		(o.generated == nil || o.generated.Valid())
}

func (o ArtifactOwner) Source() (types.Object, bool) {
	return o.source, o.Valid() && o.source != nil
}

func (o ArtifactOwner) PackageInitializer() (
	*types.Package,
	*types.Initializer,
	bool,
) {
	return o.initializerPackage,
		o.initializer,
		o.Valid() && o.initializer != nil
}

func (o ArtifactOwner) Generated() (*GeneratedArtifact, bool) {
	return o.generated, o.Valid() && o.generated != nil
}

func (o ArtifactOwner) PackageAssembly() (*types.Package, bool) {
	return o.assemblyPackage,
		o.Valid() && o.assemblyPackage != nil
}

func (o ArtifactOwner) Package() *types.Package {
	if source, ok := o.Source(); ok {
		return source.Pkg()
	}
	if sourcePackage, _, ok := o.PackageInitializer(); ok {
		return sourcePackage
	}
	if generated, ok := o.Generated(); ok {
		return generated.LexicalOwner().Package()
	}
	if sourcePackage, ok := o.PackageAssembly(); ok {
		return sourcePackage
	}
	return nil
}

func (o ArtifactOwner) Name() string {
	if source, ok := o.Source(); ok {
		return source.Name()
	}
	if generated, ok := o.Generated(); ok {
		return generated.TargetName()
	}
	if sourcePackage, initializer, ok := o.PackageInitializer(); ok {
		return sourcePackage.Path() + ".$init@" +
			fmt.Sprint(initializer.Rhs.Pos())
	}
	if sourcePackage, ok := o.PackageAssembly(); ok {
		return sourcePackage.Path() + ".$assembly"
	}
	return ""
}

func (c Context) WithArtifactOwner(owner ArtifactOwner) Context {
	if !owner.Valid() ||
		c.artifactOwner.Valid() && c.artifactOwner != owner {
		panic("target artifact context owner is invalid")
	}
	c.artifactOwner = owner
	return c
}

func (c Context) ArtifactOwner() ArtifactOwner {
	return c.artifactOwner
}

func (c Context) FunctionArtifactOwner() (*types.Func, bool) {
	source, sourceOwned := c.artifactOwner.Source()
	function, callable := source.(*types.Func)
	return function, sourceOwned && callable
}

func validPackageInitializerTarget(
	sourcePackage *types.Package,
	variable *types.Var,
) bool {
	if sourcePackage == nil ||
		variable == nil ||
		variable.Pkg() != sourcePackage {
		return false
	}
	if variable.Name() == "_" {
		return variable.Parent() == nil
	}
	return variable.Parent() == sourcePackage.Scope()
}
