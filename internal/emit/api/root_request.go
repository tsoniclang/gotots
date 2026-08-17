package api

import (
	"go/types"
	"slices"

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
	ImportBindingSideEffect
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
	if phase != ImportPhaseValue &&
		(phase != ImportPhaseType || !contract.TypeUsable()) {
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
	switch r.payload.owner.kind {
	case RootRequestImport:
		return ScopeFileImports
	case RootRequestDeclarationRequirement:
		artifact, ok := r.payload.owner.declarationRequirement.GeneratedArtifact()
		if ok && (artifact.Placement() == GeneratedArtifactPlacementCompilation ||
			artifact.Placement() == GeneratedArtifactPlacementContract) {
			return ScopeCompilationSupport
		}
		return ScopeOwningFile
	default:
		return ScopeInvalid
	}
}

func (r RootRequest) PreferredScope() PlacementScope {
	return r.LegalScope()
}

func (r RootRequest) Execution() ExecutionConstraint {
	if r.payload == nil {
		return ExecutionInvalid
	}
	switch r.payload.owner.kind {
	case RootRequestImport, RootRequestDeclarationRequirement:
		return ExecutionStatic
	default:
		return ExecutionInvalid
	}
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
	if r.payload == nil ||
		r.payload.owner.kind != RootRequestDeclarationRequirement {
		return DeclarationRequirement{}, false
	}
	return r.payload.owner.declarationRequirement, true
}

func (r RootRequest) ArtifactDependency() (ArtifactDependency, bool) {
	if r.payload == nil ||
		r.payload.owner.kind != RootRequestArtifactDependency {
		return ArtifactDependency{}, false
	}
	return r.payload.owner.artifactDependency, true
}

const (
	invalidRequestKindMask        = uint8(1) << RootRequestInvalid
	declarationRequestKindMask    = uint8(1) << RootRequestDeclarationRequirement
	nonDeclarationRequestKindMask = uint8(1)<<RootRequestImport |
		uint8(1)<<RootRequestArtifactDependency
)

type rootRequestSelection struct {
	request  RootRequest
	selected bool
}

func (r RootRequest) NestedRequests() ([]RootRequest, bool) {
	if r.sequence == nil {
		return nil, false
	}
	return slices.Clone(r.sequence.children), true
}

func SelectDeclarationRequests(
	requests []RootRequest,
) ([]RootRequest, error) {
	return selectRootRequests(requests, declarationRequestKindMask)
}

func SelectNonDeclarationRequests(
	requests []RootRequest,
) ([]RootRequest, error) {
	return selectRootRequests(requests, nonDeclarationRequestKindMask)
}

func selectRootRequests(
	requests []RootRequest,
	selectedKinds uint8,
) ([]RootRequest, error) {
	selected, _, err := selectRootRequestsWithWork(requests, selectedKinds)
	return selected, err
}

func selectRootRequestsWithWork(
	requests []RootRequest,
	selectedKinds uint8,
) ([]RootRequest, uint64, error) {
	memo := make(map[*rootRequestSequence]rootRequestSelection)
	selected := make([]RootRequest, 0, len(requests))
	var work uint64
	for _, request := range requests {
		selection, err := selectRootRequest(
			request,
			selectedKinds,
			memo,
			&work,
		)
		if err != nil {
			return nil, work, err
		}
		if selection.selected {
			selected = append(selected, selection.request)
		}
	}
	return slices.Clone(selected), work, nil
}

func selectRootRequest(
	request RootRequest,
	selectedKinds uint8,
	memo map[*rootRequestSequence]rootRequestSelection,
	work *uint64,
) (rootRequestSelection, error) {
	*work++
	if request.sequence == nil {
		if request.Kind() == RootRequestInvalid {
			return rootRequestSelection{}, &RootRequestError{
				Reason: "root request is invalid",
			}
		}
		return rootRequestSelection{
			request:  request,
			selected: request.rootRequestKinds()&selectedKinds != 0,
		}, nil
	}
	if selection, ok := memo[request.sequence]; ok {
		return selection, nil
	}
	if len(request.sequence.children) == 0 {
		return rootRequestSelection{}, &RootRequestError{
			Reason: "root request sequence is empty",
		}
	}
	if request.sequence.kinds&invalidRequestKindMask == 0 &&
		request.sequence.kinds&^selectedKinds == 0 {
		selection := rootRequestSelection{request: request, selected: true}
		memo[request.sequence] = selection
		return selection, nil
	}
	if request.sequence.kinds&selectedKinds == 0 &&
		request.sequence.kinds&invalidRequestKindMask == 0 {
		memo[request.sequence] = rootRequestSelection{}
		return rootRequestSelection{}, nil
	}
	children := make([]RootRequest, 0, len(request.sequence.children))
	unchanged := true
	for _, child := range request.sequence.children {
		selection, err := selectRootRequest(
			child,
			selectedKinds,
			memo,
			work,
		)
		if err != nil {
			return rootRequestSelection{}, err
		}
		if !selection.selected {
			unchanged = false
			continue
		}
		children = append(children, selection.request)
		if selection.request != child {
			unchanged = false
		}
	}
	selection := rootRequestSelection{}
	switch {
	case len(children) == 0:
	case unchanged && len(children) == len(request.sequence.children):
		selection = rootRequestSelection{request: request, selected: true}
	default:
		combined := combineRootRequests(children)
		selection = rootRequestSelection{
			request:  combined[0],
			selected: true,
		}
	}
	memo[request.sequence] = selection
	return selection, nil
}

func (r RootRequest) rootRequestKinds() uint8 {
	if r.sequence != nil {
		return r.sequence.kinds
	}
	return uint8(1) << r.Kind()
}

func NewDeclarationRequirementRequest(
	requirement DeclarationRequirement,
) (RootRequest, error) {
	if !requirement.Valid() {
		return RootRequest{}, &RootRequestError{
			Reason: "declaration requirement is invalid",
		}
	}
	return newDeclarationRequirementRequest(requirement), nil
}
