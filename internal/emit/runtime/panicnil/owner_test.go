package panicnil_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/emit/runtime/panicnil"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestPanicNilAndRuntimeFaultHaveDistinctDynamicIdentities(t *testing.T) {
	factory := tsgo.NewFactory()
	runtimeTarget, err := panicruntime.Build(
		factory,
		api.RuntimePanicValue,
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
	nilTarget, err := panicnil.Build(
		factory,
		api.RuntimePanicNilValue,
		"GoPanicNilError",
		"GoPanicNilValue",
		"GoRuntimePanicValue",
		"GoInterfaceValue",
	)
	if err != nil {
		t.Fatal(err)
	}
	runtimeIdentity := dynamicIdentity(
		t,
		runtimeTarget.(tsgo.ClassDeclaration),
	)
	nilIdentity := dynamicIdentity(
		t,
		nilTarget.(tsgo.ClassDeclaration),
	)
	if runtimeIdentity != "GoRuntimePanicValue" {
		t.Fatalf("runtime-fault identity = %q", runtimeIdentity)
	}
	if nilIdentity != "GoPanicNilError" {
		t.Fatalf("panic-nil identity = %q", nilIdentity)
	}
	if runtimeIdentity == nilIdentity {
		t.Fatal("panic-nil and runtime-fault dynamic identities alias")
	}
}

func dynamicIdentity(
	t *testing.T,
	class tsgo.ClassDeclaration,
) string {
	t.Helper()
	for _, member := range class.Members() {
		property, ok := member.(tsgo.PropertyDeclaration)
		if !ok {
			continue
		}
		name, ok := property.Name().(tsgo.Identifier)
		if !ok || name.Text() != interfacecontract.DynamicTypeMember {
			continue
		}
		value, ok := property.Initializer().(tsgo.Identifier)
		if !ok {
			t.Fatalf(
				"%s initializer = %T",
				interfacecontract.DynamicTypeMember,
				property.Initializer(),
			)
		}
		return value.Text()
	}
	t.Fatalf("class has no %s property", interfacecontract.DynamicTypeMember)
	return ""
}
