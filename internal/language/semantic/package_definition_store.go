package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type declarationRefRange struct {
	start uint64
	count uint64
}

type bindingRefRange struct {
	start uint64
	count uint64
}

type storedDefinition struct {
	id        definitionRef
	pkg       packageRef
	form      DefinitionForm
	authority authorityRef
	name      string
	bindings  bindingRefRange
	payload   uint64
}

type storedCallableDefinition struct {
	declarations declarationRefRange
	signature    typeRef
	receiver     bindingRef
}

type storedInitializerDefinition struct {
	declarations declarationRefRange
	entries      occurrenceRefRange
}

type storedBodylessDefinition struct {
	declaration declarationRef
	signature   typeRef
	receiver    bindingRef
}

type storedImplicitDefinition struct {
	operation identity.ImplicitDefinitionOp
}

type storedSyntheticDefinition struct {
	declaration declarationRef
	signature   typeRef
}

type packageDefinitionBuilder struct {
	records              []storedDefinition
	callable             []storedCallableDefinition
	initializers         []storedInitializerDefinition
	bodyless             []storedBodylessDefinition
	implicit             []storedImplicitDefinition
	synthetic            []storedSyntheticDefinition
	declarationRelations []declarationRef
	bindingRelations     []bindingRef
	initializerEntries   []occurrenceRef
}

func (builder *packageDefinitionBuilder) add(
	identities *packageIdentityBuilder,
	authorities *packageAuthorityBuilder,
	record DefinitionSemantics,
) {
	spec := record.spec
	stored := storedDefinition{
		id:        identities.definition(spec.Definition),
		pkg:       identities.packageID(spec.Package),
		form:      spec.Form,
		authority: authorities.authority(spec.Authority),
		name:      spec.Name,
		bindings: builder.addBindings(
			identities, spec.Bindings,
		),
	}
	switch spec.Form {
	case DefinitionFormCallable:
		builder.callable = append(
			builder.callable,
			storedCallableDefinition{
				declarations: builder.addDeclarations(
					identities, spec.Declarations,
				),
				signature: identities.typeID(spec.Signature),
				receiver:  identities.binding(spec.Receiver),
			},
		)
		stored.payload = uint64(len(builder.callable))
	case DefinitionFormInitializer:
		builder.initializers = append(
			builder.initializers,
			storedInitializerDefinition{
				declarations: builder.addDeclarations(
					identities, spec.Declarations,
				),
				entries: builder.addInitializers(
					identities, spec.InitializerEntries,
				),
			},
		)
		stored.payload = uint64(len(builder.initializers))
	case DefinitionFormBodyless:
		builder.bodyless = append(
			builder.bodyless,
			storedBodylessDefinition{
				declaration: identities.declaration(
					spec.Declarations[0],
				),
				signature: identities.typeID(spec.Signature),
				receiver:  identities.binding(spec.Receiver),
			},
		)
		stored.payload = uint64(len(builder.bodyless))
	case DefinitionFormImplicit:
		builder.implicit = append(
			builder.implicit,
			storedImplicitDefinition{operation: spec.Implicit},
		)
		stored.payload = uint64(len(builder.implicit))
	case DefinitionFormSynthetic:
		builder.synthetic = append(
			builder.synthetic,
			storedSyntheticDefinition{
				declaration: identities.declaration(
					spec.Declarations[0],
				),
				signature: identities.typeID(spec.Signature),
			},
		)
		stored.payload = uint64(len(builder.synthetic))
	}
	builder.records = append(builder.records, stored)
}

