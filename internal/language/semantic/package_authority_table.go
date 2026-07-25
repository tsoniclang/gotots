package semantic

type authorityRef uint64

type packageAuthorityBuilder struct {
	records componentPool[Authority]
}

func (builder *packageAuthorityBuilder) authority(
	value Authority,
) authorityRef {
	if !value.Valid() {
		return 0
	}
	return authorityRef(builder.records.intern(value))
}

type packageAuthorityTable struct {
	records []Authority
}

func (builder *packageAuthorityBuilder) seal() (
	packageAuthorityTable,
	[]uint64,
) {
	records, remap := canonicalizeComponents(
		builder.records.records,
		func(left, right Authority) bool {
			if left.kind != right.kind {
				return left.kind < right.kind
			}
			if left.toolchainDigest != right.toolchainDigest {
				return left.toolchainDigest < right.toolchainDigest
			}
			if left.configuration != right.configuration {
				return left.configuration < right.configuration
			}
			if left.packageInput != right.packageInput {
				return left.packageInput < right.packageInput
			}
			if left.structureDigest != right.structureDigest {
				return left.structureDigest < right.structureDigest
			}
			if left.selectionDigest != right.selectionDigest {
				return left.selectionDigest < right.selectionDigest
			}
			if left.artifactDigest != right.artifactDigest {
				return left.artifactDigest < right.artifactDigest
			}
			if left.shardDigest != right.shardDigest {
				return left.shardDigest < right.shardDigest
			}
			return left.structuralSource < right.structuralSource
		},
	)
	return packageAuthorityTable{records: records}, remap
}

func (table packageAuthorityTable) authority(
	reference authorityRef,
) (Authority, bool) {
	if reference == 0 ||
		uint64(reference) > uint64(len(table.records)) {
		return Authority{}, false
	}
	return table.records[reference-1], true
}
