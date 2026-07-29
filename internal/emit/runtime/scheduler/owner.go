package scheduler

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	BlockMember = "block"
	SpawnMember = "spawn"
	RunMember   = "run"
)

func Build(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	schedulerName string,
	panicName string,
) (tsgo.Statement, error) {
	if symbol != api.RuntimeScheduler ||
		schedulerName == "" ||
		panicName == "" {
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
	return builder{
		factory:       factory,
		schedulerName: schedulerName,
		panicName:     panicName,
	}.class(), nil
}

func (b builder) class() tsgo.ClassDeclaration {
	return b.factory.ClassDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		b.id(b.schedulerName),
		nil,
		nil,
		[]tsgo.ClassElement{
			b.propertyDeclaration("active", b.numberType(), b.number("0")),
			b.propertyDeclaration("blocked", b.numberType(), b.number("0")),
			b.propertyDeclaration(
				"checkPending",
				b.booleanType(),
				b.factory.FalseLiteral(),
			),
			b.propertyDeclaration(
				"revision",
				b.numberType(),
				b.number("0"),
			),
			b.propertyDeclaration(
				"started",
				b.booleanType(),
				b.factory.FalseLiteral(),
			),
			b.propertyDeclaration(
				"settled",
				b.booleanType(),
				b.factory.TrueLiteral(),
			),
			b.propertyDeclaration(
				"finish",
				b.finishStorageType(),
				b.id("undefined"),
			),
			b.checkMethod(),
			b.settleMethod(),
			b.failMethod(),
			b.taskCompleteMethod(),
			b.mainCompleteMethod(),
			b.blockMethod(),
			b.spawnMethod(),
			b.runMethod(),
		},
	)
}

func (b builder) checkMethod() tsgo.MethodDeclaration {
	active := b.schedulerProperty("active")
	blocked := b.schedulerProperty("blocked")
	pending := b.schedulerProperty("checkPending")
	revision := b.schedulerProperty("revision")
	settled := b.schedulerProperty("settled")
	check := b.arrow(
		nil,
		b.voidType(),
		b.expression(b.assign(pending, b.factory.FalseLiteral())),
		b.factory.IfStatement(
			settled,
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.factory.IfStatement(
			b.binary(
				b.id("scheduledRevision"),
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
				revision,
			),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.schedulerCall("check")),
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.factory.IfStatement(
			b.logicalAnd(
				b.binary(
					active,
					tsgo.BinaryOperatorGreaterThanToken,
					b.number("0"),
				),
				b.strictEqual(active, blocked),
			),
			b.factory.Block([]tsgo.Statement{b.expression(
				b.schedulerCall(
					"fail",
					b.panicValue(
						"all goroutines are asleep - deadlock!",
					),
				),
			)}, true),
			nil,
		),
	)
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		"check",
		nil,
		b.voidType(),
		b.expression(b.assign(
			revision,
			b.add(revision, b.number("1")),
		)),
		b.factory.IfStatement(
			b.logicalOr(pending, settled),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.expression(b.assign(pending, b.factory.TrueLiteral())),
		b.constStatement(
			"scheduledRevision",
			b.numberType(),
			revision,
		),
		b.expression(b.methodCall(b.promiseResolve(), "then", check)),
	)
}

func (b builder) settleMethod() tsgo.MethodDeclaration {
	settled := b.schedulerProperty("settled")
	finish := b.schedulerProperty("finish")
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		"settle",
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.failureType()),
		},
		b.voidType(),
		b.factory.IfStatement(
			settled,
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.expression(b.assign(settled, b.factory.TrueLiteral())),
		b.expression(b.assign(
			b.schedulerProperty("active"),
			b.number("0"),
		)),
		b.expression(b.assign(
			b.schedulerProperty("blocked"),
			b.number("0"),
		)),
		b.expression(b.assign(
			b.schedulerProperty("checkPending"),
			b.factory.FalseLiteral(),
		)),
		b.expression(b.assign(
			b.schedulerProperty("revision"),
			b.number("0"),
		)),
		b.constStatement("complete", b.finishStorageType(), finish),
		b.expression(b.assign(finish, b.id("undefined"))),
		b.factory.IfStatement(
			b.strictDefined(b.id("complete")),
			b.factory.Block([]tsgo.Statement{b.expression(
				b.call(b.id("complete"), b.id("failure")),
			)}, true),
			nil,
		),
	)
}

func (b builder) failMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		"fail",
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.objectType()),
		},
		b.voidType(),
		b.expression(b.schedulerCall("settle", b.id("failure"))),
	)
}

func (b builder) taskCompleteMethod() tsgo.MethodDeclaration {
	active := b.schedulerProperty("active")
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		"taskComplete",
		nil,
		b.voidType(),
		b.factory.IfStatement(
			b.schedulerProperty("settled"),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.expression(b.assign(active, b.subtract(active, b.number("1")))),
		b.expression(b.schedulerCall("check")),
	)
}

func (b builder) mainCompleteMethod() tsgo.MethodDeclaration {
	active := b.schedulerProperty("active")
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		"mainComplete",
		nil,
		b.voidType(),
		b.factory.IfStatement(
			b.schedulerProperty("settled"),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.expression(b.assign(active, b.subtract(active, b.number("1")))),
		b.expression(b.schedulerCall("settle", b.id("undefined"))),
	)
}

