package panicruntime

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildCreatesOneNonErasedPanicCarrierAndRecoveryAuthority(t *testing.T) {
	factory := tsgo.NewFactory()
	carrierTarget, err := Build(
		factory,
		api.RuntimePanic,
		"GoPanic",
		"GoInterfaceValue",
		"GoRuntimePanicValue",
		"GoRecovery",
		"goDeferPop",
		"GoErrorMethodToken",
		"GoRuntimeErrorMethodToken",
	)
	if err != nil {
		t.Fatal(err)
	}
	carrier := carrierTarget.(tsgo.ClassDeclaration)
	if className(carrier) != "GoPanic" ||
		len(carrier.TypeParameters()) != 0 ||
		len(carrier.Members()) != 6 {
		t.Fatalf(
			"panic class = %q, type parameters %d, members %d",
			className(carrier),
			len(carrier.TypeParameters()),
			len(carrier.Members()),
		)
	}
	for _, index := range []int{2, 3, 4} {
		method := carrier.Members()[index].(tsgo.MethodDeclaration)
		body := method.Body().(tsgo.Block).Statements()
		if len(body) != 1 || body[0].Kind() != tsgo.SyntaxKindThrowStatement {
			t.Fatalf("panic method %d does not throw exactly once", index)
		}
	}
	recoveryTarget, err := Build(
		factory,
		api.RuntimeRecovery,
		"GoPanic",
		"GoInterfaceValue",
		"GoRuntimePanicValue",
		"GoRecovery",
		"goDeferPop",
		"GoErrorMethodToken",
		"GoRuntimeErrorMethodToken",
	)
	if err != nil {
		t.Fatal(err)
	}
	recovery := recoveryTarget.(tsgo.ClassDeclaration)
	if className(recovery) != "GoRecovery" || len(recovery.Members()) != 4 {
		t.Fatalf(
			"recovery class = %q with %d members",
			className(recovery),
			len(recovery.Members()),
		)
	}
}

func TestBuildCreatesCheckedGenericDeferPop(t *testing.T) {
	target, err := Build(
		tsgo.NewFactory(),
		api.RuntimeDeferPop,
		"GoPanic",
		"GoInterfaceValue",
		"GoRuntimePanicValue",
		"GoRecovery",
		"goDeferPop",
		"GoErrorMethodToken",
		"GoRuntimeErrorMethodToken",
	)
	if err != nil {
		t.Fatal(err)
	}
	function, ok := target.(tsgo.FunctionDeclaration)
	if !ok ||
		function.Name().Text() != "goDeferPop" ||
		len(function.TypeParameters()) != 1 ||
		len(function.Parameters()) != 1 {
		t.Fatalf("defer pop target = %T with unexpected signature", target)
	}
	statements := function.Body().(tsgo.Block).Statements()
	if len(statements) != 3 ||
		statements[0].Kind() != tsgo.SyntaxKindVariableStatement ||
		statements[1].Kind() != tsgo.SyntaxKindIfStatement ||
		statements[2].Kind() != tsgo.SyntaxKindReturnStatement {
		t.Fatalf("defer pop statements = %#v", statements)
	}
}

func TestCallsUseClosedSourceAndRuntimeRaiseMembers(t *testing.T) {
	factory := tsgo.NewFactory()
	for _, test := range []struct {
		call tsgo.CallExpression
		want string
	}{
		{
			call: CreateRuntime(
				factory,
				"GoPanic",
				factory.StringLiteral("boom", tsgo.TokenFlagsNone),
			),
			want: CreateRuntimeName,
		},
		{
			call: Call(
				factory,
				"GoPanic",
				factory.StringLiteral("boom", tsgo.TokenFlagsNone),
			),
			want: RaiseRuntimeName,
		},
		{
			call: CallValue(
				factory,
				"GoPanic",
				factory.Identifier("payload"),
			),
			want: RaiseName,
		},
	} {
		member := test.call.Expression().(tsgo.PropertyAccessExpression)
		if member.Expression().(tsgo.Identifier).Text() != "GoPanic" ||
			member.Name().(tsgo.Identifier).Text() != test.want {
			t.Fatalf("panic call bypasses %s", test.want)
		}
	}
}

func className(class tsgo.ClassDeclaration) string {
	if class.Name() == nil {
		return ""
	}
	return class.Name().Text()
}
