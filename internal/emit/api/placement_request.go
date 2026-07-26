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
	PlacementCompanion
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
	kind         PlacementKind
	modulePath   string
	exportedName string
	companion    CompanionOwner
}

type PlacementRequest struct {
	owner           PlacementOwner
	importPhase     ImportPhase
	localName       string
	moduleSpecifier tsgo.StringLiteral
	specifier       tsgo.ImportSpecifier
	primitiveAlias  PrimitiveAlias
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

func NewCompanionRequest(
	typeName *types.TypeName,
	operation CompanionOperation,
) (PlacementRequest, error) {
	owner, err := NewCompanionOwner(typeName, operation)
	if err != nil {
		return PlacementRequest{}, err
	}
	return PlacementRequest{
		owner: PlacementOwner{
			kind:      PlacementCompanion,
			companion: owner,
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
	if r.owner.kind == PlacementCompanion {
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
	if r.owner.kind == PlacementCompanion {
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

func (r PlacementRequest) Companion() (CompanionOwner, bool) {
	if r.owner.kind != PlacementCompanion ||
		r.owner.companion.TypeName() == nil ||
		!r.owner.companion.Operation().Valid() {
		return CompanionOwner{}, false
	}
	return r.owner.companion, true
}

type PlacementRequestError struct {
	Reason string
}

func (e *PlacementRequestError) Error() string {
	return fmt.Sprintf("create placement request: %s", e.Reason)
}
