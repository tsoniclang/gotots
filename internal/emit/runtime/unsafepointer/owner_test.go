package unsafepointer

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildPrintsIndexedAddressAndProportionalRegionOperations(t *testing.T) {
	client, err := tsgo.StartClient(
		filepath.Join("..", "..", "..", ".."),
		t.TempDir(),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	printed, err := client.PrintNode(
		Build(
			tsgo.NewFactory(),
			"GoUnsafePointer",
			"GoUnsafeCodec",
			"GoPanic",
			"GoPointer",
			"goPointerUnsafeMemory",
			"GoDenseIndex",
		),
		tsgo.PrintOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"private static allocationAtAddress(value: number): GoUnsafePointer | undefined",
		"static fromRelative(value: GoUnsafePointer | undefined, address: number | bigint, zero: number | bigint): GoUnsafePointer | undefined",
		"while (low <= high)",
		"const first: number = Math.floor(offset / codec.size);",
		"const last: number = Math.ceil((offset + length) / codec.size);",
		"new Uint8Array((last - first) * codec.size)",
		"for (let index = first; index < last; index++)",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("unsafe-pointer runtime lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"for (const allocation of GoUnsafePointer.allocations)",
		"index < backing.length",
		"new Uint8Array(totalLength)",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("unsafe-pointer runtime retains %q:\n%s", forbidden, printed)
		}
	}
}

func TestBuildCreatesTypedUnsafeMemoryOwner(t *testing.T) {
	class := Build(
		tsgo.NewFactory(),
		"GoUnsafePointer",
		"GoUnsafeCodec",
		"GoPanic",
		"GoPointer",
		"goPointerUnsafeMemory",
		"GoDenseIndex",
	)
	if class.Name().Text() != "GoUnsafePointer" ||
		len(class.Modifiers()) != 1 ||
		class.Modifiers()[0].Kind() != tsgo.SyntaxKindExportKeyword ||
		len(class.Members()) != 17 {
		t.Fatalf("unsafe-pointer class has unexpected shape")
	}
	constructor, ok := class.Members()[4].(tsgo.ConstructorDeclaration)
	if !ok ||
		len(constructor.Modifiers()) != 1 ||
		constructor.Modifiers()[0].Kind() != tsgo.SyntaxKindPrivateKeyword ||
		len(constructor.Parameters()) != 8 ||
		constructor.Body() == nil {
		t.Fatal("unsafe-pointer owner lacks its private memory constructor")
	}
	for _, name := range []string{FromName, ToName} {
		method := concreteMethod(t, class, name)
		if len(method.TypeParameters()) != 2 ||
			len(method.Parameters()) != 2 ||
			method.Body() == nil {
			t.Fatalf("unsafe-pointer conversion method %q is invalid", name)
		}
	}
	if method := concreteMethod(t, class, FromIntegerName); len(method.Parameters()) != 2 {
		t.Fatal("unsafe-pointer integer decoder has an unexpected signature")
	}
	if method := concreteMethod(t, class, allocationAtAddressName); len(method.Parameters()) != 1 {
		t.Fatal("unsafe-pointer allocation index has an unexpected signature")
	}
	if method := concreteMethod(t, class, FromRelativeName); len(method.Parameters()) != 3 {
		t.Fatal("unsafe-pointer relative decoder has an unexpected signature")
	}
	if method := concreteMethod(t, class, ToIntegerName); len(method.Parameters()) != 2 {
		t.Fatal("unsafe-pointer integer encoder has an unexpected signature")
	}
}

func concreteMethod(
	t *testing.T,
	class tsgo.ClassDeclaration,
	name string,
) tsgo.MethodDeclaration {
	t.Helper()
	for _, member := range class.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok || method.Body() == nil {
			continue
		}
		identifier, ok := method.Name().(tsgo.Identifier)
		if ok && identifier.Text() == name {
			return method
		}
	}
	t.Fatalf("unsafe-pointer owner lacks concrete method %q", name)
	return nil
}
