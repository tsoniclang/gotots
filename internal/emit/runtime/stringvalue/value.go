package stringvalue

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	FromTextMember     = "fromText"
	FromRegionMember   = "fromRegion"
	TextMember         = "text"
	DataMember         = "data"
	SourceLengthMember = "sourceLength"
	SliceMember        = "slice"
	ReadMember         = "read"
)

func Build(factory tsgo.Factory, symbol api.RuntimeSymbol) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	target := builder{factory: factory}
	switch symbol {
	case api.RuntimeStringTextBacking:
		return target.textBacking(contract.ExportedName()), nil
	case api.RuntimeStringPointerBacking:
		return target.pointerBacking(contract.ExportedName()), nil
	case api.RuntimeStringValue:
		return target.value(contract.ExportedName()), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func (b builder) value(name string) tsgo.Statement {
	byteType := b.reference("uint8")
	stringType := b.reference(name)
	backingType := b.factory.UnionTypeNode([]tsgo.TypeNode{b.reference("GoStringTextBacking"), b.reference("GoStringPointerBacking")})
	newValue := func(backing, offset, length tsgo.Expression) tsgo.Expression {
		return b.factory.NewExpression(b.id(name), nil, []tsgo.Expression{backing, offset, length})
	}
	start, end := b.id("start"), b.id("end")
	invalidSlice := b.binary(b.binary(start, tsgo.BinaryOperatorLessThanToken, b.number("0")), tsgo.BinaryOperatorBarBarToken,
		b.binary(b.binary(end, tsgo.BinaryOperatorLessThanToken, start), tsgo.BinaryOperatorBarBarToken,
			b.binary(end, tsgo.BinaryOperatorGreaterThanToken, b.own("count"))))
	index := b.id("index")
	invalidIndex := b.binary(b.binary(index, tsgo.BinaryOperatorLessThanToken, b.number("0")), tsgo.BinaryOperatorBarBarToken,
		b.binary(index, tsgo.BinaryOperatorGreaterThanEqualsToken, b.own("count")))
	region := b.id("region")
	return typescriptclass.Declaration(b.factory, []tsgo.ModifierLike{b.factory.ExportKeyword()}, b.id(name), nil, nil, []tsgo.ClassElement{
		b.factory.PropertyDeclaration([]tsgo.ModifierLike{b.factory.StaticKeyword(), b.factory.ReadonlyKeyword()}, b.id("empty"), nil,
			stringType, newValue(b.factory.NewExpression(b.id("GoStringTextBacking"), nil, []tsgo.Expression{b.text("")}), b.number("0"), b.number("0"))),
		b.factory.ConstructorDeclaration([]tsgo.ModifierLike{b.factory.PrivateKeyword()}, nil, []tsgo.ParameterDeclaration{
			b.fieldParameter("backing", backingType), b.fieldParameter("offset", b.integerType()), b.fieldParameter("count", b.integerType()),
		}, nil, b.factory.Block(nil, true)),
		b.staticMethod(FromTextMember, []tsgo.ParameterDeclaration{b.parameter("value", b.textType())}, stringType,
			b.factory.IfStatement(b.binary(b.member(b.id("value"), "length"), tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.number("0")),
				b.result(b.member(b.id(name), "empty")), nil),
			b.result(newValue(b.factory.NewExpression(b.id("GoStringTextBacking"), nil, []tsgo.Expression{b.id("value")}), b.number("0"), b.member(b.id("value"), "length")))),
		b.staticMethod(FromRegionMember, []tsgo.ParameterDeclaration{
			b.parameter("region", b.optional(memoryview.RegionType(b.factory, byteType))), b.parameter("length", b.integerType()),
		}, stringType,
			b.factory.IfStatement(b.binary(b.id("length"), tsgo.BinaryOperatorLessThanToken, b.number("0")), b.fail("unsafe string length is negative"), nil),
			b.factory.IfStatement(b.binary(region, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()), b.factory.Block([]tsgo.Statement{
				b.factory.IfStatement(b.binary(b.id("length"), tsgo.BinaryOperatorExclamationEqualsToken, b.number("0")), b.fail("unsafe string on nil pointer"), nil),
				b.result(b.call(b.member(b.id(name), FromTextMember), b.text(""))),
			}, true), nil),
			b.result(newValue(b.factory.NewExpression(b.id("GoStringPointerBacking"), nil, []tsgo.Expression{region}), b.number("0"), b.id("length")))),
		b.method(SourceLengthMember, nil, b.integerType(), b.result(b.own("count"))),
		b.method(TextMember, nil, b.textType(), b.result(b.call(b.member(b.own("backing"), TextMember), b.own("offset"), b.own("count")))),
		b.method(DataMember, nil, b.optional(b.reference("Pointer", byteType)), b.result(b.call(b.member(b.own("backing"), "address"), b.own("offset")))),
		b.method(ReadMember, []tsgo.ParameterDeclaration{b.parameter("index", b.integerType())}, byteType,
			b.factory.IfStatement(invalidIndex, b.fail("Go string index out of range"), nil),
			b.result(b.call(b.member(b.own("backing"), ReadMember), b.exactArithmetic(b.own("offset"), tsgo.BinaryOperatorPlusToken, index)))),
		b.method(SliceMember, []tsgo.ParameterDeclaration{b.parameter("low", b.integerType()),
			b.factory.ParameterDeclaration(nil, nil, b.id("high"), b.factory.QuestionToken(), b.integerType(), nil)}, stringType,
			b.variable("start", b.id("low")),
			b.variable("end", b.binary(b.id("high"), tsgo.BinaryOperatorQuestionQuestionToken, b.own("count"))),
			b.factory.IfStatement(invalidSlice, b.fail("Go string slice bounds out of range"), nil),
			b.result(newValue(b.own("backing"), b.exactArithmetic(b.own("offset"), tsgo.BinaryOperatorPlusToken, start),
				b.exactArithmetic(end, tsgo.BinaryOperatorMinusToken, start)))),
	})
}
