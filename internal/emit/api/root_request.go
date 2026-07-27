package api

import (
	"fmt"
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

type RootRequestOwner struct {
	kind                   RootRequestKind
	modulePath             string
	exportedName           string
	declarationRequirement DeclarationRequirement
	artifactDependency     ArtifactDependency
}

type RootRequest struct {
	owner           RootRequestOwner
	importPhase     ImportPhase
	localName       string
	moduleSpecifier tsgo.StringLiteral
	specifier       tsgo.ImportSpecifier
	primitiveAlias  PrimitiveAlias
	runtimeSymbol   RuntimeSymbol
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
	return RootRequest{
		owner: RootRequestOwner{
			kind:         RootRequestImport,
			modulePath:   modulePath,
			exportedName: exportedName,
		},
		importPhase:     phase,
		localName:       localName,
		moduleSpecifier: factory.StringLiteral(modulePath, tsgo.TokenFlagsNone),
		specifier: factory.ImportSpecifier(
			false,
			propertyName,
			factory.Identifier(localName),
		),
	}, nil
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
	request.primitiveAlias = alias
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
	request.runtimeSymbol = symbol
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
	return RootRequest{
		owner: RootRequestOwner{
			kind:                   RootRequestDeclarationRequirement,
			declarationRequirement: requirement,
		},
	}, nil
}

func NewArtifactDependencyRequest(
	provider types.Object,
	facet ArtifactFacet,
) (RootRequest, error) {
	dependency, err := NewArtifactDependency(provider, facet)
	if err != nil {
		return RootRequest{}, err
	}
	return RootRequest{
		owner: RootRequestOwner{
			kind:               RootRequestArtifactDependency,
			artifactDependency: dependency,
		},
	}, nil
}

func (r RootRequest) Kind() RootRequestKind {
	return r.owner.kind
}

func (r RootRequest) LegalScope() PlacementScope {
	if r.owner.kind == RootRequestImport {
		return ScopeFileImports
	}
	if r.owner.kind == RootRequestDeclarationRequirement {
		return ScopeOwningFile
	}
	return ScopeInvalid
}

func (r RootRequest) PreferredScope() PlacementScope {
	return r.LegalScope()
}

func (r RootRequest) Execution() ExecutionConstraint {
	if r.owner.kind == RootRequestImport {
		return ExecutionStatic
	}
	if r.owner.kind == RootRequestDeclarationRequirement {
		return ExecutionStatic
	}
	return ExecutionInvalid
}

func (r RootRequest) Owner() RootRequestOwner {
	return r.owner
}

func (r RootRequest) ImportPhase() ImportPhase {
	return r.importPhase
}

func (r RootRequest) ModulePath() string {
	return r.owner.modulePath
}

func (r RootRequest) ExportedName() string {
	return r.owner.exportedName
}

func (r RootRequest) LocalName() string {
	return r.localName
}

func (r RootRequest) ModuleSpecifier() tsgo.StringLiteral {
	return r.moduleSpecifier
}

func (r RootRequest) Specifier() tsgo.ImportSpecifier {
	return r.specifier
}

func (r RootRequest) PrimitiveAlias() (PrimitiveAlias, bool) {
	if r.primitiveAlias == PrimitiveInvalid {
		return PrimitiveInvalid, false
	}
	return r.primitiveAlias, true
}

func (r RootRequest) RuntimeSymbol() (RuntimeSymbol, bool) {
	if r.runtimeSymbol == RuntimeInvalid {
		return RuntimeInvalid, false
	}
	return r.runtimeSymbol, true
}

func (r RootRequest) DeclarationRequirement() (
	DeclarationRequirement,
	bool,
) {
	requirement := r.owner.declarationRequirement
	if r.owner.kind != RootRequestDeclarationRequirement ||
		!requirement.Valid() {
		return DeclarationRequirement{}, false
	}
	return requirement, true
}

func (r RootRequest) ArtifactDependency() (ArtifactDependency, bool) {
	dependency := r.owner.artifactDependency
	if r.owner.kind != RootRequestArtifactDependency ||
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