func (b builder) blockMethod() tsgo.MethodDeclaration {
	typeT := b.typeT()
	blocked := b.schedulerProperty("blocked")
	resume := b.arrow(
		[]tsgo.ParameterDeclaration{b.parameter("value", typeT)},
		typeT,
		b.factory.IfStatement(
			b.schedulerProperty("settled"),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.id("value")),
			}, true),
			nil,
		),
		b.expression(b.assign(
			blocked,
			b.subtract(blocked, b.number("1")),
		)),
		b.expression(b.schedulerCall("check")),
		b.returnStatement(b.id("value")),
	)
	reject := b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.objectType()),
		},
		b.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNeverKeyword,
		),
		b.factory.IfStatement(
			b.schedulerProperty("settled"),
			b.factory.Block([]tsgo.Statement{
				b.factory.ThrowStatement(b.id("failure")),
			}, true),
			nil,
		),
		b.expression(b.assign(
			blocked,
			b.subtract(blocked, b.number("1")),
		)),
		b.expression(b.schedulerCall("check")),
		b.factory.ThrowStatement(b.id("failure")),
	)
	settled := b.methodCall(
		b.id("operation"),
		"then",
		resume,
		reject,
	)
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		nil,
		b.id(BlockMember),
		nil,
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("operation", b.promiseType(typeT)),
		},
		b.promiseType(typeT),
		b.factory.Block([]tsgo.Statement{
			b.expression(b.assign(
				blocked,
				b.add(blocked, b.number("1")),
			)),
			b.constStatement(
				"settledOperation",
				b.promiseType(typeT),
				settled,
			),
			b.expression(b.schedulerCall("check")),
			b.returnStatement(b.id("settledOperation")),
		}, true),
	)
}

func (b builder) spawnMethod() tsgo.MethodDeclaration {
	taskType := b.functionType(nil, b.promiseType(b.voidType()))
	start := b.methodCall(b.promiseResolve(), "then", b.id("task"))
	complete := b.arrow(
		nil,
		b.voidType(),
		b.expression(b.schedulerCall("taskComplete")),
	)
	fail := b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.objectType()),
		},
		b.voidType(),
		b.expression(b.schedulerCall("fail", b.id("failure"))),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		SpawnMember,
		[]tsgo.ParameterDeclaration{b.parameter("task", taskType)},
		b.voidType(),
		b.factory.IfStatement(
			b.schedulerProperty("settled"),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.expression(b.assign(
			b.schedulerProperty("active"),
			b.add(b.schedulerProperty("active"), b.number("1")),
		)),
		b.expression(b.methodCall(start, "then", complete, fail)),
	)
}

func (b builder) runMethod() tsgo.MethodDeclaration {
	mainType := b.functionType(nil, b.promiseType(b.voidType()))
	executor := b.runExecutor()
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		RunMember,
		[]tsgo.ParameterDeclaration{b.parameter("main", mainType)},
		b.promiseType(b.voidType()),
		b.factory.IfStatement(
			b.schedulerProperty("started"),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.methodCall(
					b.id("Promise"),
					"reject",
					b.panicValue("scheduler already running"),
				)),
			}, true),
			nil,
		),
		b.expression(b.assign(
			b.schedulerProperty("started"),
			b.factory.TrueLiteral(),
		)),
		b.expression(b.assign(
			b.schedulerProperty("settled"),
			b.factory.FalseLiteral(),
		)),
		b.expression(b.assign(
			b.schedulerProperty("checkPending"),
			b.factory.FalseLiteral(),
		)),
		b.expression(b.assign(
			b.schedulerProperty("active"),
			b.number("1"),
		)),
		b.expression(b.assign(
			b.schedulerProperty("blocked"),
			b.number("0"),
		)),
		b.returnStatement(b.factory.NewExpression(
			b.id("Promise"),
			[]tsgo.TypeNode{b.voidType()},
			[]tsgo.Expression{executor},
		)),
	)
}

func (b builder) runExecutor() tsgo.ArrowFunction {
	finish := b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.failureType()),
		},
		b.voidType(),
		b.factory.IfStatement(
			b.strictEqual(b.id("failure"), b.id("undefined")),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.call(b.id("resolve"))),
			}, true),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.call(b.id("reject"), b.id("failure"))),
			}, true),
		),
	)
	mainComplete := b.arrow(
		nil,
		b.voidType(),
		b.expression(b.schedulerCall("mainComplete")),
	)
	fail := b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.objectType()),
		},
		b.voidType(),
		b.expression(b.schedulerCall("fail", b.id("failure"))),
	)
	start := b.methodCall(b.promiseResolve(), "then", b.id("main"))
	return b.arrow(
		[]tsgo.ParameterDeclaration{
			b.factory.ParameterDeclaration(
				nil,
				nil,
				b.id("resolve"),
				nil,
				nil,
				nil,
			),
			b.factory.ParameterDeclaration(
				nil,
				nil,
				b.id("reject"),
				nil,
				nil,
				nil,
			),
		},
		b.voidType(),
		b.expression(b.assign(
			b.schedulerProperty("finish"),
			finish,
		)),
		b.expression(b.methodCall(start, "then", mainComplete, fail)),
	)
}
