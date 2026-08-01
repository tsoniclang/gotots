package api

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type RootRequestKind uint8

const (
	RootRequestInvalid RootRequestKind = iota
	RootRequestImport
	RootRequestDeclarationRequirement
	RootRequestArtifactDependency
)

type PlacementScope uint8

const (
	ScopeInvalid PlacementScope = iota
	ScopeFileImports
	ScopeOwningFile
	ScopeCompilationSupport
)

type ExecutionConstraint uint8

const (
	ExecutionInvalid ExecutionConstraint = iota
	ExecutionStatic
)

type ImportPhase uint8

const (
	ImportPhaseInvalid ImportPhase = iota
	ImportPhaseType
	ImportPhaseValue
)

type ImportBindingKind uint8

const (
	ImportBindingInvalid ImportBindingKind = iota
	ImportBindingNamed
	ImportBindingNamespace
)

type RootRequestOwner struct {
	kind                   RootRequestKind
	importBinding          ImportBindingKind
	modulePath             string
	exportedName           string
	declarationRequirement DeclarationRequirement
	artifactDependency     ArtifactDependency
}

type rootRequestPayload struct {
	owner           RootRequestOwner
	importPhase     ImportPhase
	localName       string
	moduleSpecifier tsgo.StringLiteral
	specifier       tsgo.ImportSpecifier
	namespace       tsgo.NamespaceImport
	primitiveAlias  PrimitiveAlias
	runtimeSymbol   RuntimeSymbol
}

type RootRequest struct {
	payload  *rootRequestPayload
	sequence *rootRequestSequence
}

func NewImportRequest(
	factory tsgo.Factory,
	phase ImportPhase,
	modulePath string,
	exportedName string,
	localName string,
) (RootRequest, error) {
	if phase != ImportPhaseType && phase != ImportPhaseValue {
		return RootRequest{}, &RootRequestError{Reason: "invalid import phase"}
	}
	if modulePath == "" {
		return RootRequest{}, &RootRequestError{Reason: "module path is empty"}
	}
	if exportedName == "" {
		return RootRequest{}, &RootRequestError{Reason: "exported name is empty"}
	}
	if localName == "" {
		return RootRequest{}, &RootRequestError{Reason: "local name is empty"}
	}
	var propertyName tsgo.ModuleExportName
	if localName != exportedName {
		propertyName = factory.Identifier(exportedName)
	}
	return RootRequest{payload: &rootRequestPayload{
		owner: RootRequestOwner{
			kind:          RootRequestImport,
			importBinding: ImportBindingNamed,
			modulePath:    modulePath,
			exportedName:  exportedName,
		},
		importPhase:     phase,
		localName:       localName,
		moduleSpecifier: factory.StringLiteral(modulePath, tsgo.TokenFlagsNone),
		specifier: factory.ImportSpecifier(
			false,
			propertyName,
			factory.Identifier(localName),
		),
	}}, nil
}

func NewNamespaceImportRequest(
	factory tsgo.Factory,
	phase ImportPhase,
	modulePath string,
	localName string,
) (RootRequest, error) {
	if phase != ImportPhaseType && phase != ImportPhaseValue {
		return RootRequest{}, &RootRequestError{Reason: "invalid import phase"}
	}
	if modulePath == "" {
		return RootRequest{}, &RootRequestError{Reason: "module path is empty"}
	}
	if localName == "" {
		return RootRequest{}, &RootRequestError{Reason: "local name is empty"}
	}
	return RootRequest{payload: &rootRequestPayload{
		owner: RootRequestOwner{
			kind:          RootRequestImport,
			importBinding: ImportBindingNamespace,
			modulePath:    modulePath,
		},
		importPhase:     phase,
		localName:       localName,
		moduleSpecifier: factory.StringLiteral(modulePath, tsgo.TokenFlagsNone),
		namespace:       factory.NamespaceImport(factory.Identifier(localName)),
	}}, nil
}

func NewPrimitiveAliasRequest(
	factory tsgo.Factory,
	modulePath string,
	alias PrimitiveAlias,
	localName string,
) (RootRequest, error) {
	exportedName, err := PrimitiveAliasName(alias)
	if err != nil {
		return RootRequest{}, err
	}
	request, err := NewImportRequest(
		factory,
		ImportPhaseType,
		modulePath,
		exportedName,
		localName,
	)
	if err != nil {
		return RootRequest{}, err
	}
	request.payload.primitiveAlias = alias
	return request, nil
}

