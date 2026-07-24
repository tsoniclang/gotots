package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type storedDeclaration struct {
	id           declarationRef
	pkg          packageRef
	class        identity.SemanticObjectClass
	name         string
	typeID       typeRef
	exported     bool
	constantKind ConstantKind
	constant     string
	authority    authorityRef
}

type packageDeclarationBuilder struct {
	records []storedDeclaration
}

func (builder *packageDeclarationBuilder) add(
	identities *packageIdentityBuilder,
	authorities *packageAuthorityBuilder,
	record Declaration,
) {
	builder.records = append(builder.records, storedDeclaration{
		id:           identities.declaration(record.id),
		pkg:          identities.packageID(record.pkg),
		class:        record.class,
		name:         record.name,
		typeID:       identities.typeID(record.typeID),
		exported:     record.exported,
		constantKind: record.constant.kind,
		constant:     record.constant.exact,
		authority:    authorities.authority(record.authority),
	})
}

type packageDeclarationStore struct {
	records []storedDeclaration
}

func (builder *packageDeclarationBuilder) seal(
	identities packageIdentityRemap,
	authorities []uint64,
) (packageDeclarationStore, error) {
	store := packageDeclarationStore{records: builder.records}
	var err error
	for index := range store.records {
		record := &store.records[index]
		if record.id, err = remapReference(
			record.id, identities.declarations,
		); err != nil {
			return packageDeclarationStore{}, err
		}
		if record.pkg, err = remapReference(
			record.pkg, identities.packages,
		); err != nil {
			return packageDeclarationStore{}, err
		}
		if record.typeID, err = remapReference(
			record.typeID, identities.types,
		); err != nil {
			return packageDeclarationStore{}, err
		}
		if record.authority, err = remapReference(
			record.authority, authorities,
		); err != nil {
			return packageDeclarationStore{}, err
		}
	}
	sort.Slice(store.records, func(left, right int) bool {
		return store.records[left].id < store.records[right].id
	})
	return store, nil
}

func (store packageDeclarationStore) record(
	identities *packageIdentityProjection,
	authorities packageAuthorityTable,
	index int,
) (Declaration, error) {
	if index < 0 || index >= len(store.records) {
		return Declaration{}, fmt.Errorf(
			"semantic declaration index %d is invalid", index,
		)
	}
	stored := store.records[index]
	authority, present := authorities.authority(stored.authority)
	if !present {
		return Declaration{}, fmt.Errorf(
			"semantic declaration authority reference is invalid",
		)
	}
	return Declaration{
		id:       identities.declaration(stored.id),
		pkg:      identities.packageID(stored.pkg),
		class:    stored.class,
		name:     stored.name,
		typeID:   identities.typeID(stored.typeID),
		exported: stored.exported,
		constant: Constant{
			kind:  stored.constantKind,
			exact: stored.constant,
		},
		authority: authority,
	}, nil
}

type storedBinding struct {
	id         bindingRef
	pkg        packageRef
	definition definitionRef
	role       identity.SemanticBindingRole
	name       string
	typeID     typeRef
	source     occurrenceRef
	captures   definitionRefRange
	authority  authorityRef
}

type packageBindingBuilder struct {
	records  []storedBinding
	captures []definitionRef
}

func (builder *packageBindingBuilder) add(
	identities *packageIdentityBuilder,
	authorities *packageAuthorityBuilder,
	record Binding,
) {
	captures := definitionRefRange{
		start: uint64(len(builder.captures)),
		count: uint64(len(record.captures)),
	}
	for _, capture := range record.captures {
		builder.captures = append(
			builder.captures,
			identities.definition(capture),
		)
	}
	builder.records = append(builder.records, storedBinding{
		id:         identities.binding(record.id),
		pkg:        identities.packageID(record.pkg),
		definition: identities.definition(record.definition),
		role:       record.role,
		name:       record.name,
		typeID:     identities.typeID(record.typeID),
		source:     identities.occurrence(record.source),
		captures:   captures,
		authority:  authorities.authority(record.authority),
	})
}

type packageBindingStore struct {
	records  []storedBinding
	captures []definitionRef
}

