package emit

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestPackageInitializerUsesDeclarationRequirementFixedPoint(
	t *testing.T,
) {
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/initializer\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(`package initializer

var PackageValue = func() int32 {
	type Local int32
	item := struct{ Value Local }{Value: Local(3)}
	copy := item
	if copy == item {
		return int32(copy.Value)
	}
	return 0
}()

var Plain int32

func Result() int32 { return PackageValue }
`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range roots {
		if err := session.requireRoot(root); err != nil {
			t.Fatal(err)
		}
	}
	drainProgramSession(t, session)

	variable := program.Roots()[0].Types().Scope().
		Lookup("PackageValue").(*types.Var)
	owner := sourceArtifactOwner(variable)
	plain := program.Roots()[0].Types().Scope().
		Lookup("Plain").(*types.Var)
	if _, initializerOwner := session.packageInitializerOwners[plain]; initializerOwner {
		t.Fatal("uninitialized package variable became an initializer artifact owner")
	}
	if _, initializerOwner := session.packageInitializerOwners[variable]; !initializerOwner {
		t.Fatal("package initializer lost its exact source owner")
	}
	builder := session.packageBuilders[program.Roots()[0]]
	index, ok := builder.initializerByOwner[owner]
	if !ok {
		t.Fatal("package initializer has no source-artifact ownership")
	}
	artifact := builder.initialization[index]
	if artifact.reconstructions != 1 {
		t.Fatalf(
			"package initializer reconstructions = %d, want one fixed-point revision",
			artifact.reconstructions,
		)
	}
	if requirements := session.requirements.appliedFor(owner); len(requirements) != 3 {
		t.Fatalf(
			"package initializer anonymous requirements = %d, want definition/copy/equal",
			len(requirements),
		)
	}
	if session.requirements.hasPending() ||
		session.artifacts.HasPending() {
		t.Fatal("package initializer did not converge in the existing fixed point")
	}
	files, err := session.targetFiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.OutputPath() == "support/anonymous-structs.ts" {
			t.Fatal("lexical package initializer created a parallel support artifact")
		}
	}
}

func TestPackageInitializerLocalIdentityUsesSourceArtifactPath(
	t *testing.T,
) {
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/files\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	for fileName, source := range map[string]string{
		"a.go": `package files

var First = func() int32 {
	type Local int32
	item := struct{ Value Local }{Value: Local(1)}
	return int32(item.Value)
}()

func FirstResult() int32 { return First }
`,
		"b.go": `package files

var Other = func() int32 {
	type Local int32
	item := struct{ Value Local }{Value: Local(2)}
	return int32(item.Value)
}()

func OtherResult() int32 { return Other }
`,
	} {
		if err := os.WriteFile(
			filepath.Join(directory, fileName),
			[]byte(source),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range roots {
		if err := session.requireRoot(root); err != nil {
			t.Fatal(err)
		}
	}
	drainProgramSession(t, session)
	keys := make(map[string]struct{})
	owners := make(map[string]struct{})
	for key, binding := range session.registry.anonymousStructs {
		if binding.owner.Placement() !=
			api.GeneratedArtifactPlacementLexical {
			continue
		}
		sourceOwner, sourceOwned := binding.owner.LexicalOwner().Source()
		if !sourceOwned {
			t.Fatal("lexical anonymous struct lost its source owner")
		}
		keys[key] = struct{}{}
		owners[sourceOwner.Name()] = struct{}{}
	}
	if len(keys) != 2 ||
		len(owners) != 2 {
		t.Fatalf(
			"cross-file lexical identities = %d keys / %d owners, want two exact artifacts",
			len(keys),
			len(owners),
		)
	}
}
