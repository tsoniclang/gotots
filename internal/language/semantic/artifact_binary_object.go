package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func writeBinaryDeclarations(
	encoder *binaryShardEncoder,
	store packageDeclarationStore,
) {
	encoder.count(len(store.records))
	for _, record := range store.records {
		encoder.unsigned(uint64(record.id))
		encoder.unsigned(uint64(record.pkg))
		encoder.unsigned(uint64(record.class))
		encoder.text(record.name)
		encoder.unsigned(uint64(record.typeID))
		encoder.boolean(record.exported)
		encoder.unsigned(uint64(record.constantKind))
		encoder.text(record.constant)
	}
}

func readBinaryDeclarations(
	decoder *binaryShardDecoder,
	expected int,
	authority authorityRef,
) (packageDeclarationStore, error) {
	count, err := readExpectedRecordCount(
		decoder, "declarations", expected,
	)
	if err != nil {
		return packageDeclarationStore{}, err
	}
	store := packageDeclarationStore{
		records: make([]storedDeclaration, 0, count),
	}
	for index := 0; index < count; index++ {
		record, readErr := readBinaryDeclarationRecord(
			decoder, authority,
		)
		if readErr != nil {
			return packageDeclarationStore{}, fmt.Errorf(
				"decode semantic binary declaration %d: %w",
				index,
				readErr,
			)
		}
		store.records = append(store.records, record)
	}
	return store, nil
}

func readBinaryDeclarationRecord(
	decoder *binaryShardDecoder,
	authority authorityRef,
) (storedDeclaration, error) {
	id, err := readIdentityReference[declarationRef](
		decoder, "declaration id",
	)
	if err != nil {
		return storedDeclaration{}, err
	}
	pkg, err := readIdentityReference[packageRef](
		decoder, "declaration package",
	)
	if err != nil {
		return storedDeclaration{}, err
	}
	class, err := readUnsignedAs[identity.SemanticObjectClass](
		decoder, "declaration class",
	)
	if err != nil {
		return storedDeclaration{}, err
	}
	name, err := decoder.text("declaration name")
	if err != nil {
		return storedDeclaration{}, err
	}
	typeID, err := readIdentityReference[typeRef](
		decoder, "declaration type",
	)
	if err != nil {
		return storedDeclaration{}, err
	}
	exported, err := decoder.boolean("declaration exported")
	if err != nil {
		return storedDeclaration{}, err
	}
	constantKind, err := readUnsignedAs[ConstantKind](
		decoder, "declaration constant kind",
	)
	if err != nil {
		return storedDeclaration{}, err
	}
	constant, err := decoder.text("declaration constant")
	return storedDeclaration{
		id: id, pkg: pkg, class: class, name: name,
		typeID: typeID, exported: exported,
		constantKind: constantKind, constant: constant,
		authority: authority,
	}, err
}

func writeBinaryBindings(
	encoder *binaryShardEncoder,
	store packageBindingStore,
) {
	encoder.count(len(store.records))
	for _, record := range store.records {
		encoder.unsigned(uint64(record.id))
		encoder.unsigned(uint64(record.pkg))
		encoder.unsigned(uint64(record.definition))
		encoder.unsigned(uint64(record.role))
		encoder.text(record.name)
		encoder.unsigned(uint64(record.typeID))
		encoder.unsigned(uint64(record.source))
		writeReferenceRange(
			encoder,
			store.captures,
			record.captures.start,
			record.captures.count,
		)
	}
}

func readBinaryBindings(
	decoder *binaryShardDecoder,
	expected int,
	authority authorityRef,
) (packageBindingStore, error) {
	count, err := readExpectedRecordCount(
		decoder, "bindings", expected,
	)
	if err != nil {
		return packageBindingStore{}, err
	}
	store := packageBindingStore{
		records: make([]storedBinding, 0, count),
	}
	for index := 0; index < count; index++ {
		if err := readBinaryBindingRecord(
			decoder, &store, authority,
		); err != nil {
			return packageBindingStore{}, fmt.Errorf(
				"decode semantic binary binding %d: %w",
				index,
				err,
			)
		}
	}
	return store, nil
}

func readBinaryBindingRecord(
	decoder *binaryShardDecoder,
	store *packageBindingStore,
	authority authorityRef,
) error {
	id, err := readIdentityReference[bindingRef](decoder, "binding id")
	if err != nil {
		return err
	}
	pkg, err := readIdentityReference[packageRef](
		decoder, "binding package",
	)
	if err != nil {
		return err
	}
	definition, err := readIdentityReference[definitionRef](
		decoder, "binding definition",
	)
	if err != nil {
		return err
	}
	role, err := readUnsignedAs[identity.SemanticBindingRole](
		decoder, "binding role",
	)
	if err != nil {
		return err
	}
	name, err := decoder.text("binding name")
	if err != nil {
		return err
	}
	typeID, err := readIdentityReference[typeRef](decoder, "binding type")
	if err != nil {
		return err
	}
	source, err := readIdentityReference[occurrenceRef](
		decoder, "binding source",
	)
	if err != nil {
		return err
	}
	start, count, err := readReferenceRange(
		decoder,
		"binding captures",
		&store.captures,
	)
	if err != nil {
		return err
	}
	store.records = append(store.records, storedBinding{
		id: id, pkg: pkg, definition: definition,
		role: role, name: name, typeID: typeID, source: source,
		captures:  definitionRefRange{start: start, count: count},
		authority: authority,
	})
	return nil
}

func writeBinaryUnsupported(
	encoder *binaryShardEncoder,
	store packageUnsupportedStore,
) {
	encoder.count(len(store.records))
	for _, record := range store.records {
		encoder.unsigned(uint64(record.id))
		encoder.unsigned(uint64(record.reason))
		encoder.text(record.evidence)
	}
}

func readBinaryUnsupported(
	decoder *binaryShardDecoder,
	expected int,
	authority authorityRef,
) (packageUnsupportedStore, error) {
	count, err := readExpectedRecordCount(
		decoder, "unsupported", expected,
	)
	if err != nil {
		return packageUnsupportedStore{}, err
	}
	store := packageUnsupportedStore{
		records: make([]storedUnsupported, 0, count),
	}
	for index := 0; index < count; index++ {
		id, readErr := readIdentityReference[unsupportedRef](
			decoder, "unsupported id",
		)
		if readErr != nil {
			return packageUnsupportedStore{}, readErr
		}
		reason, readErr := readUnsignedAs[UnsupportedReason](
			decoder, "unsupported reason",
		)
		if readErr != nil {
			return packageUnsupportedStore{}, readErr
		}
		evidence, readErr := decoder.text("unsupported evidence")
		if readErr != nil {
			return packageUnsupportedStore{}, readErr
		}
		store.records = append(store.records, storedUnsupported{
			id: id, reason: reason, evidence: evidence,
			authority: authority,
		})
	}
	return store, nil
}
