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
		"GoErrorMethodToken",
		"GoRuntimeErrorMethodToken",
	)
	if err != nil {
		t.Fatal(err)
	}
	carrier := carrierTarget.(tsgo.ClassDeclaration)
	if className(carrier) != "GoPanic" ||
		len(carrier.TypeParameters()) != 0 ||
		len(carrier.Members()) != 4 {
		t.Fatalf(
			"panic class = %q, type parameters %d, members %d",
			className(carrier),
			len(carrier.TypeParameters()),
			len(carrier.Members()),
		)
	}
	for _, index := range []int{2, 3} {
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
		"GoErrorMethodToken",
		"GoRuntimeErrorMethodToken",
	)
	if err != nil {
		t.Fatal(err)
	}
	recovery := recoveryTarget.(tsgo.ClassDeclaration)
	if className(recovery) != "GoRecovery" || len(recovery.Members()) != 3 {
		t.Fatalf(
			"recovery class = %q with %d members",
			className(recovery),
			len(recovery.Members()),
		)
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
