package semantic

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

type byteCountingWriter struct {
	output io.Writer
	bytes  int64
}

func (writer *byteCountingWriter) Write(value []byte) (int, error) {
	written, err := writer.output.Write(value)
	writer.bytes += int64(written)
	return written, err
}

func writeProviderShard(
	output io.Writer,
	pkg Package,
) (int64, error) {
	if output == nil || pkg.ID().IsZero() || !pkg.Provenance().Valid() {
		return 0, fmt.Errorf(
			"semantic provider shard writer requires package and output",
		)
	}
	counted := &byteCountingWriter{output: output}
	if err := writeShardHeader(counted, pkg); err != nil {
		return 0, err
	}
	if err := writeShardRecords(
		counted, "definitions", pkg.definitions, encodeDefinition,
	); err != nil {
		return 0, err
	}
	if err := writeShardRecords(
		counted, "resolutions", pkg.resolutions, encodeResolution,
	); err != nil {
		return 0, err
	}
	if err := writeShardRecords(
		counted, "declarations", pkg.declarations, encodeDeclaration,
	); err != nil {
		return 0, err
	}
	if err := writeShardRecords(
		counted, "bindings", pkg.bindings, encodeBinding,
	); err != nil {
		return 0, err
	}
	if err := writeShardRecords(
		counted, "types", pkg.types, encodeType,
	); err != nil {
		return 0, err
	}
	if err := writeShardRecords(
		counted, "operations", pkg.operations, encodeOperation,
	); err != nil {
		return 0, err
	}
	if err := writeShardRecords(
		counted, "unsupported", pkg.unsupported, encodeUnsupported,
	); err != nil {
		return 0, err
	}
	if _, err := io.WriteString(counted, "}"); err != nil {
		return 0, err
	}
	return counted.bytes, nil
}

func writeShardHeader(
	output io.Writer,
	pkg Package,
) error {
	packageName, err := json.Marshal(pkg.ID().String())
	if err != nil {
		return err
	}
	_, err = io.WriteString(
		output,
		`{"version":`+
			strconv.Itoa(ProviderArtifactVersion)+
			`,"package":`+
			string(packageName)+
			`,"provenance":`+
			strconv.Itoa(int(pkg.Provenance())),
	)
	return err
}

func writeShardRecords[Record any, Wire any](
	output io.Writer,
	name string,
	records []Record,
	encode func(Record) Wire,
) error {
	if _, err := io.WriteString(
		output, `,"`+name+`":[`,
	); err != nil {
		return err
	}
	for index, record := range records {
		if index != 0 {
			if _, err := io.WriteString(output, ","); err != nil {
				return err
			}
		}
		encoded, err := json.Marshal(encode(record))
		if err != nil {
			return fmt.Errorf(
				"encode semantic provider %s record %d: %w",
				name, index, err,
			)
		}
		if _, err := output.Write(encoded); err != nil {
			return err
		}
	}
	_, err := io.WriteString(output, "]")
	return err
}
