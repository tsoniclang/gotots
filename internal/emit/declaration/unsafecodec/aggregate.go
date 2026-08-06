package unsafecodec

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b *builder) arrayOperations(
	array *types.Array,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	elementStorage, err := b.storageType(array.Elem())
	if err != nil {
		return nil, nil, err
	}
	elementCodec, err := b.codec(array.Elem())
	if err != nil {
		return nil, nil, err
	}
	runtime, err := b.runtime(api.RuntimeArray, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	lengthType := b.factory.LiteralTypeNode(b.number(array.Len()))
	zero := b.memberCall(
		elementCodec,
		"decode",
		b.factory.NewExpression(
			b.id("Uint8Array"),
			nil,
			[]tsgo.Expression{b.property(elementCodec, "size")},
		),
	)
	result := b.call(
		b.property(runtime.Expression(b.factory), arraymember.Zero.Name()),
		[]tsgo.TypeNode{elementStorage, lengthType},
		b.number(array.Len()),
		zero,
	)
	readElement := b.memberCall(
		elementCodec,
		"read",
		b.id("bytes"),
		b.arrayElementOffset(array.Elem()),
	)
	read := b.readArrow(
		storage,
		b.variable("result", storage, result),
		b.indexLoop(array.Len(), b.factory.ExpressionStatement(
			b.memberCall(
				b.id("result"),
				arraymember.Set.Name(),
				b.id("index"),
				readElement,
			),
		)),
		b.factory.ReturnStatement(b.id("result")),
	)
	write := b.writeArrow(
		storage,
		b.indexLoop(array.Len(), b.factory.ExpressionStatement(
			b.memberCall(
				elementCodec,
				"write",
				b.memberCall(
					b.id("value"),
					arraymember.Get.Name(),
					b.id("index"),
				),
				b.id("bytes"),
				b.arrayElementOffset(array.Elem()),
			),
		)),
	)
	return read, write, nil
}

func (b *builder) arrayElementOffset(element types.Type) tsgo.Expression {
	stride := b.context.TypesSizes().Sizeof(element)
	return b.binary(
		b.id("offset"),
		tsgo.BinaryOperatorPlusToken,
		b.binary(
			b.id("index"),
			tsgo.BinaryOperatorAsteriskToken,
			b.number(stride),
		),
	)
}

func (b *builder) indexLoop(
	length int64,
	statements ...tsgo.Statement,
) tsgo.ForStatement {
	index := b.id("index")
	return b.factory.ForStatement(
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(
				index,
				nil,
				nil,
				b.number(0),
			)},
			tsgo.NodeFlagsLet,
		),
		b.binary(index, tsgo.BinaryOperatorLessThanToken, b.number(length)),
		b.factory.PostfixUnaryExpression(
			index,
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
		b.factory.Block(statements, true),
	)
}

func (b *builder) structOperations(
	sourceType types.Type,
	structure *types.Struct,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	if structure.NumFields() == 0 {
		return b.emptyStructOperations(sourceType, storage)
	}
	fields := make([]*types.Var, structure.NumFields())
	for index := range structure.NumFields() {
		fields[index] = structure.Field(index)
	}
	offsets := b.context.TypesSizes().Offsetsof(fields)
	properties := make([]tsgo.ObjectLiteralElementLike, 0, len(fields))
	writes := make([]tsgo.Statement, 0, len(fields))
	for index, field := range fields {
		codec, err := b.codec(field.Type())
		if err != nil {
			return nil, nil, err
		}
		fieldStorage, err := b.storageType(field.Type())
		if err != nil {
			return nil, nil, err
		}
		name, err := b.fieldName(field, index)
		if err != nil {
			return nil, nil, err
		}
		fieldOffset := b.binary(
			b.id("offset"),
			tsgo.BinaryOperatorPlusToken,
			b.number(offsets[index]),
		)
		properties = append(properties, b.factory.PropertyAssignment(
			nil,
			b.id(name),
			nil,
			fieldStorage,
			b.memberCall(codec, "read", b.id("bytes"), fieldOffset),
		))
		writes = append(writes, b.factory.ExpressionStatement(b.memberCall(
			codec,
			"write",
			b.property(b.id("value"), name),
			b.id("bytes"),
			fieldOffset,
		)))
	}
	read := b.readArrow(storage, b.factory.ReturnStatement(
		b.factory.ObjectLiteralExpression(properties, true),
	))
	return read, b.writeArrow(storage, writes...), nil
}

func (b *builder) emptyStructOperations(
	sourceType types.Type,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	zero, err := b.context.Values().Zero(
		b.context.WithRole(api.RoleStorageType),
		nil,
		sourceType,
	)
	if err != nil {
		return nil, nil, err
	}
	stored, err := b.context.Values().ToStorage(
		b.context.WithRole(api.RoleStorageType),
		nil,
		sourceType,
		zero,
	)
	if err != nil {
		return nil, nil, err
	}
	b.addRequests(stored.Requests())
	return b.readArrow(storage, b.factory.ReturnStatement(stored.Value())),
		b.writeArrow(storage), nil
}

func (b *builder) fieldName(field *types.Var, index int) (string, error) {
	if field.Name() == "_" {
		return fmt.Sprintf("$blank%d", index), nil
	}
	return b.context.Names().Member(field)
}
