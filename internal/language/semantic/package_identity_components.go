package semantic

import (
	"github.com/tsoniclang/gotots/internal/identity"
)

type moduleRef uint64
type ownerRef uint64
type packageRef uint64
type fileRef uint64
type spanRef uint64
type definitionRef uint64
type occurrenceRef uint64
type declarationRef uint64
type bindingRef uint64
type typeRef uint64
type operationRef uint64
type unsupportedRef uint64

type storedModuleIdentity struct {
	path    string
	version string
}

type storedOwnerIdentity struct {
	class  identity.OwnerClass
	module moduleRef
}

type storedPackageIdentity struct {
	owner      ownerRef
	importPath string
}

type storedFileIdentity struct {
	owner ownerRef
	rel   string
}

type storedSpanIdentity struct {
	file  fileRef
	start int
	end   int
}

type storedOccurrenceIdentity struct {
	span spanRef
	kind uint16
}

type storedDefinitionIdentity struct {
	kind      identity.DefinitionKind
	root      occurrenceRef
	pkg       packageRef
	implicit  identity.ImplicitDefinitionOp
	synthetic identity.SyntheticDefinitionRole
	name      string
}

type storedTypeIdentity struct {
	digest string
}

type storedDeclarationIdentity struct {
	form        identity.SemanticDeclarationForm
	pkg         packageRef
	ownerType   typeRef
	memberPkg   packageRef
	class       identity.SemanticObjectClass
	name        string
	ordinal     int
	predeclared uint16
	owner       occurrenceRef
	occurrence  occurrenceRef
}

type storedBindingIdentity struct {
	owner       occurrenceRef
	declaration occurrenceRef
	role        identity.SemanticBindingRole
	ordinal     int
}

type storedOperationIdentity struct {
	definition definitionRef
	occurrence occurrenceRef
	implicit   identity.ImplicitDefinitionOp
	ordinal    int
}

type storedUnsupportedIdentity struct {
	definition definitionRef
	occurrence occurrenceRef
}

type packageIdentityComponents struct {
	modules      []storedModuleIdentity
	owners       []storedOwnerIdentity
	packages     []storedPackageIdentity
	files        []storedFileIdentity
	spans        []storedSpanIdentity
	occurrences  []storedOccurrenceIdentity
	definitions  []storedDefinitionIdentity
	types        []storedTypeIdentity
	declarations []storedDeclarationIdentity
	bindings     []storedBindingIdentity
	operations   []storedOperationIdentity
	unsupported  []storedUnsupportedIdentity
}

type componentPool[Record comparable] struct {
	records []Record
	index   map[Record]uint64
}

func (pool *componentPool[Record]) intern(record Record) uint64 {
	if index, present := pool.index[record]; present {
		return index
	}
	if pool.index == nil {
		pool.index = map[Record]uint64{}
	}
	index := uint64(len(pool.records) + 1)
	pool.records = append(pool.records, record)
	pool.index[record] = index
	return index
}

type packageIdentityBuilder struct {
	modules      componentPool[storedModuleIdentity]
	owners       componentPool[storedOwnerIdentity]
	packages     componentPool[storedPackageIdentity]
	files        componentPool[storedFileIdentity]
	spans        componentPool[storedSpanIdentity]
	occurrences  componentPool[storedOccurrenceIdentity]
	definitions  componentPool[storedDefinitionIdentity]
	types        componentPool[storedTypeIdentity]
	declarations componentPool[storedDeclarationIdentity]
	bindings     componentPool[storedBindingIdentity]
	operations   componentPool[storedOperationIdentity]
	unsupported  componentPool[storedUnsupportedIdentity]
}

func (builder *packageIdentityBuilder) projectionTable() packageIdentityTable {
	if builder == nil {
		return packageIdentityTable{}
	}
	return packageIdentityTable{
		packageIdentityComponents: packageIdentityComponents{
			modules:      builder.modules.records,
			owners:       builder.owners.records,
			packages:     builder.packages.records,
			files:        builder.files.records,
			spans:        builder.spans.records,
			occurrences:  builder.occurrences.records,
			definitions:  builder.definitions.records,
			types:        builder.types.records,
			declarations: builder.declarations.records,
			bindings:     builder.bindings.records,
			operations:   builder.operations.records,
			unsupported:  builder.unsupported.records,
		},
	}
}

func (builder *packageIdentityBuilder) module(
	id identity.ModuleID,
) moduleRef {
	if id.IsZero() {
		return 0
	}
	return moduleRef(builder.modules.intern(storedModuleIdentity{
		path: id.Path(), version: id.Version(),
	}))
}

