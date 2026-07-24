package semantic

import "github.com/tsoniclang/gotots/internal/identity"

// PackageReader owns one constant-size identity projection cursor for repeated
// canonical lookups in one immutable package. It retains no public record
// collection and is intended for one sequential consumer.
type PackageReader struct {
	pkg        *Package
	identities *packageIdentityProjection
}

// Reader opens one sequential canonical-lookup cursor over the package.
func (pkg *Package) Reader() *PackageReader {
	if pkg == nil {
		return &PackageReader{}
	}
	return &PackageReader{
		pkg:        pkg,
		identities: newPackageIdentityProjection(pkg.identities),
	}
}

func (reader *PackageReader) Resolution(
	id identity.OccurrenceID,
) (OccurrenceResolution, bool) {
	if reader == nil || reader.pkg == nil || reader.identities == nil {
		return OccurrenceResolution{}, false
	}
	index, present := reader.pkg.resolutionIndex(id)
	if !present {
		return OccurrenceResolution{}, false
	}
	record, err := reader.pkg.resolutions.record(
		reader.identities,
		index,
	)
	return record, err == nil
}

func (reader *PackageReader) Operation(
	id identity.OperationID,
) (Operation, bool) {
	if reader == nil || reader.pkg == nil || reader.identities == nil {
		return Operation{}, false
	}
	index, present := reader.pkg.operationIndex(id)
	if !present {
		return Operation{}, false
	}
	record, err := reader.pkg.operationView.operation(
		reader.identities,
		index,
	)
	return record, err == nil
}

func (reader *PackageReader) Declaration(
	id identity.SemanticDeclarationID,
) (Declaration, bool) {
	if reader == nil || reader.pkg == nil || reader.identities == nil {
		return Declaration{}, false
	}
	index, present := reader.pkg.declarationIndex(id)
	if !present {
		return Declaration{}, false
	}
	record, err := reader.pkg.declarations.record(
		reader.identities,
		reader.pkg.authorities,
		index,
	)
	return record, err == nil
}

func (reader *PackageReader) Type(
	id identity.SemanticTypeID,
) (Type, bool) {
	if reader == nil || reader.pkg == nil || reader.identities == nil {
		return Type{}, false
	}
	index, present := reader.pkg.typeIndex(id)
	if !present {
		return Type{}, false
	}
	record, err := reader.pkg.types.record(reader.identities, index)
	return record, err == nil
}
