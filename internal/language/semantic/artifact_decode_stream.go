package semantic

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/tsoniclang/gotots/internal/identity"
)

type normalizedShardDecoder struct {
	decoder    *json.Decoder
	identities wireIdentityDecoder
	pkg        identity.PackageID
	provenance PackageProvenance
	counts     semanticShardCounts
}

func decodeSemanticShard(
	input io.Reader,
	authority Authority,
	entry packageShardManifest,
) (Package, error) {
	if input == nil || !authority.Valid() {
		return Package{}, fmt.Errorf(
			"semantic provider shard decoder requires input and authority",
		)
	}
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	var normalized normalizedPackageBuilder
	shard, err := beginNormalizedShard(
		decoder,
		entry,
		&normalized,
	)
	if err != nil {
		return Package{}, err
	}
	definitions := wireDefinitionDecoder{
		identities: shard.identities,
		authority:  authority,
	}
	if err := decodeShardRecords(
		decoder,
		"definitions",
		entry.DefinitionCount,
		func(encoded wireDefinitionRecord) error {
			record, decodeErr := definitions.record(encoded)
			if decodeErr != nil {
				return decodeErr
			}
			normalized.addDefinition(record)
			return nil
		},
	); err != nil {
		return Package{}, err
	}
	resolutions := wireResolutionDecoder{
		identities: shard.identities,
	}
	if err := decodeShardRecords(
		decoder,
		"resolutions",
		entry.ResolutionCount,
		func(encoded wireResolutionRecord) error {
			record, decodeErr := resolutions.record(encoded)
			if decodeErr != nil {
				return decodeErr
			}
			normalized.addResolution(record)
			return nil
		},
	); err != nil {
		return Package{}, err
	}
	objects := wireObjectDecoder{
		identities: shard.identities,
		authority:  authority,
	}
	if err := decodeShardRecords(
		decoder,
		"declarations",
		entry.DeclarationCount,
		func(encoded wireDeclarationRecord) error {
			record, decodeErr := objects.declaration(encoded)
			if decodeErr != nil {
				return decodeErr
			}
			normalized.addDeclaration(record)
			return nil
		},
	); err != nil {
		return Package{}, err
	}
	if err := decodeShardRecords(
		decoder,
		"bindings",
		entry.BindingCount,
		func(encoded wireBindingRecord) error {
			record, decodeErr := objects.binding(encoded)
			if decodeErr != nil {
				return decodeErr
			}
			normalized.addBinding(record)
			return nil
		},
	); err != nil {
		return Package{}, err
	}
	types := wireTypeDecoder{identities: shard.identities}
	if err := decodeShardRecords(
		decoder,
		"types",
		entry.TypeCount,
		func(encoded wireTypeRecord) error {
			record, decodeErr := types.record(encoded)
			if decodeErr != nil {
				return decodeErr
			}
			witness, witnessErr := NewTypeWitness(
				shard.pkg,
				record.ID(),
				authority,
			)
			if witnessErr != nil {
				return witnessErr
			}
			normalized.addType(record)
			normalized.addTypeWitness(witness)
			return nil
		},
	); err != nil {
		return Package{}, err
	}
	operations := wireOperationDecoder{
		identities: shard.identities,
	}
	if err := decodeShardRecords(
		decoder,
		"operations",
		entry.OperationCount,
		func(encoded wireOperationRecord) error {
			record, decodeErr := operations.record(encoded)
			if decodeErr != nil {
				return decodeErr
			}
			normalized.addOperation(record)
			return nil
		},
	); err != nil {
		return Package{}, err
	}
	if err := decodeShardRecords(
		decoder,
		"unsupported",
		entry.UnsupportedCount,
		func(encoded wireUnsupportedRecord) error {
			record, decodeErr := objects.unsupported(encoded)
			if decodeErr != nil {
				return decodeErr
			}
			normalized.addUnsupported(record)
			return nil
		},
	); err != nil {
		return Package{}, err
	}
	if err := finishNormalizedShard(shard); err != nil {
		return Package{}, err
	}
	return newPackageFromBuilder(
		shard.pkg,
		shard.provenance,
		&normalized,
	)
}

