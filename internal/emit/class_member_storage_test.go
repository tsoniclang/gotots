package emit

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestClassMemberContributionReconstructsTheTypeOwnedClass(t *testing.T) {
	program := loadClassMemberAssemblyFixture(t)
	sourcePackage := program.Roots()[0]
	record := sourcePackage.Types().Scope().Lookup("Record").(*types.TypeName)
	named := types.Unalias(record.Type()).(*types.Named)
	method := named.Method(0).Origin()
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	recordBuilder, err := session.builder(session.sites[record])
	if err != nil {
		t.Fatal(err)
	}
	methodBuilder, err := session.builder(session.sites[method])
	if err != nil {
		t.Fatal(err)
	}
	if recordBuilder != methodBuilder ||
		filepath.Base(recordBuilder.outputPath) != "record.ts" {
		t.Fatal("method contribution was not assigned to its class target file")
	}

	if err := session.RequireUse(record, rootUseDemand(record), gostdlib.NoUseSelection()); err != nil {
		t.Fatal(err)
	}
	object, ok := session.scheduler.next()
	if !ok || object != record {
		t.Fatalf("initial object = %v, %t; want Record", object, ok)
	}
	if err := session.emit(object); err != nil {
		t.Fatal(err)
	}
	member := session.factory.MethodDeclaration(
		nil,
		nil,
		session.factory.Identifier("Read"),
		nil,
		nil,
		nil,
		session.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		),
		session.factory.Block(
			[]tsgo.Statement{session.factory.ReturnStatement(
				session.factory.NumericLiteral("1", tsgo.TokenFlagsNone),
			)},
			true,
		),
	)
	session.commitClassMemberContribution(
		method,
		&classMemberContribution{
			owner:   record,
			method:  method,
			members: []tsgo.ClassElement{member},
		},
	)
	requirement, err := api.NewClassMethodRequirement(record, method)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.scheduleDeclarationRequirement(requirement); err != nil {
		t.Fatal(err)
	}
	owner, requirements, removed, ok := session.requirements.nextBatch()
	if !ok {
		t.Fatal("class-member attachment requirement was not scheduled")
	}
	if err := session.applyDeclarationRequirements(
		owner,
		requirements,
		removed,
	); err != nil {
		t.Fatal(err)
	}

	declaration := declarationForObject(t, session, record)
	class := targetClass(t, declaration.statements, "Record")
	count := 0
	for _, candidate := range class.Members() {
		target, ok := candidate.(tsgo.MethodDeclaration)
		if ok && target.Name().(tsgo.Identifier).Text() == "Read" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("Record.Read member count = %d, want one", count)
	}
}

func loadClassMemberAssemblyFixture(t *testing.T) *load.Program {
	t.Helper()
	directory := t.TempDir()
	for name, contents := range map[string]string{
		"go.mod": "module example.com/classmember\n\ngo 1.26.4\n",
		"record.go": `package classmember

type Record struct {
	Value int32
}
`,
		"method.go": `package classmember

func (value Record) Read() int32 {
	return value.Value
}
`,
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name),
			[]byte(contents),
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
	return program
}

func TestGenericRepresentationChangeReconstructsPackageStorage(
	t *testing.T,
) {
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/genericstorage\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(`package genericstorage

type Box[T any] struct {
	Value T
	Output *T
}

var Global Box[int32]

func Result() int32 { return Global.Value }
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
	scope := program.Roots()[0].Types().Scope()
	result, err := NewRoot(scope.Lookup("Result"))
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.requireRoot(result); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)

	global := scope.Lookup("Global").(*types.Var)
	box := scope.Lookup("Box").(*types.TypeName)
	boxDeclaration := declarationForObject(t, session, box)
	boxTypeParameters := 0
	storageProjectionCount := 0
	for _, statement := range boxDeclaration.statements {
		class, ok := statement.(tsgo.ClassDeclaration)
		if !ok {
			continue
		}
		boxTypeParameters = len(class.TypeParameters())
		for _, member := range class.Members() {
			method, ok := member.(tsgo.MethodDeclaration)
			if !ok {
				continue
			}
			name := method.Name().(tsgo.Identifier).Text()
			if name == api.StructStorageOfMember {
				storageProjectionCount++
			}
			if strings.HasPrefix(name, "$read$") ||
				strings.HasPrefix(name, "$write$") {
				t.Fatalf(
					"generic storage class retains per-field accessor %q",
					name,
				)
			}
		}
	}
	if storageProjectionCount != 1 {
		t.Fatalf(
			"generic storage projections = %d, want one whole-storage projection",
			storageProjectionCount,
		)
	}
	boxStaticRevision := session.artifacts.FacetRevision(
		api.MustSourceArtifactOwner(box),
		api.ArtifactFacetStaticSurface,
	)
	builder := session.packageBuilders[program.Roots()[0]]
	index, ok := builder.storageByObject[global]
	if !ok {
		t.Fatal("generic package variable has no storage artifact")
	}
	storage := builder.storage[index]
	reference, ok := storage.field.Type().(tsgo.TypeReferenceNode)
	if !ok {
		t.Fatalf(
			"generic package storage type = %T, want TypeReferenceNode",
			storage.field.Type(),
		)
	}
	name, ok := reference.TypeName().(tsgo.Identifier)
	if !ok || !strings.HasSuffix(name.Text(), api.StructStorageTypeSuffix) {
		t.Fatalf(
			"generic package storage type name = %T/%v, want *%s",
			reference.TypeName(),
			reference.TypeName(),
			api.StructStorageTypeSuffix,
		)
	}
	if got := len(reference.TypeArguments()); got != 1 ||
		got != boxTypeParameters {
		t.Fatalf(
			"generic package storage source arguments = %d with %d reconstructions; provider parameters = %d at static revision %d, want exact source arity",
			got,
			storage.reconstructions,
			boxTypeParameters,
			boxStaticRevision,
		)
	}
	dependencies, ok := session.artifacts.Dependencies(storage.owner)
	if !ok {
		t.Fatal("generic package storage has no published dependency record")
	}
	staticDependency := false
	for _, dependency := range dependencies {
		if dependency.Provider() == api.MustSourceArtifactOwner(box) &&
			dependency.Facet() == api.ArtifactFacetStaticSurface {
			staticDependency = true
			break
		}
	}
	if boxStaticRevision <= 1 || !staticDependency ||
		storage.reconstructions != 1 {
		t.Fatalf(
			"generic package storage reconstructions = %d at provider static revision %d with static dependency %t, want one batched reconstruction",
			storage.reconstructions,
			boxStaticRevision,
			staticDependency,
		)
	}
}

func targetClass(
	t *testing.T,
	statements []tsgo.Statement,
	name string,
) tsgo.ClassDeclaration {
	t.Helper()
	for _, statement := range statements {
		class, ok := statement.(tsgo.ClassDeclaration)
		if ok && class.Name().Text() == name {
			return class
		}
	}
	t.Fatalf("target class %s is absent", name)
	return nil
}
