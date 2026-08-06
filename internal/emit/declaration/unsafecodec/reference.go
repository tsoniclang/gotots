package unsafecodec

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	sliceruntime "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	unsafepointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/unsafepointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b *builder) pointerOperations(
	pointer *types.Pointer,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	logical, err := b.representedType(pointer.Elem())
	if err != nil {
		return nil, nil, err
	}
	elementStorage, err := b.storageType(pointer.Elem())
	if err != nil {
		return nil, nil, err
	}
	elementCodec, err := b.codec(pointer.Elem())
	if err != nil {
		return nil, nil, err
	}
	addressCodec, err := b.codec(types.Typ[types.Uintptr])
	if err != nil {
		return nil, nil, err
	}
	unsafePointer, err := b.runtime(api.RuntimeUnsafePointer, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	zero, err := b.integerZero()
	if err != nil {
		return nil, nil, err
	}
	address := b.memberCall(
		addressCodec,
		"read",
		b.id("bytes"),
		b.id("offset"),
	)
	unsafeValue := b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.FromIntegerName,
		),
		nil,
		address,
		zero,
	)
	read := b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.ToName,
		),
		[]tsgo.TypeNode{logical, elementStorage},
		unsafeValue,
		elementCodec,
	)
	unsafeValue = b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.FromName,
		),
		[]tsgo.TypeNode{logical, elementStorage},
		b.id("value"),
		elementCodec,
	)
	address = b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.ToIntegerName,
		),
		nil,
		unsafeValue,
		zero,
	)
	write := b.memberCall(
		addressCodec,
		"write",
		address,
		b.id("bytes"),
		b.id("offset"),
	)
	return b.readArrow(storage, b.factory.ReturnStatement(read)),
		b.writeArrow(storage, b.factory.ExpressionStatement(write)), nil
}

func (b *builder) unsafePointerOperations(
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	addressCodec, err := b.codec(types.Typ[types.Uintptr])
	if err != nil {
		return nil, nil, err
	}
	unsafePointer, err := b.runtime(api.RuntimeUnsafePointer, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	zero, err := b.integerZero()
	if err != nil {
		return nil, nil, err
	}
	address := b.memberCall(
		addressCodec,
		"read",
		b.id("bytes"),
		b.id("offset"),
	)
	read := b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.FromIntegerName,
		),
		nil,
		address,
		zero,
	)
	address = b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.ToIntegerName,
		),
		nil,
		b.id("value"),
		zero,
	)
	write := b.memberCall(
		addressCodec,
		"write",
		address,
		b.id("bytes"),
		b.id("offset"),
	)
	return b.readArrow(storage, b.factory.ReturnStatement(read)),
		b.writeArrow(storage, b.factory.ExpressionStatement(write)), nil
}

func (b *builder) stringOperations(
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	byteType := types.Typ[types.Uint8]
	byteLogical, err := b.representedType(byteType)
	if err != nil {
		return nil, nil, err
	}
	byteStorage, err := b.storageType(byteType)
	if err != nil {
		return nil, nil, err
	}
	byteCodec, err := b.codec(byteType)
	if err != nil {
		return nil, nil, err
	}
	addressCodec, err := b.codec(types.Typ[types.Uintptr])
	if err != nil {
		return nil, nil, err
	}
	lengthCodec, err := b.codec(types.Typ[types.Int])
	if err != nil {
		return nil, nil, err
	}
	unsafePointer, err := b.runtime(api.RuntimeUnsafePointer, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	unsafeString, err := b.runtime(api.RuntimeUnsafeString, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	unsafeStringData, err := b.runtime(
		api.RuntimeUnsafeStringData,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	zero, err := b.integerZero()
	if err != nil {
		return nil, nil, err
	}
	address := b.memberCall(addressCodec, "read", b.id("bytes"), b.id("offset"))
	lengthOffset := b.binary(
		b.id("offset"),
		tsgo.BinaryOperatorPlusToken,
		b.number(b.context.TypesSizes().Sizeof(types.Typ[types.Uintptr])),
	)
	length := b.memberCall(lengthCodec, "read", b.id("bytes"), lengthOffset)
	unsafeValue := b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.FromIntegerName,
		),
		nil,
		address,
		zero,
	)
	pointer := b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.ToName,
		),
		[]tsgo.TypeNode{byteLogical, byteStorage},
		unsafeValue,
		byteCodec,
	)
	read := b.call(
		unsafeString.Expression(b.factory),
		[]tsgo.TypeNode{byteStorage},
		pointer,
		length,
	)
	data := b.call(
		unsafeStringData.Expression(b.factory),
		[]tsgo.TypeNode{byteStorage},
		b.id("value"),
		b.integerConverter(byteType),
	)
	unsafeValue = b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.FromName,
		),
		[]tsgo.TypeNode{byteLogical, byteStorage},
		data,
		byteCodec,
	)
	address = b.call(
		b.property(
			unsafePointer.Expression(b.factory),
			unsafepointerruntime.ToIntegerName,
		),
		nil,
		unsafeValue,
		zero,
	)
	writeAddress := b.factory.ExpressionStatement(b.memberCall(
		addressCodec,
		"write",
		address,
		b.id("bytes"),
		b.id("offset"),
	))
	writeLength := b.factory.ExpressionStatement(b.memberCall(
		lengthCodec,
		"write",
		b.convertInteger(b.property(b.id("value"), "length")),
		b.id("bytes"),
		lengthOffset,
	))
	return b.readArrow(storage, b.factory.ReturnStatement(read)),
		b.writeArrow(storage, writeAddress, writeLength), nil
}

