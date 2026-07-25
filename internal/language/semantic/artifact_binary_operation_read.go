package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func readBinaryOperations(
	decoder *binaryShardDecoder,
	expected int,
) (packageOperationStore, error) {
	count, err := readExpectedRecordCount(
		decoder, "operations", expected,
	)
	if err != nil {
		return packageOperationStore{}, err
	}
	store := packageOperationStore{
		records: make([]storedOperation, 0, count),
	}
	for index := 0; index < count; index++ {
		if err := readBinaryOperationRecord(decoder, &store); err != nil {
			return packageOperationStore{}, fmt.Errorf(
				"decode semantic binary operation %d: %w",
				index,
				err,
			)
		}
	}
	return store, nil
}

func readBinaryOperationRecord(
	decoder *binaryShardDecoder,
	store *packageOperationStore,
) error {
	record, err := readBinaryOperationCore(decoder)
	if err != nil {
		return err
	}
	record.constant, err = readBinaryOperationConstant(decoder, store)
	if err != nil {
		return err
	}
	record.object, err = readBinaryOperationObject(decoder, store)
	if err != nil {
		return err
	}
	record.selection, err = readBinaryOperationSelection(decoder, store)
	if err != nil {
		return err
	}
	record.instance, err = readBinaryOperationInstance(decoder, store)
	if err != nil {
		return err
	}
	start, count, err := readReferenceRange(
		decoder,
		"operation operands",
		&store.operands,
	)
	if err != nil {
		return err
	}
	record.operands = occurrenceRefRange{start: start, count: count}
	start, count, err = readReferenceRange(
		decoder,
		"operation definitions",
		&store.definitions,
	)
	if err != nil {
		return err
	}
	record.definitions = definitionRefRange{start: start, count: count}
	record.implicit, err = readBinaryImplicitOperations(decoder, store)
	if err != nil {
		return err
	}
	record.controlTarget, err = readIdentityReference[operationRef](
		decoder, "operation control target",
	)
	if err != nil {
		return err
	}
	record.label, err = readIdentityReference[bindingRef](
		decoder, "operation label",
	)
	if err != nil {
		return err
	}
	store.records = append(store.records, record)
	return nil
}

