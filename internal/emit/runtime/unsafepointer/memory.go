package unsafepointer

import (
	unsafecodecruntime "github.com/tsoniclang/gotots/internal/emit/runtime/unsafecodec"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) from() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	pointerType := b.typeReference(b.pointerName, typeL, typeS)
	value := b.id("value")
	memory := b.id("memory")
	located := b.id("located")
	region := b.id("region")
	rootKey := b.id("rootKey")
	root := b.id("root")
	result := b.id("result")
	totalLength := b.id("totalLength")
	readBytes := b.id("readBytes")
	writeBytes := b.id("writeBytes")
	return b.method(
		[]tsgo.ModifierLike{b.factory.PublicKeyword(), b.factory.StaticKeyword()},
		FromName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.optional(pointerType), nil),
			b.parameter("codec", b.codecType(typeS), nil),
		},
		b.optional(b.unsafeType()),
		b.factory.IfStatement(
			b.binary(value, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
			b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(b.undefined())}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"memory",
			nil,
			b.factory.CallExpression(
				b.id(b.pointerMemoryName),
				nil,
				[]tsgo.TypeNode{typeL, typeS},
				[]tsgo.Expression{value},
				tsgo.NodeFlagsNone,
			),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"located",
			nil,
			b.call(
				b.property(b.id(b.className), "locations"),
				"get",
				b.element(memory, b.number("0")),
			),
		),
		b.factory.IfStatement(
			b.binary(located, tsgo.BinaryOperatorExclamationEqualsEqualsToken, b.undefined()),
			b.factory.Block([]tsgo.Statement{
				b.factory.ExpressionStatement(b.call(
					b.id("codec"),
					unsafecodecruntime.BindName,
					located,
					memory,
				)),
				b.factory.ExpressionStatement(b.bindPointerSync(value, located)),
				b.factory.ReturnStatement(located),
			}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"region",
			nil,
			b.element(memory, b.number("3")),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"rootKey",
			b.objectType(),
			b.factory.ConditionalExpression(
				b.binary(region, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
				b.factory.QuestionToken(),
				b.element(memory, b.number("0")),
				b.factory.ColonToken(),
				b.element(region, b.number("0")),
			),
		),
		b.variable(
			tsgo.NodeFlagsLet,
			"root",
			b.optional(b.unsafeType()),
			b.call(b.property(b.id(b.className), "roots"), "get", rootKey),
		),
		b.factory.IfStatement(
			b.binary(root, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
			b.factory.Block([]tsgo.Statement{
				b.variable(tsgo.NodeFlagsLet, "totalLength", b.numberType(), nil),
				b.variable(tsgo.NodeFlagsLet, "readBytes", b.readBytesType(), nil),
				b.variable(tsgo.NodeFlagsLet, "writeBytes", b.writeBytesType(), nil),
				b.factory.IfStatement(
					b.binary(region, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
					b.factory.Block(b.scalarMemoryStatements(
						memory,
						totalLength,
						readBytes,
						writeBytes,
					), true),
					b.factory.Block(b.regionMemoryStatements(
						region,
						totalLength,
						readBytes,
						writeBytes,
					), true),
				),
				b.factory.ExpressionStatement(b.assign(
					root,
					b.factory.NewExpression(
						b.id(b.className),
						nil,
						[]tsgo.Expression{
							b.property(b.id(b.className), "nextBase"),
							b.number("0"),
							totalLength,
							readBytes,
							writeBytes,
							b.factory.NewExpression(b.id("Map"), nil, nil),
							b.factory.ArrayLiteralExpression(nil, false),
							b.factory.ArrayLiteralExpression(nil, false),
						},
					),
				)),
				b.factory.ExpressionStatement(b.assign(
					b.property(b.id(b.className), "nextBase"),
					b.binary(
						b.binary(
							b.property(b.id(b.className), "nextBase"),
							tsgo.BinaryOperatorPlusToken,
							totalLength,
						),
						tsgo.BinaryOperatorPlusToken,
						b.number("4096"),
					),
				)),
				b.factory.ExpressionStatement(b.call(
					b.property(b.id(b.className), "roots"),
					"set",
					rootKey,
					root,
				)),
				b.factory.ExpressionStatement(b.call(
					b.property(b.id(b.className), "allocations"),
					"push",
					root,
				)),
			}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"result",
			b.unsafeType(),
			b.call(
				root,
				atName,
				b.factory.ConditionalExpression(
					b.binary(region, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
					b.factory.QuestionToken(),
					b.number("0"),
					b.factory.ColonToken(),
					b.binary(
						b.element(region, b.number("1")),
						tsgo.BinaryOperatorAsteriskToken,
						b.property(b.id("codec"), "size"),
					),
				),
			),
		),
		b.factory.ExpressionStatement(b.call(
			b.id("codec"),
			unsafecodecruntime.BindName,
			result,
			memory,
		)),
		b.factory.ExpressionStatement(b.bindPointerSync(value, result)),
		b.factory.ReturnStatement(result),
	)
}

func (b builder) scalarMemoryStatements(
	memory tsgo.Expression,
	totalLength tsgo.Expression,
	readBytes tsgo.Expression,
	writeBytes tsgo.Expression,
) []tsgo.Statement {
	read := b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("offset", b.numberType(), nil),
			b.parameter("length", b.numberType(), nil),
		},
		nil,
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"bytes",
				b.byteArrayType(),
				b.call(
					b.id("codec"),
					unsafecodecruntime.EncodeName,
					b.factory.CallExpression(
						b.element(memory, b.number("1")),
						nil,
						nil,
						nil,
						tsgo.NodeFlagsNone,
					),
				),
			),
			b.factory.ReturnStatement(b.call(
				b.id("bytes"),
				"slice",
				b.id("offset"),
				b.binary(
					b.id("offset"),
					tsgo.BinaryOperatorPlusToken,
					b.id("length"),
				),
			)),
		}, true),
	)
	write := b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("offset", b.numberType(), nil),
			b.parameter("replacement", b.byteArrayType(), nil),
		},
		nil,
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"bytes",
				b.byteArrayType(),
				b.call(
					b.id("codec"),
					unsafecodecruntime.EncodeName,
					b.factory.CallExpression(
						b.element(memory, b.number("1")),
						nil,
						nil,
						nil,
						tsgo.NodeFlagsNone,
					),
				),
			),
			b.factory.ExpressionStatement(b.call(
				b.id("bytes"),
				"set",
				b.id("replacement"),
				b.id("offset"),
			)),
			b.factory.ExpressionStatement(b.factory.CallExpression(
				b.element(memory, b.number("2")),
				nil,
				nil,
				[]tsgo.Expression{b.call(
					b.id("codec"),
					unsafecodecruntime.DecodeName,
					b.id("bytes"),
				)},
				tsgo.NodeFlagsNone,
			)),
		}, true),
	)
	return []tsgo.Statement{
		b.factory.ExpressionStatement(b.assign(
			totalLength,
			b.property(b.id("codec"), "size"),
		)),
		b.factory.ExpressionStatement(b.assign(readBytes, read)),
		b.factory.ExpressionStatement(b.assign(writeBytes, write)),
	}
}