func (b *builder) sliceOperations(
	slice *types.Slice,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	logical, err := b.representedType(slice.Elem())
	if err != nil {
		return nil, nil, err
	}
	elementStorage, err := b.storageType(slice.Elem())
	if err != nil {
		return nil, nil, err
	}
	elementCodec, err := b.codec(slice.Elem())
	if err != nil {
		return nil, nil, err
	}
	addressCodec, err := b.codec(types.Typ[types.Uintptr])
	if err != nil {
		return nil, nil, err
	}
	lengthCodec, err := b.codec(types.Typ[types.Int])
	if err != nil {
		return nil, nil, err
	}
	unsafePointer, err := b.runtime(api.RuntimeUnsafePointer, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	unsafeSliceData, err := b.runtime(api.RuntimeUnsafeSliceData, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	unsafeSliceHeader, err := b.runtime(api.RuntimeUnsafeSliceHeader, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	zero, err := b.integerZero()
	if err != nil {
		return nil, nil, err
	}
	wordSize := b.context.TypesSizes().Sizeof(types.Typ[types.Uintptr])
	address := b.memberCall(addressCodec, "read", b.id("bytes"), b.id("offset"))
	lengthOffset := b.binary(b.id("offset"), tsgo.BinaryOperatorPlusToken, b.number(wordSize))
	capacityOffset := b.binary(b.id("offset"), tsgo.BinaryOperatorPlusToken, b.number(wordSize*2))
	length := b.memberCall(lengthCodec, "read", b.id("bytes"), lengthOffset)
	capacity := b.memberCall(lengthCodec, "read", b.id("bytes"), capacityOffset)
	unsafeValue := b.call(
		b.property(unsafePointer.Expression(b.factory), unsafepointerruntime.FromIntegerName),
		nil,
		address,
		zero,
	)
	pointer := b.call(
		b.property(unsafePointer.Expression(b.factory), unsafepointerruntime.ToName),
		[]tsgo.TypeNode{logical, elementStorage},
		unsafeValue,
		elementCodec,
	)
	read := b.call(
		unsafeSliceHeader.Expression(b.factory),
		[]tsgo.TypeNode{logical, elementStorage},
		pointer,
		length,
		capacity,
	)
	data := b.call(
		unsafeSliceData.Expression(b.factory),
		[]tsgo.TypeNode{logical, elementStorage},
		b.id("value"),
	)
	unsafeValue = b.call(
		b.property(unsafePointer.Expression(b.factory), unsafepointerruntime.FromName),
		[]tsgo.TypeNode{logical, elementStorage},
		data,
		elementCodec,
	)
	address = b.call(
		b.property(unsafePointer.Expression(b.factory), unsafepointerruntime.ToIntegerName),
		nil,
		unsafeValue,
		zero,
	)
	writes := []tsgo.Statement{
		b.factory.ExpressionStatement(b.memberCall(
			addressCodec, "write", address, b.id("bytes"), b.id("offset"),
		)),
		b.factory.ExpressionStatement(b.memberCall(
			lengthCodec,
			"write",
			b.convertInteger(b.property(b.id("value"), sliceruntime.MemberName(sliceruntime.MemberLength))),
			b.id("bytes"),
			lengthOffset,
		)),
		b.factory.ExpressionStatement(b.memberCall(
			lengthCodec,
			"write",
			b.convertInteger(b.property(b.id("value"), sliceruntime.MemberName(sliceruntime.MemberCapacity))),
			b.id("bytes"),
			capacityOffset,
		)),
	}
	return b.readArrow(storage, b.factory.ReturnStatement(read)),
		b.writeArrow(storage, writes...), nil
}

func (b *builder) integerConverter(sourceType types.Type) tsgo.Expression {
	intrinsic := api.TargetIntrinsicNumber
	if b.context.ScalarABI().UsesBigInt(sourceType) {
		intrinsic = api.TargetIntrinsicBigInt
	}
	return intrinsic.Expression(b.factory)
}
