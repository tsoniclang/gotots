package api

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type PlacementKind uint8

const (
	PlacementInvalid PlacementKind = iota
	PlacementImport
	PlacementDeclarationRequirement
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

type PlacementOwner struct {
	kind                   PlacementKind
	modulePath             string
	exportedName           string
	declarationRequirement DeclarationRequirement
}

type PlacementRequest struct {
	owner           PlacementOwner
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
) (PlacementRequest, error) {
	if phase != ImportPhaseType && phase != ImportPhaseValue {
		return PlacementRequest{}, &PlacementRequestError{Reason: "invalid import phase"}
	}
	if modulePath == "" {
		return PlacementRequest{}, &PlacementRequestError{Reason: "module path is empty"}
	}
	if exportedName == "" {
		return PlacementRequest{}, &PlacementRequestError{Reason: "exported name is empty"}
	}
	if localName == "" {
		return PlacementRequest{}, &PlacementRequestError{Reason: "local name is empty"}
	}
	var propertyName tsgo.ModuleExportName
	if localName != exportedName {
		propertyName = factory.Identifier(exportedName)
	}
	return PlacementRequest{
		owner: PlacementOwner{
			kind:         PlacementImport,
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
) (PlacementRequest, error) {
	exportedName, err := PrimitiveAliasName(alias)
	if err != nil {
		return PlacementRequest{}, err
	}
	request, err := NewImportRequest(
		factory,
		ImportPhaseType,
		modulePath,
		exportedName,
		localName,
	)
	if err != nil {
		return PlacementRequest{}, err
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
) (PlacementRequest, error) {
	contract, err := RuntimeContract(symbol)
	if err != nil {
		return PlacementRequest{}, err
	}
	if !contract.AllowsImportPhase(phase) {
		return PlacementRequest{}, &PlacementRequestError{
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
		return PlacementRequest{}, err
	}
	request.runtimeSymbol = symbol
	return request, nil
}

func NewNamedStructOperationRequest(
	typeName *types.TypeName,
	operation NamedStructOperation,
) (PlacementRequest, error) {
	requirement, err := NewNamedStructOperationRequirement(typeName, operation)
	if err != nil {
		return PlacementRequest{}, err
	}
	return PlacementRequest{
		owner: PlacementOwner{
			kind:                   PlacementDeclarationRequirement,
			declarationRequirement: requirement,
		},
	}, nil
}

func (r PlacementRequest) Kind() PlacementKind {
	return r.owner.kind
}

func (r PlacementRequest) LegalScope() PlacementScope {
	if r.owner.kind == PlacementImport {
		return ScopeFileImports
	}
	if r.owner.kind == PlacementDeclarationRequirement {
		return ScopeOwningFile
	}
	return ScopeInvalid
}

func (r PlacementRequest) PreferredScope() PlacementScope {
	return r.LegalScope()
}

func (r PlacementRequest) Execution() ExecutionConstraint {
	if r.owner.kind == PlacementImport {
		return ExecutionStatic
	}
	if r.owner.kind == PlacementDeclarationRequirement {
		return ExecutionStatic
	}
	return ExecutionInvalid
}

func (r PlacementRequest) Owner() PlacementOwner {
	return r.owner
}

func (r PlacementRequest) ImportPhase() ImportPhase {
	return r.importPhase
}

func (r PlacementRequest) ModulePath() string {
	return r.owner.modulePath
}

func (r PlacementRequest) ExportedName() string {
	return r.owner.exportedName
}

func (r PlacementRequest) LocalName() string {
	return r.localName
}

func (r PlacementRequest) ModuleSpecifier() tsgo.StringLiteral {
	return r.moduleSpecifier
}

func (r PlacementRequest) Specifier() tsgo.ImportSpecifier {
	return r.specifier
}

func (r PlacementRequest) PrimitiveAlias() (PrimitiveAlias, bool) {
	if r.primitiveAlias == PrimitiveInvalid {
		return PrimitiveInvalid, false
	}
	return r.primitiveAlias, true
}

func (r PlacementRequest) RuntimeSymbol() (RuntimeSymbol, bool) {
	if r.runtimeSymbol == RuntimeInvalid {
		return RuntimeInvalid, false
	}
	return r.runtimeSymbol, true
}

func (r PlacementRequest) DeclarationRequirement() (
	DeclarationRequirement,
	bool,
) {
	requirement := r.owner.declarationRequirement
	if r.owner.kind != PlacementDeclarationRequirement ||
		!requirement.Valid() {
		return DeclarationRequirement{}, false
	}
	return requirement, true
}

type PlacementRequestError struct {
	Reason string
}

func (e *PlacementRequestError) Error() string {
	return fmt.Sprintf("create placement request: %s", e.Reason)
}
