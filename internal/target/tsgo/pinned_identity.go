package tsgo

// PinnedToolVersion is the module version of the pinned TS-Go toolchain the
// generated protocol bindings were derived from.
func PinnedToolVersion() string {
	return pinnedToolVersion
}

// PinnedProtocolVersion is the external AST protocol version of the pinned
// TS-Go schema the generated encoder targets.
func PinnedProtocolVersion() int {
	return protocolVersion
}

// PinnedSchemaRevision is the exact upstream TS-Go revision of the pinned
// schema contract.
func PinnedSchemaRevision() string {
	return pinnedSchemaRevision
}

// PinnedSchemaContractDigest content-addresses the complete pinned schema
// contract manifest the generated bindings were derived from.
func PinnedSchemaContractDigest() string {
	return pinnedSchemaContractDigest
}