func (builder *packageBindingBuilder) seal(
	identities packageIdentityRemap,
	authorities []uint64,
) (packageBindingStore, error) {
	store := packageBindingStore{
		records:  builder.records,
		captures: builder.captures,
	}
	var err error
	for index := range store.records {
		record := &store.records[index]
		if record.id, err = remapReference(
			record.id, identities.bindings,
		); err != nil {
			return packageBindingStore{}, err
		}
		if record.pkg, err = remapReference(
			record.pkg, identities.packages,
		); err != nil {
			return packageBindingStore{}, err
		}
		if record.definition, err = remapReference(
			record.definition, identities.definitions,
		); err != nil {
			return packageBindingStore{}, err
		}
		if record.typeID, err = remapReference(
			record.typeID, identities.types,
		); err != nil {
			return packageBindingStore{}, err
		}
		if record.source, err = remapReference(
			record.source, identities.occurrences,
		); err != nil {
			return packageBindingStore{}, err
		}
		if record.authority, err = remapReference(
			record.authority, authorities,
		); err != nil {
			return packageBindingStore{}, err
		}
	}
	if err := remapReferences(
		store.captures, identities.definitions,
	); err != nil {
		return packageBindingStore{}, err
	}
	sort.Slice(store.records, func(left, right int) bool {
		return store.records[left].id < store.records[right].id
	})
	return store, nil
}

func (store packageBindingStore) record(
	identities *packageIdentityProjection,
	authorities packageAuthorityTable,
	index int,
) (Binding, error) {
	if index < 0 || index >= len(store.records) {
		return Binding{}, fmt.Errorf(
			"semantic binding index %d is invalid", index,
		)
	}
	stored := store.records[index]
	authority, present := authorities.authority(stored.authority)
	if !present {
		return Binding{}, fmt.Errorf(
			"semantic binding authority reference is invalid",
		)
	}
	captures, err := relationValues(
		store.captures,
		stored.captures.start,
		stored.captures.count,
		identities.definition,
	)
	if err != nil {
		return Binding{}, err
	}
	return Binding{
		id:         identities.binding(stored.id),
		pkg:        identities.packageID(stored.pkg),
		definition: identities.definition(stored.definition),
		role:       stored.role,
		name:       stored.name,
		typeID:     identities.typeID(stored.typeID),
		source:     identities.occurrence(stored.source),
		captures:   captures,
		authority:  authority,
	}, nil
}

type storedTypeWitness struct {
	pkg       packageRef
	typeID    typeRef
	authority authorityRef
}

type packageTypeWitnessBuilder struct {
	records []storedTypeWitness
}

func (builder *packageTypeWitnessBuilder) add(
	identities *packageIdentityBuilder,
	authorities *packageAuthorityBuilder,
	record TypeWitness,
) {
	builder.records = append(builder.records, storedTypeWitness{
		pkg:       identities.packageID(record.pkg),
		typeID:    identities.typeID(record.typeID),
		authority: authorities.authority(record.authority),
	})
}

type packageTypeWitnessStore struct {
	records []storedTypeWitness
}

func (builder *packageTypeWitnessBuilder) seal(
	identities packageIdentityRemap,
	authorities []uint64,
) (packageTypeWitnessStore, error) {
	store := packageTypeWitnessStore{records: builder.records}
	var err error
	for index := range store.records {
		record := &store.records[index]
		if record.pkg, err = remapReference(
			record.pkg, identities.packages,
		); err != nil {
			return packageTypeWitnessStore{}, err
		}
		if record.typeID, err = remapReference(
			record.typeID, identities.types,
		); err != nil {
			return packageTypeWitnessStore{}, err
		}
		if record.authority, err = remapReference(
			record.authority, authorities,
		); err != nil {
			return packageTypeWitnessStore{}, err
		}
	}
	sort.Slice(store.records, func(left, right int) bool {
		return store.records[left].typeID <
			store.records[right].typeID
	})
	return store, nil
}

func (store packageTypeWitnessStore) record(
	identities *packageIdentityProjection,
	authorities packageAuthorityTable,
	index int,
) (TypeWitness, error) {
	if index < 0 || index >= len(store.records) {
		return TypeWitness{}, fmt.Errorf(
			"semantic type-witness index %d is invalid", index,
		)
	}
	stored := store.records[index]
	authority, present := authorities.authority(stored.authority)
	if !present {
		return TypeWitness{}, fmt.Errorf(
			"semantic type-witness authority reference is invalid",
		)
	}
	return TypeWitness{
		pkg:       identities.packageID(stored.pkg),
		typeID:    identities.typeID(stored.typeID),
		authority: authority,
	}, nil
}
