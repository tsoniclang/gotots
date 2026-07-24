package semantic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

const semanticShardBufferBytes = 256 * 1024

type byteCountingWriter struct {
	output io.Writer
	bytes  int64
}

func (writer *byteCountingWriter) Write(value []byte) (int, error) {
	written, err := writer.output.Write(value)
	writer.bytes += int64(written)
	return written, err
}

func writeSemanticShard(
	output io.Writer,
	pkg Package,
) (semanticShardMeasurement, error) {
	if output == nil || pkg.ID().IsZero() || !pkg.Provenance().Valid() {
		return semanticShardMeasurement{}, fmt.Errorf(
			"semantic provider shard writer requires package and output",
		)
	}
	measurement := newSemanticShardMeasurement(pkg.ID())
	buffered := bufio.NewWriterSize(output, semanticShardBufferBytes)
	counted := &byteCountingWriter{output: buffered}
	identities := wireIdentityEncoder{table: pkg.identities}
	if err := writeNormalizedShardHeader(
		counted, pkg, identities,
	); err != nil {
		return semanticShardMeasurement{}, err
	}
	definitions := wireDefinitionEncoder{
		identities: identities,
		store:      pkg.definitions,
	}
	if err := writeNormalizedRecords(
		counted,
		"definitions",
		len(pkg.definitions.records),
		definitions.record,
		func(index int, encodedBytes int64) {
			id := pkg.identities.definition(
				pkg.definitions.records[index].id,
			)
			measurement.consider(
				&measurement.definitionTail,
				id.String(),
				encodedBytes,
			)
		},
	); err != nil {
		return semanticShardMeasurement{}, err
	}
	if err := writeWireSeparator(counted); err != nil {
		return semanticShardMeasurement{}, err
	}
	resolutions := wireResolutionEncoder{
		identities: identities,
		store:      pkg.resolutions,
	}
	if err := writeNormalizedRecords(
		counted,
		"resolutions",
		len(pkg.resolutions.records),
		resolutions.record,
		nil,
	); err != nil {
		return semanticShardMeasurement{}, err
	}
	objects := wireObjectEncoder{identities: identities}
	if err := writeWireSeparator(counted); err != nil {
		return semanticShardMeasurement{}, err
	}
	if err := writeNormalizedRecords(
		counted,
		"declarations",
		len(pkg.declarations.records),
		func(index int) (wireDeclarationRecord, error) {
			return objects.declaration(pkg.declarations, index)
		},
		nil,
	); err != nil {
		return semanticShardMeasurement{}, err
	}
	if err := writeWireSeparator(counted); err != nil {
		return semanticShardMeasurement{}, err
	}
	if err := writeNormalizedRecords(
		counted,
		"bindings",
		len(pkg.bindings.records),
		func(index int) (wireBindingRecord, error) {
			return objects.binding(pkg.bindings, index)
		},
		nil,
	); err != nil {
		return semanticShardMeasurement{}, err
	}
	types := wireTypeEncoder{
		identities: identities,
		store:      pkg.types,
	}
	if err := writeWireSeparator(counted); err != nil {
		return semanticShardMeasurement{}, err
	}
	if err := writeNormalizedRecords(
		counted,
		"types",
		len(pkg.types.records),
		types.record,
		func(index int, encodedBytes int64) {
			id := pkg.identities.typeID(pkg.types.records[index].id)
			measurement.consider(
				&measurement.typeTail,
				id.String(),
				encodedBytes,
			)
		},
	); err != nil {
		return semanticShardMeasurement{}, err
	}
	operations := wireOperationEncoder{
		identities: identities,
		store:      pkg.operations,
	}
	if err := writeWireSeparator(counted); err != nil {
		return semanticShardMeasurement{}, err
	}
	if err := writeNormalizedRecords(
		counted,
		"operations",
		len(pkg.operations.records),
		operations.record,
		func(index int, encodedBytes int64) {
			id := pkg.identities.operation(
				pkg.operations.records[index].id,
			)
			measurement.consider(
				&measurement.operationTail,
				id.String(),
				encodedBytes,
			)
		},
	); err != nil {
		return semanticShardMeasurement{}, err
	}
	if err := writeWireSeparator(counted); err != nil {
		return semanticShardMeasurement{}, err
	}
	if err := writeNormalizedRecords(
		counted,
		"unsupported",
		len(pkg.unsupported.records),
		func(index int) (wireUnsupportedRecord, error) {
			return objects.unsupported(pkg.unsupported, index)
		},
		nil,
	); err != nil {
		return semanticShardMeasurement{}, err
	}
	if _, err := io.WriteString(counted, "}"); err != nil {
		return semanticShardMeasurement{}, err
	}
	if err := buffered.Flush(); err != nil {
		return semanticShardMeasurement{}, err
	}
	measurement.encodedBytes = counted.bytes
	return measurement, nil
}