func (builder *packageDefinitionBuilder) addDeclarations(
	identities *packageIdentityBuilder,
	values []identity.SemanticDeclarationID,
) declarationRefRange {
	out := declarationRefRange{
		start: uint64(len(builder.declarationRelations)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.declarationRelations = append(
			builder.declarationRelations,
			identities.declaration(value),
		)
	}
	return out
}

func (builder *packageDefinitionBuilder) addBindings(
	identities *packageIdentityBuilder,
	values []identity.SemanticBindingID,
) bindingRefRange {
	out := bindingRefRange{
		start: uint64(len(builder.bindingRelations)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.bindingRelations = append(
			builder.bindingRelations,
			identities.binding(value),
		)
	}
	return out
}

func (builder *packageDefinitionBuilder) addInitializers(
	identities *packageIdentityBuilder,
	values []identity.OccurrenceID,
) occurrenceRefRange {
	out := occurrenceRefRange{
		start: uint64(len(builder.initializerEntries)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.initializerEntries = append(
			builder.initializerEntries,
			identities.occurrence(value),
		)
	}
	return out
}

type packageDefinitionStore struct {
	records              []storedDefinition
	callable             []storedCallableDefinition
	initializers         []storedInitializerDefinition
	bodyless             []storedBodylessDefinition
	implicit             []storedImplicitDefinition
	synthetic            []storedSyntheticDefinition
	declarationRelations []declarationRef
	bindingRelations     []bindingRef
	initializerEntries   []occurrenceRef
}

func (builder *packageDefinitionBuilder) seal(
	identities packageIdentityRemap,
	authorities []uint64,
) (packageDefinitionStore, error) {
	store := packageDefinitionStore{
		records:              builder.records,
		callable:             builder.callable,
		initializers:         builder.initializers,
		bodyless:             builder.bodyless,
		implicit:             builder.implicit,
		synthetic:            builder.synthetic,
		declarationRelations: builder.declarationRelations,
		bindingRelations:     builder.bindingRelations,
		initializerEntries:   builder.initializerEntries,
	}
	var err error
	for index := range store.records {
		record := &store.records[index]
		if record.id, err = remapReference(
			record.id, identities.definitions,
		); err != nil {
			return packageDefinitionStore{}, err
		}
		if record.pkg, err = remapReference(
			record.pkg, identities.packages,
		); err != nil {
			return packageDefinitionStore{}, err
		}
		if record.authority, err = remapReference(
			record.authority, authorities,
		); err != nil {
			return packageDefinitionStore{}, err
		}
	}
	for index := range store.callable {
		payload := &store.callable[index]
		if payload.signature, err = remapReference(
			payload.signature, identities.types,
		); err != nil {
			return packageDefinitionStore{}, err
		}
		if payload.receiver, err = remapReference(
			payload.receiver, identities.bindings,
		); err != nil {
			return packageDefinitionStore{}, err
		}
	}
	for index := range store.bodyless {
		payload := &store.bodyless[index]
		if payload.declaration, err = remapReference(
			payload.declaration, identities.declarations,
		); err != nil {
			return packageDefinitionStore{}, err
		}
		if payload.signature, err = remapReference(
			payload.signature, identities.types,
		); err != nil {
			return packageDefinitionStore{}, err
		}
		if payload.receiver, err = remapReference(
			payload.receiver, identities.bindings,
		); err != nil {
			return packageDefinitionStore{}, err
		}
	}
	for index := range store.synthetic {
		payload := &store.synthetic[index]
		if payload.declaration, err = remapReference(
			payload.declaration, identities.declarations,
		); err != nil {
			return packageDefinitionStore{}, err
		}
		if payload.signature, err = remapReference(
			payload.signature, identities.types,
		); err != nil {
			return packageDefinitionStore{}, err
		}
	}
	if err := remapReferences(
		store.declarationRelations, identities.declarations,
	); err != nil {
		return packageDefinitionStore{}, err
	}
	if err := remapReferences(
		store.bindingRelations, identities.bindings,
	); err != nil {
		return packageDefinitionStore{}, err
	}
	if err := remapReferences(
		store.initializerEntries, identities.occurrences,
	); err != nil {
		return packageDefinitionStore{}, err
	}
	sort.Slice(store.records, func(left, right int) bool {
		return store.records[left].id < store.records[right].id
	})
	return store, nil
}

func relationValues[Stored any, Value any](
	values []Stored,
	start uint64,
	count uint64,
	project func(Stored) Value,
) ([]Value, error) {
	if start > uint64(len(values)) ||
		count > uint64(len(values))-start {
		return nil, fmt.Errorf(
			"semantic relation range %d+%d exceeds %d",
			start, count, len(values),
		)
	}
	out := make([]Value, int(count))
	for index, reference := range values[start : start+count] {
		out[index] = project(reference)
	}
	return out, nil
}

func (store packageDefinitionStore) record(
	identities *packageIdentityProjection,
	authorities packageAuthorityTable,
	index int,
) (DefinitionSemantics, error) {
	if index < 0 || index >= len(store.records) {
		return DefinitionSemantics{}, fmt.Errorf(
			"semantic definition index %d is invalid", index,
		)
	}
	stored := store.records[index]
	authority, present := authorities.authority(stored.authority)
	if !present {
		return DefinitionSemantics{}, fmt.Errorf(
			"semantic definition authority reference is invalid",
		)
	}
	spec := DefinitionSemanticsSpec{
		Definition: identities.definition(stored.id),
		Package:    identities.packageID(stored.pkg),
		Form:       stored.form,
		Authority:  authority,
		Name:       stored.name,
	}
	var err error
	spec.Bindings, err = relationValues(
		store.bindingRelations,
		stored.bindings.start,
		stored.bindings.count,
		identities.binding,
	)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	switch stored.form {
	case DefinitionFormCallable:
		payload, err := payloadAt(store.callable, stored.payload)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.Declarations, err = relationValues(
			store.declarationRelations,
			payload.declarations.start,
			payload.declarations.count,
			identities.declaration,
		)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.Signature = identities.typeID(payload.signature)
		spec.Receiver = identities.binding(payload.receiver)
	case DefinitionFormInitializer:
		payload, err := payloadAt(store.initializers, stored.payload)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.Declarations, err = relationValues(
			store.declarationRelations,
			payload.declarations.start,
			payload.declarations.count,
			identities.declaration,
		)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.InitializerEntries, err = relationValues(
			store.initializerEntries,
			payload.entries.start,
			payload.entries.count,
			identities.occurrence,
		)
		if err != nil {
			return DefinitionSemantics{}, err
		}
	case DefinitionFormBodyless:
		payload, err := payloadAt(store.bodyless, stored.payload)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.Declarations = []identity.SemanticDeclarationID{
			identities.declaration(payload.declaration),
		}
		spec.Signature = identities.typeID(payload.signature)
		spec.Receiver = identities.binding(payload.receiver)
	case DefinitionFormImplicit:
		payload, err := payloadAt(store.implicit, stored.payload)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.Implicit = payload.operation
	case DefinitionFormSynthetic:
		payload, err := payloadAt(store.synthetic, stored.payload)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.Declarations = []identity.SemanticDeclarationID{
			identities.declaration(payload.declaration),
		}
		spec.Signature = identities.typeID(payload.signature)
	default:
		return DefinitionSemantics{}, fmt.Errorf(
			"semantic definition has invalid form %d", stored.form,
		)
	}
	return DefinitionSemantics{spec: spec}, nil
}
