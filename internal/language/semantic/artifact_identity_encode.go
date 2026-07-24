package semantic

import (
	"encoding/json"
	"fmt"
	"io"
)

type wireIdentityEncoder struct {
	table packageIdentityTable
}

func encodeWireReference[Reference ~uint64](
	index uint64,
	limit int,
) (Reference, error) {
	if index == 0 {
		return 0, nil
	}
	if index > uint64(limit) {
		return 0, fmt.Errorf(
			"semantic identity reference %d exceeds %d",
			index, limit,
		)
	}
	return newWireReference[Reference](index)
}

func (encoder wireIdentityEncoder) module(
	reference moduleRef,
) (wireModuleReference, error) {
	return encodeWireReference[wireModuleReference](
		uint64(reference),
		len(encoder.table.modules),
	)
}

func (encoder wireIdentityEncoder) owner(
	reference ownerRef,
) (wireOwnerReference, error) {
	return encodeWireReference[wireOwnerReference](
		uint64(reference),
		len(encoder.table.owners),
	)
}

func (encoder wireIdentityEncoder) packageID(
	reference packageRef,
) (wirePackageReference, error) {
	return encodeWireReference[wirePackageReference](
		uint64(reference),
		len(encoder.table.packages),
	)
}

func (encoder wireIdentityEncoder) file(
	reference fileRef,
) (wireFileReference, error) {
	return encodeWireReference[wireFileReference](
		uint64(reference),
		len(encoder.table.files),
	)
}

func (encoder wireIdentityEncoder) span(
	reference spanRef,
) (wireSpanReference, error) {
	return encodeWireReference[wireSpanReference](
		uint64(reference),
		len(encoder.table.spans),
	)
}

func (encoder wireIdentityEncoder) occurrence(
	reference occurrenceRef,
) (wireOccurrenceReference, error) {
	return encodeWireReference[wireOccurrenceReference](
		uint64(reference),
		len(encoder.table.occurrences),
	)
}

func (encoder wireIdentityEncoder) definition(
	reference definitionRef,
) (wireDefinitionReference, error) {
	return encodeWireReference[wireDefinitionReference](
		uint64(reference),
		len(encoder.table.definitions),
	)
}

func (encoder wireIdentityEncoder) typeID(
	reference typeRef,
) (wireTypeReference, error) {
	return encodeWireReference[wireTypeReference](
		uint64(reference),
		len(encoder.table.types),
	)
}

func (encoder wireIdentityEncoder) declaration(
	reference declarationRef,
) (wireDeclarationReference, error) {
	return encodeWireReference[wireDeclarationReference](
		uint64(reference),
		len(encoder.table.declarations),
	)
}

func (encoder wireIdentityEncoder) binding(
	reference bindingRef,
) (wireBindingReference, error) {
	return encodeWireReference[wireBindingReference](
		uint64(reference),
		len(encoder.table.bindings),
	)
}

func (encoder wireIdentityEncoder) operation(
	reference operationRef,
) (wireOperationReference, error) {
	return encodeWireReference[wireOperationReference](
		uint64(reference),
		len(encoder.table.operations),
	)
}

func (encoder wireIdentityEncoder) unsupported(
	reference unsupportedRef,
) (wireUnsupportedReference, error) {
	return encodeWireReference[wireUnsupportedReference](
		uint64(reference),
		len(encoder.table.unsupported),
	)
}

func writeWireField(output io.Writer, name string) error {
	encoded, err := json.Marshal(name)
	if err != nil {
		return err
	}
	if _, err := output.Write(encoded); err != nil {
		return err
	}
	_, err = io.WriteString(output, ":")
	return err
}

func writeWireArray[Record any](
	output io.Writer,
	name string,
	count int,
	project func(int) (Record, error),
	observe ...func(int, int64),
) error {
	if err := writeWireField(output, name); err != nil {
		return err
	}
	if _, err := io.WriteString(output, "["); err != nil {
		return err
	}
	recordOutput := wireRecordWriter{output: output}
	encoder := json.NewEncoder(&recordOutput)
	for index := 0; index < count; index++ {
		if index != 0 {
			if _, err := io.WriteString(output, ","); err != nil {
				return err
			}
		}
		record, err := project(index)
		if err != nil {
			return err
		}
		recordOutput.bytes = 0
		if err := encoder.Encode(record); err != nil {
			return err
		}
		if recordOutput.bytes == 0 ||
			recordOutput.last != '\n' {
			return fmt.Errorf(
				"semantic wire encoder omitted its record delimiter",
			)
		}
		if len(observe) != 0 && observe[0] != nil {
			observe[0](index, recordOutput.bytes-1)
		}
	}
	_, err := io.WriteString(output, "]")
	return err
}

type wireRecordWriter struct {
	output io.Writer
	bytes  int64
	last   byte
}

func (writer *wireRecordWriter) Write(value []byte) (int, error) {
	written, err := writer.output.Write(value)
	writer.bytes += int64(written)
	if written != 0 {
		writer.last = value[written-1]
	}
	return written, err
}

func writeWireSeparator(output io.Writer) error {
	_, err := io.WriteString(output, ",")
	return err
}

func writeIdentityTables(
	output io.Writer,
	table packageIdentityTable,
) error {
	encoder := wireIdentityEncoder{table: table}
	if err := writeWireField(output, "identities"); err != nil {
		return err
	}
	if _, err := io.WriteString(output, "{"); err != nil {
		return err
	}
	if err := writeWireArray(
		output, "modules", len(table.modules),
		func(index int) (wireModuleIdentity, error) {
			value := table.modules[index]
			return wireModuleIdentity{
				Path: value.path, Version: value.version,
			}, nil
		},
	); err != nil {
		return err
	}
	if err := writeWireSeparator(output); err != nil {
		return err
	}
	if err := writeWireArray(
		output, "owners", len(table.owners),
		func(index int) (wireOwnerIdentity, error) {
			value := table.owners[index]
			module, err := encoder.module(
				value.module,
			)
			return wireOwnerIdentity{
				Class: uint8(value.class), Module: module,
			}, err
		},
	); err != nil {
		return err
	}
	if err := writeRemainingIdentityTables(output, encoder); err != nil {
		return err
	}
	_, err := io.WriteString(output, "}")
	return err
}

func writeRemainingIdentityTables(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	writers := []func() error{
		func() error { return writePackageIdentities(output, encoder) },
		func() error { return writeFileIdentities(output, encoder) },
		func() error { return writeSpanIdentities(output, encoder) },
		func() error { return writeOccurrenceIdentities(output, encoder) },
		func() error { return writeDefinitionIdentities(output, encoder) },
		func() error { return writeTypeIdentities(output, encoder) },
		func() error { return writeDeclarationIdentities(output, encoder) },
		func() error { return writeBindingIdentities(output, encoder) },
		func() error { return writeOperationIdentities(output, encoder) },
		func() error { return writeUnsupportedIdentities(output, encoder) },
	}
	for _, write := range writers {
		if err := writeWireSeparator(output); err != nil {
			return err
		}
		if err := write(); err != nil {
			return err
		}
	}
	return nil
}