func readBinaryOperationCore(
	decoder *binaryShardDecoder,
) (storedOperation, error) {
	var (
		record storedOperation
		err    error
	)
	record.id, err = readIdentityReference[operationRef](
		decoder, "operation id",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.kind, err = readUnsignedAs[OperationKind](
		decoder, "operation kind",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.syntax, err = readUnsignedAs[catalog.Kind](
		decoder, "operation syntax",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.variant, err = readUnsignedAs[catalog.Variant](
		decoder, "operation variant",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.role, err = readUnsignedAs[catalog.Role](
		decoder, "operation role",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.token, err = readUnsignedAs[catalog.TokenKind](
		decoder, "operation token",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.mode, err = readUnsignedAs[ValueMode](
		decoder, "operation mode",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.arity, err = readUnsignedAs[ResultArity](
		decoder, "operation arity",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.place, err = readUnsignedAs[PlaceKind](
		decoder, "operation place",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.resultType, err = readIdentityReference[typeRef](
		decoder, "operation result type",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.expectedType, err = readIdentityReference[typeRef](
		decoder, "operation expected type",
	)
	if err != nil {
		return storedOperation{}, err
	}
	record.addressable, err = decoder.boolean("operation addressable")
	if err != nil {
		return storedOperation{}, err
	}
	record.assignable, err = decoder.boolean("operation assignable")
	if err != nil {
		return storedOperation{}, err
	}
	record.hasOk, err = decoder.boolean("operation has-ok")
	return record, err
}

func readBinaryOperationConstant(
	decoder *binaryShardDecoder,
	store *packageOperationStore,
) (constantRef, error) {
	present, err := decoder.boolean("operation constant present")
	if err != nil || !present {
		return 0, err
	}
	kind, err := readUnsignedAs[ConstantKind](
		decoder, "operation constant kind",
	)
	if err != nil {
		return 0, err
	}
	exact, err := decoder.text("operation constant exact value")
	if err != nil {
		return 0, err
	}
	store.constants = append(store.constants, storedConstant{
		kind: kind, exact: exact,
	})
	return constantRef(len(store.constants)), nil
}

func readBinaryOperationObject(
	decoder *binaryShardDecoder,
	store *packageOperationStore,
) (objectReferenceRef, error) {
	present, err := decoder.boolean("operation object present")
	if err != nil || !present {
		return 0, err
	}
	object, err := readBinaryStoredObject(decoder)
	if err != nil {
		return 0, err
	}
	store.objects = append(store.objects, object)
	return objectReferenceRef(len(store.objects)), nil
}

func readBinaryStoredObject(
	decoder *binaryShardDecoder,
) (storedObjectReference, error) {
	kind, err := readUnsignedAs[ObjectReferenceKind](
		decoder, "object reference kind",
	)
	if err != nil {
		return storedObjectReference{}, err
	}
	declaration, err := readIdentityReference[declarationRef](
		decoder, "object declaration",
	)
	if err != nil {
		return storedObjectReference{}, err
	}
	binding, err := readIdentityReference[bindingRef](
		decoder, "object binding",
	)
	return storedObjectReference{
		kind: kind, declaration: declaration, binding: binding,
	}, err
}

func readBinaryOperationSelection(
	decoder *binaryShardDecoder,
	store *packageOperationStore,
) (selectionRef, error) {
	present, err := decoder.boolean("operation selection present")
	if err != nil || !present {
		return 0, err
	}
	kind, err := readUnsignedAs[SelectionKind](
		decoder, "selection kind",
	)
	if err != nil {
		return 0, err
	}
	receiver, err := readIdentityReference[typeRef](
		decoder, "selection receiver",
	)
	if err != nil {
		return 0, err
	}
	object, err := readIdentityReference[declarationRef](
		decoder, "selection object",
	)
	if err != nil {
		return 0, err
	}
	count, err := decoder.count("selection index")
	if err != nil {
		return 0, err
	}
	start := uint64(len(store.selectionIndexes))
	for index := 0; index < count; index++ {
		value, readErr := readSignedInt(decoder, "selection index value")
		if readErr != nil {
			return 0, readErr
		}
		store.selectionIndexes = append(
			store.selectionIndexes, value,
		)
	}
	indirect, err := decoder.boolean("selection indirect")
	if err != nil {
		return 0, err
	}
	store.selections = append(store.selections, storedSelection{
		kind: kind, receiver: receiver, object: object,
		index:    integerRange{start: start, count: uint64(count)},
		indirect: indirect,
	})
	return selectionRef(len(store.selections)), nil
}

func readBinaryOperationInstance(
	decoder *binaryShardDecoder,
	store *packageOperationStore,
) (instanceRef, error) {
	present, err := decoder.boolean("operation instance present")
	if err != nil || !present {
		return 0, err
	}
	target, err := readBinaryStoredObject(decoder)
	if err != nil {
		return 0, err
	}
	store.objects = append(store.objects, target)
	targetReference := objectReferenceRef(len(store.objects))
	start, count, err := readReferenceRange(
		decoder,
		"instance types",
		&store.instanceTypes,
	)
	if err != nil {
		return 0, err
	}
	signature, err := readIdentityReference[typeRef](
		decoder, "instance signature",
	)
	if err != nil {
		return 0, err
	}
	store.instances = append(store.instances, storedInstance{
		target:    targetReference,
		types:     typeRefRange{start: start, count: count},
		signature: signature,
	})
	return instanceRef(len(store.instances)), nil
}

func readBinaryImplicitOperations(
	decoder *binaryShardDecoder,
	store *packageOperationStore,
) (implicitOperationRange, error) {
	count, err := decoder.count("implicit operations")
	if err != nil {
		return implicitOperationRange{}, err
	}
	start := uint64(len(store.implicit))
	for index := 0; index < count; index++ {
		implicit, readErr := readBinaryImplicitOperation(decoder)
		if readErr != nil {
			return implicitOperationRange{}, readErr
		}
		store.implicit = append(store.implicit, implicit)
	}
	return implicitOperationRange{
		start: start, count: uint64(count),
	}, nil
}

func readBinaryImplicitOperation(
	decoder *binaryShardDecoder,
) (storedImplicitOperation, error) {
	kind, err := readUnsignedAs[catalog.ImplicitOp](
		decoder, "implicit operation kind",
	)
	if err != nil {
		return storedImplicitOperation{}, err
	}
	site, err := readIdentityReference[occurrenceRef](
		decoder, "implicit operation site",
	)
	if err != nil {
		return storedImplicitOperation{}, err
	}
	ordinal, err := readSignedInt(
		decoder, "implicit operation ordinal",
	)
	if err != nil {
		return storedImplicitOperation{}, err
	}
	source, err := readIdentityReference[typeRef](
		decoder, "implicit operation source",
	)
	if err != nil {
		return storedImplicitOperation{}, err
	}
	target, err := readIdentityReference[typeRef](
		decoder, "implicit operation target",
	)
	return storedImplicitOperation{
		kind: kind, site: site, ordinal: ordinal,
		source: source, target: target,
	}, err
}
