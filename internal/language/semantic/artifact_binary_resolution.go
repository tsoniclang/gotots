package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func writeBinaryResolutions(
	encoder *binaryShardEncoder,
	store packageResolutionStore,
) {
	encoder.count(len(store.records))
	for _, record := range store.records {
		encoder.unsigned(uint64(record.occurrence))
		encoder.unsigned(uint64(record.owner))
		encoder.unsigned(uint64(record.syntax))
		encoder.unsigned(uint64(record.role))
		encoder.unsigned(uint64(record.variant))
		encoder.unsigned(uint64(record.domain))
		encoder.unsigned(uint64(record.kind))
		switch record.kind {
		case ResolutionStructuralOnly:
			payload := store.structural[record.payload-1]
			encoder.unsigned(uint64(payload.disposition))
			encoder.unsigned(uint64(payload.declaration))
			encoder.unsigned(uint64(payload.typeID))
		case ResolutionDefinitionComponent:
			payload := store.definitionComponents[record.payload-1]
			encoder.unsigned(uint64(payload.component))
			encoder.unsigned(uint64(payload.definition))
		case ResolutionDeclaration:
			encoder.unsigned(
				uint64(store.declarations[record.payload-1]),
			)
		case ResolutionBinding:
			encoder.unsigned(
				uint64(store.bindings[record.payload-1]),
			)
		case ResolutionType:
			encoder.unsigned(uint64(store.types[record.payload-1]))
		case ResolutionOperation:
			encoder.unsigned(
				uint64(store.operations[record.payload-1]),
			)
		case ResolutionUnsupported:
			encoder.unsigned(
				uint64(store.unsupported[record.payload-1]),
			)
		default:
			if encoder.err == nil {
				encoder.err = fmt.Errorf(
					"semantic binary resolution kind %d is invalid",
					record.kind,
				)
			}
		}
	}
}

func readBinaryResolutions(
	decoder *binaryShardDecoder,
	expected int,
) (packageResolutionStore, error) {
	count, err := readExpectedRecordCount(
		decoder, "resolutions", expected,
	)
	if err != nil {
		return packageResolutionStore{}, err
	}
	store := packageResolutionStore{
		records: make([]storedResolution, 0, count),
	}
	for index := 0; index < count; index++ {
		if err := readBinaryResolutionRecord(decoder, &store); err != nil {
			return packageResolutionStore{}, fmt.Errorf(
				"decode semantic binary resolution %d: %w",
				index,
				err,
			)
		}
	}
	return store, nil
}

func readBinaryResolutionRecord(
	decoder *binaryShardDecoder,
	store *packageResolutionStore,
) error {
	occurrence, err := readIdentityReference[occurrenceRef](
		decoder, "resolution occurrence",
	)
	if err != nil {
		return err
	}
	owner, err := readIdentityReference[definitionRef](
		decoder, "resolution owner",
	)
	if err != nil {
		return err
	}
	syntax, err := readUnsignedAs[catalog.Kind](
		decoder, "resolution syntax",
	)
	if err != nil {
		return err
	}
	role, err := readUnsignedAs[catalog.Role](
		decoder, "resolution role",
	)
	if err != nil {
		return err
	}
	variant, err := readUnsignedAs[catalog.Variant](
		decoder, "resolution variant",
	)
	if err != nil {
		return err
	}
	domain, err := readUnsignedAs[catalog.ResolutionDomain](
		decoder, "resolution domain",
	)
	if err != nil {
		return err
	}
	kind, err := readUnsignedAs[ResolutionKind](
		decoder, "resolution kind",
	)
	if err != nil {
		return err
	}
	record := storedResolution{
		occurrence: occurrence, owner: owner,
		syntax: syntax, role: role, variant: variant,
		domain: domain, kind: kind,
	}
	switch kind {
	case ResolutionStructuralOnly:
		payload, readErr := readBinaryStructuralResolution(decoder)
		if readErr != nil {
			return readErr
		}
		store.structural = append(store.structural, payload)
		record.payload = uint64(len(store.structural))
	case ResolutionDefinitionComponent:
		payload, readErr := readBinaryDefinitionComponent(decoder)
		if readErr != nil {
			return readErr
		}
		store.definitionComponents = append(
			store.definitionComponents, payload,
		)
		record.payload = uint64(len(store.definitionComponents))
	case ResolutionDeclaration:
		reference, readErr := readIdentityReference[declarationRef](
			decoder, "resolution declaration",
		)
		if readErr != nil {
			return readErr
		}
		store.declarations = append(store.declarations, reference)
		record.payload = uint64(len(store.declarations))
	case ResolutionBinding:
		reference, readErr := readIdentityReference[bindingRef](
			decoder, "resolution binding",
		)
		if readErr != nil {
			return readErr
		}
		store.bindings = append(store.bindings, reference)
		record.payload = uint64(len(store.bindings))
	case ResolutionType:
		reference, readErr := readIdentityReference[typeRef](
			decoder, "resolution type",
		)
		if readErr != nil {
			return readErr
		}
		store.types = append(store.types, reference)
		record.payload = uint64(len(store.types))
	case ResolutionOperation:
		reference, readErr := readIdentityReference[operationRef](
			decoder, "resolution operation",
		)
		if readErr != nil {
			return readErr
		}
		store.operations = append(store.operations, reference)
		record.payload = uint64(len(store.operations))
	case ResolutionUnsupported:
		reference, readErr := readIdentityReference[unsupportedRef](
			decoder, "resolution unsupported",
		)
		if readErr != nil {
			return readErr
		}
		store.unsupported = append(store.unsupported, reference)
		record.payload = uint64(len(store.unsupported))
	default:
		return fmt.Errorf(
			"semantic binary resolution kind %d is invalid", kind,
		)
	}
	store.records = append(store.records, record)
	return nil
}

func readBinaryStructuralResolution(
	decoder *binaryShardDecoder,
) (storedStructuralResolution, error) {
	disposition, err := readUnsignedAs[StructuralDisposition](
		decoder, "structural disposition",
	)
	if err != nil {
		return storedStructuralResolution{}, err
	}
	declaration, err := readIdentityReference[declarationRef](
		decoder, "structural declaration",
	)
	if err != nil {
		return storedStructuralResolution{}, err
	}
	typeID, err := readIdentityReference[typeRef](
		decoder, "structural type",
	)
	return storedStructuralResolution{
		disposition: disposition,
		declaration: declaration,
		typeID:      typeID,
	}, err
}

func readBinaryDefinitionComponent(
	decoder *binaryShardDecoder,
) (storedDefinitionComponent, error) {
	component, err := readUnsignedAs[DefinitionComponentKind](
		decoder, "definition component",
	)
	if err != nil {
		return storedDefinitionComponent{}, err
	}
	definition, err := readIdentityReference[definitionRef](
		decoder, "definition component owner",
	)
	return storedDefinitionComponent{
		component: component, definition: definition,
	}, err
}
