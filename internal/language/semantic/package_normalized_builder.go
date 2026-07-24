package semantic

type normalizedPackageBuilder struct {
	identities   packageIdentityBuilder
	authorities  packageAuthorityBuilder
	definitions  packageDefinitionBuilder
	resolutions  packageResolutionBuilder
	declarations packageDeclarationBuilder
	bindings     packageBindingBuilder
	types        packageTypeBuilder
	witnesses    packageTypeWitnessBuilder
	operations   packageOperationBuilder
	unsupported  packageUnsupportedBuilder
}

func (builder *normalizedPackageBuilder) addDefinition(
	record DefinitionSemantics,
) {
	builder.definitions.add(
		&builder.identities,
		&builder.authorities,
		record,
	)
}

func (builder *normalizedPackageBuilder) addResolution(
	record OccurrenceResolution,
) {
	builder.resolutions.add(&builder.identities, record)
}

func (builder *normalizedPackageBuilder) addOperation(
	record Operation,
) {
	builder.operations.add(&builder.identities, record)
}

func (builder *normalizedPackageBuilder) addOperationSpec(
	spec OperationSpec,
) {
	builder.operations.addSpec(&builder.identities, spec)
}

func (builder *normalizedPackageBuilder) addDeclaration(
	record Declaration,
) {
	builder.declarations.add(
		&builder.identities,
		&builder.authorities,
		record,
	)
}

func (builder *normalizedPackageBuilder) addBinding(
	record Binding,
) {
	builder.bindings.add(
		&builder.identities,
		&builder.authorities,
		record,
	)
}

func (builder *normalizedPackageBuilder) addType(
	record Type,
) {
	builder.types.add(&builder.identities, record)
}

func (builder *normalizedPackageBuilder) addTypeWitness(
	witness TypeWitness,
) {
	builder.witnesses.add(
		&builder.identities,
		&builder.authorities,
		witness,
	)
}

func (builder *normalizedPackageBuilder) addUnsupported(
	record Unsupported,
) {
	builder.unsupported.add(
		&builder.identities,
		&builder.authorities,
		record,
	)
}

type normalizedPackageStores struct {
	identities   packageIdentityTable
	authorities  packageAuthorityTable
	definitions  packageDefinitionStore
	resolutions  packageResolutionStore
	declarations packageDeclarationStore
	bindings     packageBindingStore
	types        packageTypeStore
	witnesses    packageTypeWitnessStore
	operations   packageOperationStore
	unsupported  packageUnsupportedStore
}

func (builder *normalizedPackageBuilder) seal() (
	normalizedPackageStores,
	error,
) {
	identities, remap, err := builder.identities.seal()
	if err != nil {
		return normalizedPackageStores{}, err
	}
	authorities, authorityRemap := builder.authorities.seal()
	definitions, err := builder.definitions.seal(
		remap, authorityRemap,
	)
	if err != nil {
		return normalizedPackageStores{}, err
	}
	resolutions, err := builder.resolutions.seal(remap)
	if err != nil {
		return normalizedPackageStores{}, err
	}
	declarations, err := builder.declarations.seal(
		remap, authorityRemap,
	)
	if err != nil {
		return normalizedPackageStores{}, err
	}
	bindings, err := builder.bindings.seal(
		remap, authorityRemap,
	)
	if err != nil {
		return normalizedPackageStores{}, err
	}
	types, err := builder.types.seal(remap)
	if err != nil {
		return normalizedPackageStores{}, err
	}
	witnesses, err := builder.witnesses.seal(
		remap, authorityRemap,
	)
	if err != nil {
		return normalizedPackageStores{}, err
	}
	operations, err := builder.operations.seal(remap)
	if err != nil {
		return normalizedPackageStores{}, err
	}
	unsupported, err := builder.unsupported.seal(
		remap, authorityRemap,
	)
	if err != nil {
		return normalizedPackageStores{}, err
	}
	return normalizedPackageStores{
		identities:   identities,
		authorities:  authorities,
		definitions:  definitions,
		resolutions:  resolutions,
		declarations: declarations,
		bindings:     bindings,
		types:        types,
		witnesses:    witnesses,
		operations:   operations,
		unsupported:  unsupported,
	}, nil
}
