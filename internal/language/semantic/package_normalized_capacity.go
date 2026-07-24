package semantic

import "fmt"

func reserveValues[Value any](
	values []Value,
	additional int,
) []Value {
	if additional <= cap(values)-len(values) {
		return values
	}
	capacity := len(values) + additional
	out := make([]Value, len(values), capacity)
	copy(out, values)
	return out
}

func (pool *componentPool[Record]) reserve(additional int) {
	pool.records = reserveValues(pool.records, additional)
	if pool.index == nil {
		pool.index = make(map[Record]uint64, additional)
	}
}

func (builder *normalizedPackageBuilder) reserve(
	counts semanticShardCounts,
) error {
	identityCounts := []struct {
		name  string
		count uint64
	}{
		{name: "modules", count: counts.Modules},
		{name: "owners", count: counts.Owners},
		{name: "packages", count: counts.Packages},
		{name: "files", count: counts.Files},
		{name: "spans", count: counts.Spans},
		{name: "occurrences", count: counts.Occurrences},
		{name: "definitions", count: counts.Definitions},
		{name: "types", count: counts.Types},
		{name: "declarations", count: counts.Declarations},
		{name: "bindings", count: counts.Bindings},
		{name: "operations", count: counts.Operations},
		{name: "unsupported", count: counts.Unsupported},
	}
	bounded := make([]int, len(identityCounts))
	for index, current := range identityCounts {
		count, err := boundedShardCount(
			current.name, current.count,
		)
		if err != nil {
			return err
		}
		bounded[index] = count
	}
	builder.identities.modules.reserve(bounded[0])
	builder.identities.owners.reserve(bounded[1])
	builder.identities.packages.reserve(bounded[2])
	builder.identities.files.reserve(bounded[3])
	builder.identities.spans.reserve(bounded[4])
	builder.identities.occurrences.reserve(bounded[5])
	builder.identities.definitions.reserve(bounded[6])
	builder.identities.types.reserve(bounded[7])
	builder.identities.declarations.reserve(bounded[8])
	builder.identities.bindings.reserve(bounded[9])
	builder.identities.operations.reserve(bounded[10])
	builder.identities.unsupported.reserve(bounded[11])
	if err := builder.reserveRecords(counts); err != nil {
		return fmt.Errorf("reserve semantic records: %w", err)
	}
	return nil
}

func (builder *normalizedPackageBuilder) reserveRecords(
	counts semanticShardCounts,
) error {
	definitions, err := boundedShardCount(
		"definition records", counts.DefinitionRecords,
	)
	if err != nil {
		return err
	}
	resolutions, err := boundedShardCount(
		"resolution records", counts.ResolutionRecords,
	)
	if err != nil {
		return err
	}
	declarations, err := boundedShardCount(
		"declaration records", counts.DeclarationRecords,
	)
	if err != nil {
		return err
	}
	bindings, err := boundedShardCount(
		"binding records", counts.BindingRecords,
	)
	if err != nil {
		return err
	}
	types, err := boundedShardCount(
		"type records", counts.TypeRecords,
	)
	if err != nil {
		return err
	}
	operations, err := boundedShardCount(
		"operation records", counts.OperationRecords,
	)
	if err != nil {
		return err
	}
	unsupported, err := boundedShardCount(
		"unsupported records", counts.UnsupportedRecords,
	)
	if err != nil {
		return err
	}
	builder.definitions.records = reserveValues(
		builder.definitions.records, definitions,
	)
	builder.resolutions.records = reserveValues(
		builder.resolutions.records, resolutions,
	)
	builder.declarations.records = reserveValues(
		builder.declarations.records, declarations,
	)
	builder.bindings.records = reserveValues(
		builder.bindings.records, bindings,
	)
	builder.types.records = reserveValues(
		builder.types.records, types,
	)
	builder.witnesses.records = reserveValues(
		builder.witnesses.records, types,
	)
	builder.operations.records = reserveValues(
		builder.operations.records, operations,
	)
	builder.unsupported.records = reserveValues(
		builder.unsupported.records, unsupported,
	)
	return nil
}