func (b builder) regionMemoryStatements(
	region tsgo.Expression,
	totalLength tsgo.Expression,
	readBytes tsgo.Expression,
	writeBytes tsgo.Expression,
) []tsgo.Statement {
	backing := b.id("backing")
	return []tsgo.Statement{
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			nil,
			b.element(region, b.number("0")),
		),
		b.factory.ExpressionStatement(b.assign(
			totalLength,
			b.binary(
				b.property(backing, "length"),
				tsgo.BinaryOperatorAsteriskToken,
				b.property(b.id("codec"), "size"),
			),
		)),
		b.factory.ExpressionStatement(b.assign(
			readBytes,
			b.regionReadArrow(backing, totalLength),
		)),
		b.factory.ExpressionStatement(b.assign(
			writeBytes,
			b.regionWriteArrow(backing, totalLength),
		)),
	}
}

func (b builder) regionReadArrow(
	backing tsgo.Expression,
	totalLength tsgo.Expression,
) tsgo.ArrowFunction {
	return b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("offset", b.numberType(), nil),
			b.parameter("length", b.numberType(), nil),
		},
		nil,
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"bytes",
				b.byteArrayType(),
				b.factory.NewExpression(
					b.id("Uint8Array"),
					nil,
					[]tsgo.Expression{totalLength},
				),
			),
			b.regionCodecLoop(backing, b.id("bytes"), false),
			b.factory.ReturnStatement(b.call(
				b.id("bytes"),
				"slice",
				b.id("offset"),
				b.binary(
					b.id("offset"),
					tsgo.BinaryOperatorPlusToken,
					b.id("length"),
				),
			)),
		}, true),
	)
}

func (b builder) regionWriteArrow(
	backing tsgo.Expression,
	totalLength tsgo.Expression,
) tsgo.ArrowFunction {
	return b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("offset", b.numberType(), nil),
			b.parameter("replacement", b.byteArrayType(), nil),
		},
		nil,
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"bytes",
				b.byteArrayType(),
				b.factory.NewExpression(
					b.id("Uint8Array"),
					nil,
					[]tsgo.Expression{totalLength},
				),
			),
			b.regionCodecLoop(backing, b.id("bytes"), false),
			b.factory.ExpressionStatement(b.call(
				b.id("bytes"),
				"set",
				b.id("replacement"),
				b.id("offset"),
			)),
			b.regionCodecLoop(backing, b.id("bytes"), true),
		}, true),
	)
}

func (b builder) regionCodecLoop(
	backing tsgo.Expression,
	bytes tsgo.Expression,
	decode bool,
) tsgo.ForStatement {
	index := b.id("index")
	offset := b.binary(
		index,
		tsgo.BinaryOperatorAsteriskToken,
		b.property(b.id("codec"), "size"),
	)
	var statement tsgo.Statement
	if decode {
		statement = b.factory.ExpressionStatement(b.assign(
			b.element(backing, index),
			b.call(
				b.id("codec"),
				unsafecodecruntime.ReadName,
				bytes,
				offset,
			),
		))
	} else {
		statement = b.factory.ExpressionStatement(b.call(
			b.id("codec"),
			unsafecodecruntime.WriteName,
			b.call(b.id(b.denseIndexName), "get", backing, index),
			bytes,
			offset,
		))
	}
	return b.factory.ForStatement(
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(
				index,
				nil,
				nil,
				b.number("0"),
			)},
			tsgo.NodeFlagsLet,
		),
		b.binary(index, tsgo.BinaryOperatorLessThanToken, b.property(backing, "length")),
		b.factory.PostfixUnaryExpression(
			index,
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
		b.factory.Block([]tsgo.Statement{statement}, true),
	)
}
