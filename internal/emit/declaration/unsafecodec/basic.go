package unsafecodec

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	complexruntime "github.com/tsoniclang/gotots/internal/emit/runtime/complex"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b *builder) basicOperations(
	sourceType types.Type,
	basic *types.Basic,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	switch basic.Kind() {
	case types.Bool:
		read, write := b.boolOperations(storage)
		return read, write, nil
	case types.String:
		return b.stringOperations(storage)
	case types.Complex64, types.Complex128:
		return b.complexOperations(basic.Kind(), storage)
	case types.UnsafePointer:
		return b.unsafePointerOperations(storage)
	}
	if basic.Info()&types.IsInteger != 0 {
		return b.integerOperations(sourceType, basic, storage)
	}
	if basic.Info()&types.IsFloat != 0 {
		return b.floatOperations(sourceType, storage)
	}
	return nil, nil, &api.GeneratedArtifactShapeError{
		Reason: "unsafe-codec basic layout is unsupported",
	}
}

func (b *builder) boolOperations(
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression) {
	readValue := b.binary(
		b.memberCall(b.dataView(), "getUint8", b.id("offset")),
		tsgo.BinaryOperatorExclamationEqualsEqualsToken,
		b.number(0),
	)
	writeValue := b.factory.ConditionalExpression(
		b.id("value"),
		b.factory.QuestionToken(),
		b.number(1),
		b.factory.ColonToken(),
		b.number(0),
	)
	return b.readArrow(storage, b.factory.ReturnStatement(readValue)),
		b.writeArrow(storage, b.factory.ExpressionStatement(
			b.memberCall(
				b.dataView(),
				"setUint8",
				b.id("offset"),
				writeValue,
			),
		))
}

func (b *builder) integerOperations(
	sourceType types.Type,
	basic *types.Basic,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	size := b.context.TypesSizes().Sizeof(sourceType)
	unsigned := basic.Info()&types.IsUnsigned != 0
	get, set, wide, err := integerDataViewMembers(size, unsigned)
	if err != nil {
		return nil, nil, err
	}
	readArguments := []tsgo.Expression{b.id("offset")}
	if size > 1 {
		readArguments = append(readArguments, b.littleEndian())
	}
	readValue := tsgo.Expression(
		b.memberCall(b.dataView(), get, readArguments...),
	)
	carrier, represented := integervalue.DescribeUnderlying(
		b.context.TypesSizes(),
		sourceType,
	)
	if !represented {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "unsafe-codec integer carrier is unavailable",
		}
	}
	bigIntCarrier := integervalue.UsesBigInt(
		b.context.IntegerRepresentation(),
		carrier,
	)
	if wide && !bigIntCarrier {
		readValue = b.call(
			api.TargetIntrinsicNumber.Expression(b.factory),
			nil,
			readValue,
		)
	}
	writeValue := tsgo.Expression(b.id("value"))
	if wide && !bigIntCarrier {
		writeValue = b.call(
			api.TargetIntrinsicBigInt.Expression(b.factory),
			nil,
			writeValue,
		)
	}
	writeArguments := []tsgo.Expression{b.id("offset"), writeValue}
	if size > 1 {
		writeArguments = append(writeArguments, b.littleEndian())
	}
	return b.readArrow(storage, b.factory.ReturnStatement(readValue)),
		b.writeArrow(storage, b.factory.ExpressionStatement(
			b.memberCall(b.dataView(), set, writeArguments...),
		)), nil
}

func integerDataViewMembers(
	size int64,
	unsigned bool,
) (string, string, bool, error) {
	prefix := "Int"
	if unsigned {
		prefix = "Uint"
	}
	switch size {
	case 1, 2, 4:
		bits := size * 8
		return "get" + prefix + decimal(bits),
			"set" + prefix + decimal(bits), false, nil
	case 8:
		return "getBig" + prefix + "64", "setBig" + prefix + "64", true, nil
	default:
		return "", "", false, &api.GeneratedArtifactShapeError{
			Reason: "unsafe-codec integer width is unsupported",
		}
	}
}

func (b *builder) floatOperations(
	sourceType types.Type,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	size := b.context.TypesSizes().Sizeof(sourceType)
	member := ""
	switch size {
	case 4:
		member = "Float32"
	case 8:
		member = "Float64"
	default:
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "unsafe-codec float width is unsupported",
		}
	}
	read := b.memberCall(
		b.dataView(),
		"get"+member,
		b.id("offset"),
		b.littleEndian(),
	)
	write := b.memberCall(
		b.dataView(),
		"set"+member,
		b.id("offset"),
		b.id("value"),
		b.littleEndian(),
	)
	return b.readArrow(storage, b.factory.ReturnStatement(read)),
		b.writeArrow(storage, b.factory.ExpressionStatement(write)), nil
}

func (b *builder) complexOperations(
	kind types.BasicKind,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	symbol := api.RuntimeComplex64
	width := int64(4)
	if kind == types.Complex128 {
		symbol = api.RuntimeComplex128
		width = 8
	}
	reference, err := b.runtime(symbol, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	readPart := func(delta int64) tsgo.Expression {
		return b.memberCall(
			b.dataView(),
			"getFloat"+decimal(width*8),
			b.binary(b.id("offset"), tsgo.BinaryOperatorPlusToken, b.number(delta)),
			b.littleEndian(),
		)
	}
	read := b.call(
		b.property(reference.Expression(b.factory), complexruntime.MakeMember),
		nil,
		readPart(0),
		readPart(width),
	)
	writePart := func(member string, delta int64) tsgo.Statement {
		return b.factory.ExpressionStatement(b.memberCall(
			b.dataView(),
			"setFloat"+decimal(width*8),
			b.binary(b.id("offset"), tsgo.BinaryOperatorPlusToken, b.number(delta)),
			b.property(b.id("value"), member),
			b.littleEndian(),
		))
	}
	return b.readArrow(storage, b.factory.ReturnStatement(read)),
		b.writeArrow(
			storage,
			writePart(complexruntime.RealMember, 0),
			writePart(complexruntime.ImagMember, width),
		), nil
}

func decimal(value int64) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	index := len(digits)
	for value != 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}