func writeNormalizedShardHeader(
	output io.Writer,
	pkg Package,
	identities wireIdentityEncoder,
) error {
	if _, err := io.WriteString(
		output,
		`{"version":`+strconv.Itoa(ProviderArtifactVersion),
	); err != nil {
		return err
	}
	if _, err := io.WriteString(
		output,
		`,"provenance":`+strconv.Itoa(int(pkg.Provenance())),
	); err != nil {
		return err
	}
	counts := normalizedShardCounts(pkg)
	encodedCounts, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(output, `,"counts":`); err != nil {
		return err
	}
	if _, err := output.Write(encodedCounts); err != nil {
		return err
	}
	if err := writeWireSeparator(output); err != nil {
		return err
	}
	if err := writeIdentityTables(output, pkg.identities); err != nil {
		return err
	}
	reference, err := identities.packageID(
		pkg.identities.packageReference(pkg.ID()),
	)
	if err != nil || reference == 0 {
		return fmt.Errorf(
			"semantic package identity is absent from normalized dictionary",
		)
	}
	if _, err := io.WriteString(output, ","); err != nil {
		return err
	}
	if err := writeWireField(output, "package"); err != nil {
		return err
	}
	encodedPackage, err := json.Marshal(reference)
	if err != nil {
		return err
	}
	if _, err := output.Write(encodedPackage); err != nil {
		return err
	}
	return writeWireSeparator(output)
}

func normalizedShardCounts(pkg Package) semanticShardCounts {
	identities := pkg.identities
	return semanticShardCounts{
		Modules:            uint64(len(identities.modules)),
		Owners:             uint64(len(identities.owners)),
		Packages:           uint64(len(identities.packages)),
		Files:              uint64(len(identities.files)),
		Spans:              uint64(len(identities.spans)),
		Occurrences:        uint64(len(identities.occurrences)),
		Definitions:        uint64(len(identities.definitions)),
		Types:              uint64(len(identities.types)),
		Declarations:       uint64(len(identities.declarations)),
		Bindings:           uint64(len(identities.bindings)),
		Operations:         uint64(len(identities.operations)),
		Unsupported:        uint64(len(identities.unsupported)),
		DefinitionRecords:  uint64(len(pkg.definitions.records)),
		ResolutionRecords:  uint64(len(pkg.resolutions.records)),
		DeclarationRecords: uint64(len(pkg.declarations.records)),
		BindingRecords:     uint64(len(pkg.bindings.records)),
		TypeRecords:        uint64(len(pkg.types.records)),
		OperationRecords:   uint64(len(pkg.operations.records)),
		UnsupportedRecords: uint64(len(pkg.unsupported.records)),
	}
}

func writeNormalizedRecords[Wire any](
	output io.Writer,
	name string,
	count int,
	project func(int) (Wire, error),
	observe func(int, int64),
) error {
	if err := writeWireArray(
		output,
		name,
		count,
		func(index int) (Wire, error) {
			record, err := project(index)
			if err != nil {
				var zero Wire
				return zero, fmt.Errorf(
					"project normalized semantic %s record %d: %w",
					name,
					index,
					err,
				)
			}
			return record, nil
		},
		observe,
	); err != nil {
		return fmt.Errorf(
			"encode normalized semantic %s: %w", name, err,
		)
	}
	return nil
}
