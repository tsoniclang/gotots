package semantic

import (
	"encoding/json"
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

type wireIdentityTable[Reference ~uint64] struct {
	references []Reference
	used       []bool
}

type wireIdentityDecoder struct {
	builder      *packageIdentityBuilder
	projection   *packageIdentityProjection
	modules      wireIdentityTable[moduleRef]
	owners       wireIdentityTable[ownerRef]
	packages     wireIdentityTable[packageRef]
	files        wireIdentityTable[fileRef]
	spans        wireIdentityTable[spanRef]
	occurrences  wireIdentityTable[occurrenceRef]
	definitions  wireIdentityTable[definitionRef]
	types        wireIdentityTable[typeRef]
	declarations wireIdentityTable[declarationRef]
	bindings     wireIdentityTable[bindingRef]
	operations   wireIdentityTable[operationRef]
	unsupported  wireIdentityTable[unsupportedRef]
}

func decodeIdentityTables(
	decoder *json.Decoder,
	counts semanticShardCounts,
	builder *packageIdentityBuilder,
) (wireIdentityDecoder, error) {
	if decoder == nil || builder == nil {
		return wireIdentityDecoder{}, fmt.Errorf(
			"semantic identity decoder requires decoder and builder",
		)
	}
	out := wireIdentityDecoder{
		builder:    builder,
		projection: newPackageIdentityProjection(packageIdentityTable{}),
	}
	if err := requireShardField(decoder, "identities"); err != nil {
		return wireIdentityDecoder{}, err
	}
	if err := requireShardDelimiter(decoder, '{'); err != nil {
		return wireIdentityDecoder{}, err
	}
	var err error
	out.modules.references, err = decodeIdentityRecords(
		decoder,
		"modules",
		counts.Modules,
		func(record wireModuleIdentity) (identity.ModuleID, error) {
			return identity.NewModuleID(record.Path, record.Version)
		},
		func(left, right identity.ModuleID) int {
			return left.Compare(right)
		},
		builder.module,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.modules.used = make([]bool, len(out.modules.references))
	out.owners.references, err = decodeIdentityRecords(
		decoder,
		"owners",
		counts.Owners,
		out.decodeOwner,
		func(left, right identity.Owner) int {
			return left.Compare(right)
		},
		builder.owner,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.owners.used = make([]bool, len(out.owners.references))
	out.packages.references, err = decodeIdentityRecords(
		decoder,
		"packages",
		counts.Packages,
		out.decodePackage,
		func(left, right identity.PackageID) int {
			return left.Compare(right)
		},
		builder.packageID,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.packages.used = make([]bool, len(out.packages.references))
	out.files.references, err = decodeIdentityRecords(
		decoder,
		"files",
		counts.Files,
		out.decodeFile,
		func(left, right identity.FileID) int {
			return left.Compare(right)
		},
		builder.file,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.files.used = make([]bool, len(out.files.references))
	out.spans.references, err = decodeIdentityRecords(
		decoder,
		"spans",
		counts.Spans,
		out.decodeSpan,
		func(left, right identity.SpanID) int {
			return left.Compare(right)
		},
		builder.span,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.spans.used = make([]bool, len(out.spans.references))
	out.occurrences.references, err = decodeIdentityRecords(
		decoder,
		"occurrences",
		counts.Occurrences,
		out.decodeOccurrence,
		func(left, right identity.OccurrenceID) int {
			return left.Compare(right)
		},
		builder.occurrence,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.occurrences.used = make([]bool, len(out.occurrences.references))
	out.definitions.references, err = decodeIdentityRecords(
		decoder,
		"definitions",
		counts.Definitions,
		out.decodeDefinition,
		func(left, right identity.DefinitionID) int {
			return left.Compare(right)
		},
		builder.definition,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.definitions.used = make([]bool, len(out.definitions.references))
	out.types.references, err = decodeIdentityRecords(
		decoder,
		"types",
		counts.Types,
		func(record wireTypeIdentity) (identity.SemanticTypeID, error) {
			return identity.NewSemanticTypeID(record.Digest)
		},
		func(left, right identity.SemanticTypeID) int {
			return left.Compare(right)
		},
		builder.typeID,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.types.used = make([]bool, len(out.types.references))
	out.declarations.references, err = decodeIdentityRecords(
		decoder,
		"declarations",
		counts.Declarations,
		out.decodeDeclaration,
		func(left, right identity.SemanticDeclarationID) int {
			return left.Compare(right)
		},
		builder.declaration,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.declarations.used = make([]bool, len(out.declarations.references))
	out.bindings.references, err = decodeIdentityRecords(
		decoder,
		"bindings",
		counts.Bindings,
		out.decodeBinding,
		func(left, right identity.SemanticBindingID) int {
			return left.Compare(right)
		},
		builder.binding,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.bindings.used = make([]bool, len(out.bindings.references))
	out.operations.references, err = decodeIdentityRecords(
		decoder,
		"operations",
		counts.Operations,
		out.decodeOperation,
		func(left, right identity.OperationID) int {
			return left.Compare(right)
		},
		builder.operation,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.operations.used = make([]bool, len(out.operations.references))
	out.unsupported.references, err = decodeIdentityRecords(
		decoder,
		"unsupported",
		counts.Unsupported,
		out.decodeUnsupported,
		func(left, right identity.UnsupportedID) int {
			return left.Compare(right)
		},
		builder.unsupportedID,
	)
	if err != nil {
		return wireIdentityDecoder{}, err
	}
	out.unsupported.used = make([]bool, len(out.unsupported.references))
	if err := requireShardDelimiter(decoder, '}'); err != nil {
		return wireIdentityDecoder{}, err
	}
	return out, nil
}

func decodeIdentityRecords[
	Wire any,
	Identity any,
	Reference ~uint64,
](
	decoder *json.Decoder,
	name string,
	expected uint64,
	decode func(Wire) (Identity, error),
	compare func(Identity, Identity) int,
	admit func(Identity) Reference,
) ([]Reference, error) {
	count, err := boundedShardCount(name, expected)
	if err != nil {
		return nil, err
	}
	out := make([]Reference, 1, count+1)
	var previous Identity
	hasPrevious := false
	err = decodeShardRecords(
		decoder,
		name,
		count,
		func(record Wire) error {
			value, decodeErr := decode(record)
			if decodeErr != nil {
				return decodeErr
			}
			if hasPrevious && compare(value, previous) <= 0 {
				return fmt.Errorf(
					"semantic identity table %s is not canonical",
					name,
				)
			}
			reference := admit(value)
			if reference == 0 {
				return fmt.Errorf(
					"semantic identity table %s produced zero reference",
					name,
				)
			}
			out = append(out, reference)
			previous = value
			hasPrevious = true
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func boundedShardCount(name string, count uint64) (int, error) {
	maxInt := uint64(^uint(0) >> 1)
	if count > maxInt {
		return 0, fmt.Errorf(
			"semantic shard %s count %d exceeds platform capacity",
			name,
			count,
		)
	}
	return int(count), nil
}
