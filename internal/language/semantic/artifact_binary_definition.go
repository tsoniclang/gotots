package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func writeBinaryDefinitions(
	encoder *binaryShardEncoder,
	pkg Package,
	measurement *semanticShardMeasurement,
) {
	store := pkg.definitions
	encoder.count(len(store.records))
	identities := newPackageIdentityProjection(pkg.identities)
	for _, record := range store.records {
		start := encoder.written
		encoder.unsigned(uint64(record.id))
		encoder.unsigned(uint64(record.pkg))
		encoder.unsigned(uint64(record.form))
		encoder.text(record.name)
		writeReferenceRange(
			encoder,
			store.bindingRelations,
			record.bindings.start,
			record.bindings.count,
		)
		writeBinaryDefinitionPayload(encoder, store, record)
		measurement.consider(
			&measurement.definitionTail,
			identities.definition(record.id).String(),
			encoder.written-start,
		)
	}
}

func writeBinaryDefinitionPayload(
	encoder *binaryShardEncoder,
	store packageDefinitionStore,
	record storedDefinition,
) {
	switch record.form {
	case DefinitionFormCallable:
		payload := store.callable[record.payload-1]
		writeReferenceRange(
			encoder,
			store.declarationRelations,
			payload.declarations.start,
			payload.declarations.count,
		)
		encoder.unsigned(uint64(payload.signature))
		encoder.unsigned(uint64(payload.receiver))
	case DefinitionFormInitializer:
		payload := store.initializers[record.payload-1]
		writeReferenceRange(
			encoder,
			store.declarationRelations,
			payload.declarations.start,
			payload.declarations.count,
		)
		writeReferenceRange(
			encoder,
			store.initializerEntries,
			payload.entries.start,
			payload.entries.count,
		)
	case DefinitionFormBodyless:
		payload := store.bodyless[record.payload-1]
		encoder.unsigned(uint64(payload.declaration))
		encoder.unsigned(uint64(payload.signature))
		encoder.unsigned(uint64(payload.receiver))
	case DefinitionFormImplicit:
		payload := store.implicit[record.payload-1]
		encoder.unsigned(uint64(payload.operation))
	case DefinitionFormSynthetic:
		payload := store.synthetic[record.payload-1]
		encoder.unsigned(uint64(payload.declaration))
		encoder.unsigned(uint64(payload.signature))
	default:
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary definition form %d is invalid",
				record.form,
			)
		}
	}
}

func readBinaryDefinitions(
	decoder *binaryShardDecoder,
	expected int,
	authority authorityRef,
) (packageDefinitionStore, error) {
	count, err := readExpectedRecordCount(
		decoder, "definitions", expected,
	)
	if err != nil {
		return packageDefinitionStore{}, err
	}
	store := packageDefinitionStore{
		records: make([]storedDefinition, 0, count),
	}
	for index := 0; index < count; index++ {
		if err := readBinaryDefinitionRecord(
			decoder, &store, authority,
		); err != nil {
			return packageDefinitionStore{}, fmt.Errorf(
				"decode semantic binary definition %d: %w",
				index,
				err,
			)
		}
	}
	return store, nil
}

