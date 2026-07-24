package identity

import "cmp"

// Compare returns the canonical structural ordering of two module identities.
func (left ModuleID) Compare(right ModuleID) int {
	if order := cmp.Compare(left.path, right.path); order != 0 {
		return order
	}
	return cmp.Compare(left.version, right.version)
}

// Compare returns the canonical structural ordering of two owners.
func (left Owner) Compare(right Owner) int {
	if order := cmp.Compare(left.class, right.class); order != 0 {
		return order
	}
	return left.module.Compare(right.module)
}

// Compare returns the canonical structural ordering of two package identities.
func (left PackageID) Compare(right PackageID) int {
	if order := left.owner.Compare(right.owner); order != 0 {
		return order
	}
	return cmp.Compare(left.importPath, right.importPath)
}

// Compare returns the canonical structural ordering of two file identities.
func (left FileID) Compare(right FileID) int {
	if order := left.owner.Compare(right.owner); order != 0 {
		return order
	}
	return cmp.Compare(left.rel, right.rel)
}

// Compare returns the canonical structural ordering of two span identities.
func (left SpanID) Compare(right SpanID) int {
	if order := left.file.Compare(right.file); order != 0 {
		return order
	}
	if order := cmp.Compare(left.start, right.start); order != 0 {
		return order
	}
	return cmp.Compare(left.end, right.end)
}

// Compare returns the canonical structural ordering of two occurrence
// identities.
func (left OccurrenceID) Compare(right OccurrenceID) int {
	if order := left.span.Compare(right.span); order != 0 {
		return order
	}
	return cmp.Compare(left.kind, right.kind)
}

// Compare returns the canonical structural ordering of two definition
// identities.
func (left DefinitionID) Compare(right DefinitionID) int {
	if order := cmp.Compare(left.sourceKind, right.sourceKind); order != 0 {
		return order
	}
	if order := left.root.Compare(right.root); order != 0 {
		return order
	}
	if order := left.pkg.Compare(right.pkg); order != 0 {
		return order
	}
	if order := cmp.Compare(left.implicit, right.implicit); order != 0 {
		return order
	}
	if order := cmp.Compare(left.synthetic, right.synthetic); order != 0 {
		return order
	}
	return cmp.Compare(left.name, right.name)
}

// Compare returns the canonical structural ordering of two header-region
// identities.
func (left HeaderRegionID) Compare(right HeaderRegionID) int {
	return left.definition.Compare(right.definition)
}

// Compare returns the canonical structural ordering of two execution-boundary
// identities.
func (left ExecutionBoundaryID) Compare(
	right ExecutionBoundaryID,
) int {
	return left.definition.Compare(right.definition)
}

// Compare returns the canonical structural ordering of two executable-region
// identities.
func (left ExecutableRegionID) Compare(right ExecutableRegionID) int {
	return left.definition.Compare(right.definition)
}

// Compare returns the canonical structural ordering of two semantic-type
// identities.
func (left SemanticTypeID) Compare(right SemanticTypeID) int {
	return cmp.Compare(left.digest, right.digest)
}

// Compare returns the canonical structural ordering of two semantic
// declaration identities.
func (left SemanticDeclarationID) Compare(
	right SemanticDeclarationID,
) int {
	if order := cmp.Compare(left.form, right.form); order != 0 {
		return order
	}
	if order := left.pkg.Compare(right.pkg); order != 0 {
		return order
	}
	if order := left.ownerType.Compare(right.ownerType); order != 0 {
		return order
	}
	if order := left.memberPkg.Compare(right.memberPkg); order != 0 {
		return order
	}
	if order := cmp.Compare(left.class, right.class); order != 0 {
		return order
	}
	if order := cmp.Compare(left.name, right.name); order != 0 {
		return order
	}
	if order := cmp.Compare(left.ordinal, right.ordinal); order != 0 {
		return order
	}
	if order := cmp.Compare(left.predeclared, right.predeclared); order != 0 {
		return order
	}
	if order := left.owner.Compare(right.owner); order != 0 {
		return order
	}
	return left.occurrence.Compare(right.occurrence)
}

// Compare returns the canonical structural ordering of two semantic binding
// identities.
func (left SemanticBindingID) Compare(right SemanticBindingID) int {
	if order := left.owner.Compare(right.owner); order != 0 {
		return order
	}
	if order := left.declaration.Compare(right.declaration); order != 0 {
		return order
	}
	if order := cmp.Compare(left.role, right.role); order != 0 {
		return order
	}
	return cmp.Compare(left.ordinal, right.ordinal)
}

// Compare returns the canonical structural ordering of two operation
// identities.
func (left OperationID) Compare(right OperationID) int {
	if order := left.definition.Compare(right.definition); order != 0 {
		return order
	}
	if order := left.occurrence.Compare(right.occurrence); order != 0 {
		return order
	}
	if order := cmp.Compare(left.implicit, right.implicit); order != 0 {
		return order
	}
	return cmp.Compare(left.ordinal, right.ordinal)
}

// Compare returns the canonical structural ordering of two unsupported-record
// identities.
func (left UnsupportedID) Compare(right UnsupportedID) int {
	if order := left.definition.Compare(right.definition); order != 0 {
		return order
	}
	return left.occurrence.Compare(right.occurrence)
}
