package semantic

import "sort"

func (builder *packageTypeBuilder) seal(
	remap packageIdentityRemap,
) (packageTypeStore, error) {
	store := packageTypeStore{
		records:       builder.records,
		basic:         builder.basic,
		nominal:       builder.nominal,
		parameters:    builder.parameters,
		elements:      builder.elements,
		arrays:        builder.arrays,
		maps:          builder.maps,
		channels:      builder.channels,
		signatures:    builder.signatures,
		structs:       builder.structs,
		interfaces:    builder.interfaces,
		tuples:        builder.tuples,
		unions:        builder.unions,
		typeRelations: builder.typeRelations,
		fields:        builder.fields,
		methods:       builder.methods,
		terms:         builder.terms,
	}
	var err error
	for index := range store.records {
		if store.records[index].id, err = remapReference(
			store.records[index].id, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	for index := range store.nominal {
		record := &store.nominal[index]
		if record.declaration, err = remapReference(
			record.declaration, remap.declarations,
		); err != nil {
			return packageTypeStore{}, err
		}
		if record.target, err = remapReference(
			record.target, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	for index := range store.parameters {
		record := &store.parameters[index]
		if record.declaration, err = remapReference(
			record.declaration, remap.declarations,
		); err != nil {
			return packageTypeStore{}, err
		}
		if record.definition, err = remapReference(
			record.definition, remap.definitions,
		); err != nil {
			return packageTypeStore{}, err
		}
		if record.constraint, err = remapReference(
			record.constraint, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	if err := remapReferences(store.elements, remap.types); err != nil {
		return packageTypeStore{}, err
	}
	for index := range store.arrays {
		if store.arrays[index].element, err = remapReference(
			store.arrays[index].element, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	for index := range store.maps {
		record := &store.maps[index]
		if record.key, err = remapReference(
			record.key, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
		if record.element, err = remapReference(
			record.element, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	for index := range store.channels {
		if store.channels[index].element, err = remapReference(
			store.channels[index].element, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	for index := range store.signatures {
		if store.signatures[index].receiver, err = remapReference(
			store.signatures[index].receiver, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	if err := remapReferences(
		store.typeRelations, remap.types,
	); err != nil {
		return packageTypeStore{}, err
	}
	for index := range store.fields {
		record := &store.fields[index]
		if record.pkg, err = remapReference(
			record.pkg, remap.packages,
		); err != nil {
			return packageTypeStore{}, err
		}
		if record.typeID, err = remapReference(
			record.typeID, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	for index := range store.methods {
		record := &store.methods[index]
		if record.pkg, err = remapReference(
			record.pkg, remap.packages,
		); err != nil {
			return packageTypeStore{}, err
		}
		if record.signature, err = remapReference(
			record.signature, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	for index := range store.terms {
		if store.terms[index].typeID, err = remapReference(
			store.terms[index].typeID, remap.types,
		); err != nil {
			return packageTypeStore{}, err
		}
	}
	sort.Slice(store.records, func(left, right int) bool {
		return store.records[left].id < store.records[right].id
	})
	return store, nil
}
