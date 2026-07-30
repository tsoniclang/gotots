package emit

import (
	"context"
	"errors"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"testing"

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

	if err := session.require(record); err != nil {
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
	requirements, ok := session.requirements.nextBatch()
	if !ok {
		t.Fatal("class-member attachment requirement was not scheduled")
	}
	if err := session.applyDeclarationRequirements(requirements); err != nil {
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

func TestAddressableStorageReconstructsOnlyOwningBodiesIncludingInit(
	t *testing.T,
) {
	program := loadAddressableArtifactFixture(t)
	sourcePackage := program.Roots()[0]
	scope := sourcePackage.Types().Scope()
	addressed := scope.Lookup("Addressed").(*types.Func)
	caller := scope.Lookup("Caller").(*types.Func)
	literalValue := scope.Lookup("literalValue").(*types.Var)
	literalInitializer := packageInitializerForVariable(
		t,
		sourcePackage,
		literalValue,
	)
	literalOwner := api.MustPackageInitializerArtifactOwner(
		sourcePackage.Types(),
		literalInitializer,
	)
	var initializer *types.Func
	for _, declaration := range sourcePackage.Files()[0].Syntax().Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || !isPackageInitDeclaration(function) {
			continue
		}
		initializer = sourcePackage.TypesInfo().Defs[function.Name].(*types.Func)
	}
	if initializer == nil {
		t.Fatal("package init identity is absent")
	}

	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.require(caller); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)

	for object, want := range map[types.Object]uint64{
		addressed:   1,
		initializer: 1,
		caller:      0,
	} {
		declaration := declarationForObject(t, session, object)
		if declaration.reconstructions != want {
			t.Fatalf(
				"%s reconstructions = %d, want %d",
				object.Name(),
				declaration.reconstructions,
				want,
			)
		}
		if session.artifacts.FacetRevision(
			sourceArtifactOwner(object),
			api.ArtifactFacetCallableSignature,
		) != 1 {
			t.Fatalf(
				"%s callable signature changed during body reconstruction",
				object.Name(),
			)
		}
	}
	if len(session.requirements.appliedFor(sourceArtifactOwner(addressed))) != 1 ||
		len(session.requirements.appliedFor(sourceArtifactOwner(initializer))) != 1 ||
		len(session.requirements.appliedFor(literalOwner)) != 1 ||
		len(session.requirements.appliedFor(sourceArtifactOwner(caller))) != 0 {
		t.Fatal("addressable-storage requirements escaped their exact body owners")
	}
	builder := session.packageBuilders[sourcePackage]
	literalIndex, ok := builder.initializerByOwner[literalOwner]
	if !ok || builder.initialization[literalIndex].reconstructions != 1 {
		t.Fatal("function-literal storage did not reconstruct its package initializer")
	}
	if session.artifacts.HasPending() ||
		session.requirements.hasPending() ||
		session.scheduler.hasPending() {
		t.Fatal("addressable-storage reconstruction did not reach a fixed point")
	}
}

func TestAddressableStorageRejectsForeignAndForgedSameSpellingVariables(
	t *testing.T,
) {
	program := loadAddressableArtifactFixture(t)
	scope := program.Roots()[0].Types().Scope()
	addressed := scope.Lookup("Addressed").(*types.Func)
	caller := scope.Lookup("Caller").(*types.Func)
	addressedSignature := addressed.Type().(*types.Signature)
	callerSignature := caller.Type().(*types.Signature)
	if addressedSignature.Params().At(0).Name() !=
		callerSignature.Params().At(0).Name() {
		t.Fatal("foreign-variable fixture does not share a spelling")
	}
	requirement, err := api.NewAddressableStorageRequirement(
		sourceArtifactOwner(addressed),
		callerSignature.Params().At(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.require(addressed); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	if err := session.scheduleDeclarationRequirement(requirement); err != nil {
		t.Fatal(err)
	}
	requirements, ok := session.requirements.nextBatch()
	if !ok {
		t.Fatal("foreign same-spelling requirement was not scheduled")
	}
	err = session.applyDeclarationRequirements(requirements)
	var invariant *api.InvariantError
	if !errors.As(err, &invariant) ||
		invariant.Reason !=
			"artifact received foreign addressable-storage requirement" {
		t.Fatalf("foreign same-spelling requirement error = %#v", err)
	}

	addressedParameter := addressedSignature.Params().At(0)
	forged := types.NewVar(
		addressedParameter.Pos(),
		addressedParameter.Pkg(),
		addressedParameter.Name(),
		addressedParameter.Type(),
	)
	requirement, err = api.NewAddressableStorageRequirement(
		sourceArtifactOwner(addressed),
		forged,
	)
	if err != nil {
		t.Fatal(err)
	}
	session, err = newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.require(addressed); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	if err := session.scheduleDeclarationRequirement(requirement); err != nil {
		t.Fatal(err)
	}
	requirements, ok = session.requirements.nextBatch()
	if !ok {
		t.Fatal("forged exact-position requirement was not scheduled")
	}
	err = session.applyDeclarationRequirements(requirements)
	var nameError *api.NameError
	if !errors.As(err, &nameError) ||
		nameError.Reason !=
			"declaration object was not indexed from its Go scope" {
		t.Fatalf("forged exact-position requirement error = %#v", err)
	}
}

func loadAddressableArtifactFixture(t *testing.T) *load.Program {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/addressable-artifact\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(`package addressableartifact

var initialized int32

var literalValue = func() int32 {
	value := int32(3)
	pointer := &value
	*pointer++
	return value
}()

func init() {
	value := int32(2)
	pointer := &value
	*pointer++
	initialized = value
}

func Addressed(value int32) int32 {
	pointer := &value
	*pointer++
	return value
}

func Caller(value int32) int32 {
	return Addressed(value) + initialized + literalValue
}
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
	return program
}