func NewRuntimeImportRequest(
	factory tsgo.Factory,
	phase ImportPhase,
	modulePath string,
	symbol RuntimeSymbol,
	localName string,
) (RootRequest, error) {
	contract, err := RuntimeContract(symbol)
	if err != nil {
		return RootRequest{}, err
	}
	if !contract.AllowsImportPhase(phase) {
		return RootRequest{}, &RootRequestError{
			Reason: "runtime symbol does not allow the requested import phase",
		}
	}
	request, err := NewImportRequest(
		factory,
		phase,
		modulePath,
		contract.ExportedName(),
		localName,
	)
	if err != nil {
		return RootRequest{}, err
	}
	request.payload.runtimeSymbol = symbol
	return request, nil
}

func NewNamedStructOperationRequest(
	typeName *types.TypeName,
	operation NamedStructOperation,
) (RootRequest, error) {
	requirement, err := NewNamedStructOperationRequirement(typeName, operation)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewLexicalNamedStructOperationRequest(
	owner ArtifactOwner,
	typeName *types.TypeName,
	operation NamedStructOperation,
) (RootRequest, error) {
	requirement, err := NewLexicalNamedStructOperationRequirement(
		owner,
		typeName,
		operation,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func newDeclarationRequirementRequest(
	requirement DeclarationRequirement,
) RootRequest {
	return RootRequest{payload: &rootRequestPayload{
		owner: RootRequestOwner{
			kind:                   RootRequestDeclarationRequirement,
			declarationRequirement: requirement,
		},
	}}
}

func NewAddressableStorageRequest(
	owner ArtifactOwner,
	variable *types.Var,
) (RootRequest, error) {
	requirement, err := NewAddressableStorageRequirement(owner, variable)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewCallableControlRequest(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	control CallableControlFacet,
) (RootRequest, error) {
	requirement, err := NewCallableControlRequirement(
		owner,
		enclosing,
		callable,
		control,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewGotoControlRequest(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	label *types.Label,
	position token.Pos,
) (RootRequest, error) {
	requirement, err := NewGotoControlRequirement(
		owner,
		enclosing,
		callable,
		label,
		position,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewIteratorReturnControlRequest(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	source *ast.RangeStmt,
) (RootRequest, error) {
	requirement, err := NewIteratorReturnControlRequirement(
		owner,
		enclosing,
		callable,
		source,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewDirectCallableControlRequest(
	owner *types.Func,
	control CallableControlFacet,
) (RootRequest, error) {
	requirement, err := NewDirectCallableControlRequirement(owner, control)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewConstantProjectionRequest(
	constant *types.Const,
	projection types.BasicKind,
) (RootRequest, error) {
	requirement, err := NewConstantProjectionRequirement(constant, projection)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewLocalConstantProjectionRequest(
	owner *types.Func,
	constant *types.Const,
	projection types.BasicKind,
) (RootRequest, error) {
	requirement, err := NewLocalConstantProjectionRequirement(
		owner,
		constant,
		projection,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewAnonymousStructRequest(
	artifact *GeneratedArtifact,
	demand AnonymousStructDemand,
) (RootRequest, error) {
	requirement, err := NewAnonymousStructRequirement(
		artifact,
		demand,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewMapSpecializationRequest(
	artifact *GeneratedArtifact,
	demand MapSpecializationDemand,
) (RootRequest, error) {
	requirement, err := NewMapSpecializationRequirement(artifact, demand)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewInterfaceAdapterRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewInterfaceAdapterRequirement(artifact)
	return generatedDefinitionRequest(requirement, err)
}

func NewInterfaceAdapterContractRequest(
	artifact *GeneratedArtifact,
	contract *types.Interface,
	contractKey string,
) (RootRequest, error) {
	requirement, err := NewInterfaceAdapterContractRequirement(
		artifact,
		contract,
		contractKey,
	)
	return generatedDefinitionRequest(requirement, err)
}

func NewAnonymousInterfaceRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewAnonymousInterfaceRequirement(artifact)
	return generatedDefinitionRequest(requirement, err)
}

func NewInterfaceMethodTokenRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewInterfaceMethodTokenRequirement(artifact)
	return generatedDefinitionRequest(requirement, err)
}

func NewInterfaceDynamicTypeTokenRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewInterfaceDynamicTypeTokenRequirement(artifact)
	return generatedDefinitionRequest(requirement, err)
}

func NewProviderInterfaceBridgeRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewProviderInterfaceBridgeRequirement(artifact)
	return generatedDefinitionRequest(requirement, err)
}

func generatedDefinitionRequest(
	requirement DeclarationRequirement,
	err error,
) (RootRequest, error) {
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r RootRequest) Kind() RootRequestKind {
	if r.payload == nil {
		return RootRequestInvalid
	}
	return r.payload.owner.kind
}

func (r RootRequest) LegalScope() PlacementScope {
	if r.payload == nil {
		return ScopeInvalid
	}
	if r.payload.owner.kind == RootRequestImport {
		return ScopeFileImports
	}
	if r.payload.owner.kind == RootRequestDeclarationRequirement {
		if artifact, ok := r.payload.owner.declarationRequirement.
			GeneratedArtifact(); ok &&
			(artifact.Placement() ==
				GeneratedArtifactPlacementCompilation ||
				artifact.Placement() ==
					GeneratedArtifactPlacementContract) {
			return ScopeCompilationSupport
		}
		return ScopeOwningFile
	}
	return ScopeInvalid
}

func (r RootRequest) PreferredScope() PlacementScope {
	return r.LegalScope()
}

func (r RootRequest) Execution() ExecutionConstraint {
	if r.payload == nil {
		return ExecutionInvalid
	}
	if r.payload.owner.kind == RootRequestImport {
		return ExecutionStatic
	}
	if r.payload.owner.kind == RootRequestDeclarationRequirement {
		return ExecutionStatic
	}
	return ExecutionInvalid
}

func (r RootRequest) Owner() RootRequestOwner {
	if r.payload == nil {
		return RootRequestOwner{}
	}
	return r.payload.owner
}

func (r RootRequest) ImportPhase() ImportPhase {
	if r.payload == nil {
		return ImportPhaseInvalid
	}
	return r.payload.importPhase
}

func (r RootRequest) ImportBinding() ImportBindingKind {
	if r.payload == nil {
		return ImportBindingInvalid
	}
	return r.payload.owner.importBinding
}

func (r RootRequest) ModulePath() string {
	if r.payload == nil {
		return ""
	}
	return r.payload.owner.modulePath
}

func (r RootRequest) ExportedName() string {
	if r.payload == nil {
		return ""
	}
	return r.payload.owner.exportedName
}

func (r RootRequest) LocalName() string {
	if r.payload == nil {
		return ""
	}
	return r.payload.localName
}

func (r RootRequest) ModuleSpecifier() tsgo.StringLiteral {
	if r.payload == nil {
		return nil
	}
	return r.payload.moduleSpecifier
}

func (r RootRequest) Specifier() tsgo.ImportSpecifier {
	if r.payload == nil ||
		r.payload.owner.importBinding != ImportBindingNamed {
		return nil
	}
	return r.payload.specifier
}

func (r RootRequest) NamespaceSpecifier() tsgo.NamespaceImport {
	if r.payload == nil ||
		r.payload.owner.importBinding != ImportBindingNamespace {
		return nil
	}
	return r.payload.namespace
}

func (r RootRequest) PrimitiveAlias() (PrimitiveAlias, bool) {
	if r.payload == nil || r.payload.primitiveAlias == PrimitiveInvalid {
		return PrimitiveInvalid, false
	}
	return r.payload.primitiveAlias, true
}

func (r RootRequest) RuntimeSymbol() (RuntimeSymbol, bool) {
	if r.payload == nil || r.payload.runtimeSymbol == RuntimeInvalid {
		return RuntimeInvalid, false
	}
	return r.payload.runtimeSymbol, true
}

func (r RootRequest) DeclarationRequirement() (
	DeclarationRequirement,
	bool,
) {
	if r.payload == nil {
		return DeclarationRequirement{}, false
	}
	requirement := r.payload.owner.declarationRequirement
	if r.payload.owner.kind != RootRequestDeclarationRequirement ||
		!requirement.Valid() {
		return DeclarationRequirement{}, false
	}
	return requirement, true
}

func (r RootRequest) ArtifactDependency() (ArtifactDependency, bool) {
	if r.payload == nil {
		return ArtifactDependency{}, false
	}
	dependency := r.payload.owner.artifactDependency
	if r.payload.owner.kind != RootRequestArtifactDependency ||
		!dependency.Valid() {
		return ArtifactDependency{}, false
	}
	return dependency, true
}

type RootRequestError struct {
	Reason string
}

func (e *RootRequestError) Error() string {
	return fmt.Sprintf("create root request: %s", e.Reason)
}