func beginNormalizedShard(
	decoder *json.Decoder,
	entry packageShardManifest,
	normalized *normalizedPackageBuilder,
) (normalizedShardDecoder, error) {
	if decoder == nil || normalized == nil {
		return normalizedShardDecoder{}, fmt.Errorf(
			"semantic shard decoder is absent",
		)
	}
	if err := requireShardDelimiter(decoder, '{'); err != nil {
		return normalizedShardDecoder{}, err
	}
	var version int
	if err := decodeShardField(
		decoder,
		"version",
		&version,
	); err != nil {
		return normalizedShardDecoder{}, err
	}
	if version != ProviderArtifactVersion {
		return normalizedShardDecoder{}, fmt.Errorf(
			"semantic shard version %d is unsupported", version,
		)
	}
	var encodedProvenance uint8
	if err := decodeShardField(
		decoder,
		"provenance",
		&encodedProvenance,
	); err != nil {
		return normalizedShardDecoder{}, err
	}
	provenance := PackageProvenance(encodedProvenance)
	if encodedProvenance != entry.Provenance ||
		!provenance.Valid() {
		return normalizedShardDecoder{}, fmt.Errorf(
			"semantic shard provenance disagrees with manifest",
		)
	}
	var counts semanticShardCounts
	if err := decodeShardField(
		decoder,
		"counts",
		&counts,
	); err != nil {
		return normalizedShardDecoder{}, err
	}
	if err := counts.validate(entry); err != nil {
		return normalizedShardDecoder{}, err
	}
	if err := normalized.reserve(counts); err != nil {
		return normalizedShardDecoder{}, err
	}
	identityDecoder, err := decodeIdentityTables(
		decoder,
		counts,
		&normalized.identities,
	)
	if err != nil {
		return normalizedShardDecoder{}, err
	}
	var packageReference wirePackageReference
	if err := decodeShardField(
		decoder,
		"package",
		&packageReference,
	); err != nil {
		return normalizedShardDecoder{}, err
	}
	pkg, err := identityDecoder.packageID(packageReference)
	if err != nil {
		return normalizedShardDecoder{}, err
	}
	manifestPackage, err := identity.ParsePackageID(entry.Package)
	if err != nil {
		return normalizedShardDecoder{}, err
	}
	if pkg.IsZero() || pkg != manifestPackage {
		return normalizedShardDecoder{}, fmt.Errorf(
			"semantic shard package %s disagrees with manifest %s",
			pkg,
			manifestPackage,
		)
	}
	return normalizedShardDecoder{
		decoder:    decoder,
		identities: identityDecoder,
		pkg:        pkg,
		provenance: provenance,
		counts:     counts,
	}, nil
}

func finishNormalizedShard(shard normalizedShardDecoder) error {
	if err := requireShardDelimiter(
		shard.decoder,
		'}',
	); err != nil {
		return err
	}
	if err := requireJSONEnd(shard.decoder); err != nil {
		return err
	}
	return shard.identities.validateUsage()
}

func decodeShardField(
	decoder *json.Decoder,
	name string,
	target any,
) error {
	if err := requireShardField(decoder, name); err != nil {
		return err
	}
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf(
			"decode semantic provider shard field %s: %w",
			name,
			err,
		)
	}
	return nil
}

func decodeShardRecords[Wire any](
	decoder *json.Decoder,
	name string,
	expected int,
	admit func(Wire) error,
) error {
	if err := requireShardField(decoder, name); err != nil {
		return err
	}
	if err := requireShardDelimiter(decoder, '['); err != nil {
		return err
	}
	count := 0
	for decoder.More() {
		if count >= expected {
			return fmt.Errorf(
				"semantic provider shard %s exceeds manifest count %d",
				name,
				expected,
			)
		}
		var encoded Wire
		if err := decoder.Decode(&encoded); err != nil {
			return fmt.Errorf(
				"decode semantic provider %s record %d: %w",
				name,
				count,
				err,
			)
		}
		if err := admit(encoded); err != nil {
			return fmt.Errorf(
				"admit semantic provider %s record %d: %w",
				name,
				count,
				err,
			)
		}
		count++
	}
	if err := requireShardDelimiter(decoder, ']'); err != nil {
		return err
	}
	if count != expected {
		return fmt.Errorf(
			"semantic provider shard %s count %d disagrees with manifest %d",
			name,
			count,
			expected,
		)
	}
	return nil
}

func requireShardField(
	decoder *json.Decoder,
	expected string,
) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	actual, field := token.(string)
	if !field || actual != expected {
		return fmt.Errorf(
			"semantic provider shard field %q, want %q",
			actual,
			expected,
		)
	}
	return nil
}

func requireShardDelimiter(
	decoder *json.Decoder,
	expected json.Delim,
) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	actual, delimiter := token.(json.Delim)
	if !delimiter {
		return fmt.Errorf(
			"semantic provider shard token %q, want delimiter %q",
			token,
			expected,
		)
	}
	if actual != expected {
		return fmt.Errorf(
			"semantic provider shard delimiter %q, want %q",
			actual,
			expected,
		)
	}
	return nil
}

func requireJSONEnd(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf(
				"semantic provider artifact has trailing JSON",
			)
		}
		return err
	}
	return nil
}