func (builder *packageIdentityBuilder) owner(
	value identity.Owner,
) ownerRef {
	if value.IsZero() {
		return 0
	}
	return ownerRef(builder.owners.intern(storedOwnerIdentity{
		class:  value.Class(),
		module: builder.module(value.Module()),
	}))
}

func (builder *packageIdentityBuilder) packageID(
	id identity.PackageID,
) packageRef {
	if id.IsZero() {
		return 0
	}
	return packageRef(builder.packages.intern(storedPackageIdentity{
		owner:      builder.owner(id.Owner()),
		importPath: id.ImportPath(),
	}))
}

func (builder *packageIdentityBuilder) file(
	id identity.FileID,
) fileRef {
	if id.IsZero() {
		return 0
	}
	return fileRef(builder.files.intern(storedFileIdentity{
		owner: builder.owner(id.Owner()),
		rel:   id.Rel(),
	}))
}

func (builder *packageIdentityBuilder) span(
	id identity.SpanID,
) spanRef {
	if id.IsZero() {
		return 0
	}
	return spanRef(builder.spans.intern(storedSpanIdentity{
		file:  builder.file(id.File()),
		start: id.Start(),
		end:   id.End(),
	}))
}

func (builder *packageIdentityBuilder) occurrence(
	id identity.OccurrenceID,
) occurrenceRef {
	if id.IsZero() {
		return 0
	}
	return occurrenceRef(builder.occurrences.intern(
		storedOccurrenceIdentity{
			span: builder.span(id.Span()),
			kind: id.KindID(),
		},
	))
}

func (builder *packageIdentityBuilder) definition(
	id identity.DefinitionID,
) definitionRef {
	if id.IsZero() {
		return 0
	}
	return definitionRef(builder.definitions.intern(
		storedDefinitionIdentity{
			kind:      id.Kind(),
			root:      builder.occurrence(id.Root()),
			pkg:       builder.packageID(id.Package()),
			implicit:  id.ImplicitOp(),
			synthetic: id.SyntheticRole(),
			name:      id.SyntheticName(),
		},
	))
}

func (builder *packageIdentityBuilder) typeID(
	id identity.SemanticTypeID,
) typeRef {
	if id.IsZero() {
		return 0
	}
	return typeRef(builder.types.intern(storedTypeIdentity{
		digest: id.Digest(),
	}))
}

func (builder *packageIdentityBuilder) declaration(
	id identity.SemanticDeclarationID,
) declarationRef {
	if id.IsZero() {
		return 0
	}
	return declarationRef(builder.declarations.intern(
		storedDeclarationIdentity{
			form:        id.Form(),
			pkg:         builder.packageID(id.Package()),
			ownerType:   builder.typeID(id.OwnerType()),
			memberPkg:   builder.packageID(id.MemberPackage()),
			class:       id.Class(),
			name:        id.Name(),
			ordinal:     id.Ordinal(),
			predeclared: id.Predeclared(),
			owner: builder.occurrence(
				id.OwnerOccurrence(),
			),
			occurrence: builder.occurrence(
				id.Occurrence(),
			),
		},
	))
}

func (builder *packageIdentityBuilder) binding(
	id identity.SemanticBindingID,
) bindingRef {
	if id.IsZero() {
		return 0
	}
	return bindingRef(builder.bindings.intern(storedBindingIdentity{
		owner: builder.occurrence(id.Owner()),
		declaration: builder.occurrence(
			id.Declaration(),
		),
		role:    id.Role(),
		ordinal: id.Ordinal(),
	}))
}

func (builder *packageIdentityBuilder) operation(
	id identity.OperationID,
) operationRef {
	if id.IsZero() {
		return 0
	}
	return operationRef(builder.operations.intern(
		storedOperationIdentity{
			definition: builder.definition(
				id.Definition(),
			),
			occurrence: builder.occurrence(
				id.Occurrence(),
			),
			implicit: id.ImplicitOp(),
			ordinal:  id.Ordinal(),
		},
	))
}

func (builder *packageIdentityBuilder) unsupportedID(
	id identity.UnsupportedID,
) unsupportedRef {
	if id.IsZero() {
		return 0
	}
	return unsupportedRef(builder.unsupported.intern(
		storedUnsupportedIdentity{
			definition: builder.definition(
				id.Definition(),
			),
			occurrence: builder.occurrence(
				id.Occurrence(),
			),
		},
	))
}