func readBinaryDefinitionRecord(
	decoder *binaryShardDecoder,
	store *packageDefinitionStore,
	authority authorityRef,
) error {
	id, err := readIdentityReference[definitionRef](
		decoder, "definition id",
	)
	if err != nil {
		return err
	}
	pkg, err := readIdentityReference[packageRef](
		decoder, "definition package",
	)
	if err != nil {
		return err
	}
	form, err := readUnsignedAs[DefinitionForm](
		decoder, "definition form",
	)
	if err != nil {
		return err
	}
	name, err := decoder.text("definition name")
	if err != nil {
		return err
	}
	bindingStart, bindingCount, err := readReferenceRange(
		decoder,
		"definition bindings",
		&store.bindingRelations,
	)
	if err != nil {
		return err
	}
	record := storedDefinition{
		id: id, pkg: pkg, form: form, authority: authority,
		name: name,
		bindings: bindingRefRange{
			start: bindingStart, count: bindingCount,
		},
	}
	switch form {
	case DefinitionFormCallable:
		payload, readErr := readBinaryCallableDefinition(
			decoder, store,
		)
		if readErr != nil {
			return readErr
		}
		store.callable = append(store.callable, payload)
		record.payload = uint64(len(store.callable))
	case DefinitionFormInitializer:
		payload, readErr := readBinaryInitializerDefinition(
			decoder, store,
		)
		if readErr != nil {
			return readErr
		}
		store.initializers = append(store.initializers, payload)
		record.payload = uint64(len(store.initializers))
	case DefinitionFormBodyless:
		payload, readErr := readBinaryBodylessDefinition(decoder)
		if readErr != nil {
			return readErr
		}
		store.bodyless = append(store.bodyless, payload)
		record.payload = uint64(len(store.bodyless))
	case DefinitionFormImplicit:
		operation, readErr := readUnsignedAs[identity.ImplicitDefinitionOp](decoder, "implicit definition operation")
		if readErr != nil {
			return readErr
		}
		store.implicit = append(
			store.implicit,
			storedImplicitDefinition{operation: operation},
		)
		record.payload = uint64(len(store.implicit))
	case DefinitionFormSynthetic:
		payload, readErr := readBinarySyntheticDefinition(decoder)
		if readErr != nil {
			return readErr
		}
		store.synthetic = append(store.synthetic, payload)
		record.payload = uint64(len(store.synthetic))
	default:
		return fmt.Errorf(
			"semantic binary definition form %d is invalid", form,
		)
	}
	store.records = append(store.records, record)
	return nil
}

func readBinaryCallableDefinition(
	decoder *binaryShardDecoder,
	store *packageDefinitionStore,
) (storedCallableDefinition, error) {
	start, count, err := readReferenceRange(
		decoder,
		"callable declarations",
		&store.declarationRelations,
	)
	if err != nil {
		return storedCallableDefinition{}, err
	}
	signature, err := readIdentityReference[typeRef](
		decoder, "callable signature",
	)
	if err != nil {
		return storedCallableDefinition{}, err
	}
	receiver, err := readIdentityReference[bindingRef](
		decoder, "callable receiver",
	)
	return storedCallableDefinition{
		declarations: declarationRefRange{start: start, count: count},
		signature:    signature, receiver: receiver,
	}, err
}

func readBinaryInitializerDefinition(
	decoder *binaryShardDecoder,
	store *packageDefinitionStore,
) (storedInitializerDefinition, error) {
	declarationStart, declarationCount, err := readReferenceRange(
		decoder,
		"initializer declarations",
		&store.declarationRelations,
	)
	if err != nil {
		return storedInitializerDefinition{}, err
	}
	entryStart, entryCount, err := readReferenceRange(
		decoder,
		"initializer entries",
		&store.initializerEntries,
	)
	return storedInitializerDefinition{
		declarations: declarationRefRange{
			start: declarationStart, count: declarationCount,
		},
		entries: occurrenceRefRange{
			start: entryStart, count: entryCount,
		},
	}, err
}

func readBinaryBodylessDefinition(
	decoder *binaryShardDecoder,
) (storedBodylessDefinition, error) {
	declaration, err := readIdentityReference[declarationRef](
		decoder, "bodyless declaration",
	)
	if err != nil {
		return storedBodylessDefinition{}, err
	}
	signature, err := readIdentityReference[typeRef](
		decoder, "bodyless signature",
	)
	if err != nil {
		return storedBodylessDefinition{}, err
	}
	receiver, err := readIdentityReference[bindingRef](
		decoder, "bodyless receiver",
	)
	return storedBodylessDefinition{
		declaration: declaration, signature: signature,
		receiver: receiver,
	}, err
}

func readBinarySyntheticDefinition(
	decoder *binaryShardDecoder,
) (storedSyntheticDefinition, error) {
	declaration, err := readIdentityReference[declarationRef](
		decoder, "synthetic declaration",
	)
	if err != nil {
		return storedSyntheticDefinition{}, err
	}
	signature, err := readIdentityReference[typeRef](
		decoder, "synthetic signature",
	)
	return storedSyntheticDefinition{
		declaration: declaration, signature: signature,
	}, err
}
