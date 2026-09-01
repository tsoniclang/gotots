package sourcefact

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestExactDeclarationTargetRejectsMissingAndDuplicateDeclarations(t *testing.T) {
	factory := tsgo.NewFactory()
	function := factory.FunctionDeclaration(
		nil,
		nil,
		factory.Identifier("Build"),
		nil,
		nil,
		nil,
		factory.Block(nil, true),
	)
	if _, err := exactDeclarationTarget(
		factory,
		[]string{"Missing"},
		artifactTargetValue,
		[]tsgo.Statement{function},
	); err == nil {
		t.Fatal("missing source-fact declaration target was admitted")
	}
	if _, err := exactDeclarationTarget(
		factory,
		[]string{"Build"},
		artifactTargetValue,
		[]tsgo.Statement{function, function},
	); err == nil {
		t.Fatal("duplicate source-fact declaration target was admitted")
	}
}

func TestConstructedArtifactTargetDistinguishesClassAndFactoryValue(t *testing.T) {
	factory := tsgo.NewFactory()
	class := factory.ClassDeclaration(nil, factory.Identifier("Record"), nil, nil, nil)
	classTarget, err := exactDeclarationTarget(
		factory,
		[]string{"Record"},
		artifactTargetConstructedType,
		[]tsgo.Statement{class},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := classTarget.(tsgo.TypeReferenceNode); !ok {
		t.Fatalf("class constructed target = %T, want instance type reference", classTarget)
	}
	declaration := factory.VariableDeclaration(factory.Identifier("RecordFactory"), nil, nil, nil)
	variable := factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{declaration},
			tsgo.NodeFlagsConst,
		),
	)
	factoryTarget, err := exactDeclarationTarget(
		factory,
		[]string{"RecordFactory"},
		artifactTargetConstructedType,
		[]tsgo.Statement{variable},
	)
	if err != nil {
		t.Fatal(err)
	}
	reference, ok := factoryTarget.(tsgo.TypeReferenceNode)
	if !ok {
		t.Fatalf("factory constructed target = %T, want InstanceType reference", factoryTarget)
	}
	identifier, ok := reference.TypeName().(tsgo.Identifier)
	if !ok || identifier.Text() != "InstanceType" {
		t.Fatal("factory constructed target does not retain its instance contract")
	}
}

func TestConcreteMethodTargetUsesActualStaticnessAndKernelIdentity(t *testing.T) {
	factory := tsgo.NewFactory()
	instance := testMethod(factory, nil, "Read")
	staticKernel := testMethod(
		factory,
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		"Read"+api.GenericKernelSuffix,
	)
	for name, testCase := range map[string]struct {
		member tsgo.ClassElement
		query  bool
		name   string
	}{
		"instance":      {instance, false, "Read"},
		"static kernel": {staticKernel, true, "Read" + api.GenericKernelSuffix},
	} {
		t.Run(name, func(t *testing.T) {
			class := factory.ClassDeclaration(
				nil,
				factory.Identifier("Record"),
				nil,
				nil,
				[]tsgo.ClassElement{testCase.member},
			)
			target, selected, err := exactConcreteMethodTarget(
				factory,
				"Record",
				0,
				"Read",
				[]tsgo.Statement{class},
			)
			if err != nil {
				t.Fatal(err)
			}
			_, query := target.(tsgo.TypeQueryNode)
			if query != testCase.query || selected != testCase.name {
				t.Fatalf(
					"method target = %T/%q, want query=%t name=%q",
					target,
					selected,
					testCase.query,
					testCase.name,
				)
			}
		})
	}
	class := factory.ClassDeclaration(
		nil,
		factory.Identifier("Record"),
		nil,
		nil,
		[]tsgo.ClassElement{instance, staticKernel},
	)
	if _, _, err := exactConcreteMethodTarget(
		factory,
		"Record",
		0,
		"Read",
		[]tsgo.Statement{class},
	); err == nil {
		t.Fatal("ambiguous concrete method fact target was admitted")
	}
}

func testMethod(
	factory tsgo.Factory,
	modifiers []tsgo.ModifierLike,
	name string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		modifiers,
		nil,
		factory.Identifier(name),
		nil,
		nil,
		nil,
		nil,
		factory.Block(nil, true),
	)
}
