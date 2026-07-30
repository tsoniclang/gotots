package emit

import (
	"context"
	"errors"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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

func pair() (int32, int32) { return 1, 2 }

var PairLeft, PairRight = pair()

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
	initializer := packageInitializerForVariable(
		t,
		program.Roots()[0],
		variable,
	)
	owner := api.MustPackageInitializerArtifactOwner(
		program.Roots()[0].Types(),
		initializer,
	)
	plain := program.Roots()[0].Types().Scope().
		Lookup("Plain").(*types.Var)
	for _, candidate := range program.Roots()[0].TypesInfo().InitOrder {
		for _, target := range candidate.Lhs {
			if target == plain {
				t.Fatal("uninitialized package variable entered the initializer order")
			}
		}
	}
	left := program.Roots()[0].Types().Scope().
		Lookup("PairLeft").(*types.Var)
	right := program.Roots()[0].Types().Scope().
		Lookup("PairRight").(*types.Var)
	leftInitializer := packageInitializerForVariable(
		t,
		program.Roots()[0],
		left,
	)
	rightInitializer := packageInitializerForVariable(
		t,
		program.Roots()[0],
		right,
	)
	if leftInitializer != rightInitializer {
		t.Fatal("one multi-target initializer split into variable-owned artifacts")
	}
	builder := session.packageBuilders[program.Roots()[0]]
	index, ok := builder.initializerByOwner[owner]
	if !ok {
		t.Fatal("package initializer has no exact initializer ownership")
	}
	for _, sourceOwner := range []api.ArtifactOwner{
		sourceArtifactOwner(variable),
		sourceArtifactOwner(left),
		sourceArtifactOwner(right),
	} {
		if _, exists := builder.initializerByOwner[sourceOwner]; exists {
			t.Fatal("package initializer retained a variable-owner surrogate")
		}
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

func TestPackageInitializerLocalIdentityUsesExactInitializer(
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
	for _, artifact := range session.registry.GeneratedArtifacts(
		api.GeneratedArtifactAnonymousStruct,
	) {
		if artifact.Placement() !=
			api.GeneratedArtifactPlacementLexical {
			continue
		}
		sourcePackage, initializer, initializerOwned :=
			artifact.LexicalOwner().PackageInitializer()
		if !initializerOwned ||
			sourcePackage != program.Roots()[0].Types() ||
			initializer == nil {
			t.Fatal("lexical anonymous struct lost its exact initializer owner")
		}
		keys[artifact.ArtifactKey()] = struct{}{}
		owners[artifact.LexicalOwner().Name()] = struct{}{}
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

func TestFunctionValuedPackageStorageReconstructsFromCallableABI(
	t *testing.T,
) {
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/packagecallable\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(`package packagecallable

func receive(values <-chan int32) int32 { return <-values }

var PackageReceiver = receive

func Run() int32 {
	values := make(chan int32, 1)
	values <- 7
	return PackageReceiver(values)
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
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := DefaultOptions()
	options.ConcurrencySemantics = ConcurrencySemanticsCooperative
	session, err := newProgramSession(program, options)
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
		Lookup("PackageReceiver").(*types.Var)
	builder := session.packageBuilders[program.Roots()[0]]
	index, ok := builder.storageByObject[variable]
	if !ok || index >= len(builder.storage) {
		t.Fatal("function-valued package storage has no exact artifact")
	}
	storage := builder.storage[index]
	if storage.owner != api.MustSourceArtifactOwner(variable) ||
		storage.reconstructions != 1 {
		t.Fatalf(
			"package callable owner/reconstructions = %q/%d, want exact owner/1",
			storage.owner.Name(),
			storage.reconstructions,
		)
	}
	union, ok := storage.field.Type().(tsgo.UnionTypeNode)
	if !ok || len(union.Types()) != 2 {
		t.Fatalf(
			"package callable storage type = %T, want callable | undefined",
			storage.field.Type(),
		)
	}
	callable, ok := union.Types()[0].(tsgo.FunctionTypeNode)
	if !ok {
		t.Fatalf(
			"package callable non-nil type = %T, want function",
			union.Types()[0],
		)
	}
	result, ok := callable.Type().(tsgo.TypeReferenceNode)
	if !ok || result.TypeName().(tsgo.Identifier).Text() != "Promise" {
		t.Fatalf(
			"package callable result = %T, want Promise",
			callable.Type(),
		)
	}
	if session.requirements.hasPending() ||
		session.artifacts.HasPending() {
		t.Fatal("package callable ABI fixed point did not converge")
	}
}

func TestPackageAssemblyTracksCommittedExportSurface(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/exportfacet\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(`package exportfacet

type Writer interface {
	Write([]byte) (int, error)
}

type Box struct {
	Value int32
}

func demandBox(values []Box) *Box {
	return &values[0]
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
	sourcePackage := program.Roots()[0]
	writer := sourcePackage.Types().Scope().Lookup("Writer")
	box := sourcePackage.Types().Scope().Lookup("Box")
	demandBox := sourcePackage.Types().Scope().Lookup("demandBox")
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.require(writer); err != nil {
		t.Fatal(err)
	}
	if err := session.require(box); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	builder := session.packageBuilders[sourcePackage]
	if builder == nil {
		t.Fatal("package assembly builder is absent")
	}
	if got := packageExportBindings(builder.exportStatements); !equalStrings(
		got,
		[]string{"Box", "Writer", "Writer$contract", "Writer$is"},
	) {
		t.Fatalf("initial assembly exports = %v", got)
	}
	writerBindings, ok := session.artifacts.ExportedBindings(
		api.MustSourceArtifactOwner(writer),
	)
	if !ok || !equalStrings(
		writerBindings,
		[]string{"Writer", "Writer$contract", "Writer$is"},
	) {
		t.Fatalf("committed Writer export surface = %v, %t", writerBindings, ok)
	}
	initialRevisions := builder.exportRevisions
	initialFacetRevision := session.artifacts.FacetRevision(
		builder.assemblyOwner,
		api.ArtifactFacetImplementation,
	)
	if err := session.require(demandBox); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	want := []string{
		"Box",
		"Box$Storage",
		"Writer",
		"Writer$contract",
		"Writer$is",
	}
	contractBindings, ok := session.artifacts.ExportedBindings(
		api.MustSourceArtifactOwner(box),
	)
	if !ok || !equalStrings(contractBindings, []string{"Box", "Box$Storage"}) {
		t.Fatalf("committed Box export surface = %v, %t", contractBindings, ok)
	}
	if got := packageExportBindings(builder.exportStatements); !equalStrings(
		got,
		want,
	) {
		t.Fatalf("reconstructed assembly exports = %v", got)
	}
	if builder.exportRevisions != initialRevisions+1 {
		t.Fatalf(
			"package export reconstructions = %d, want %d",
			builder.exportRevisions,
			initialRevisions+1,
		)
	}
	if revision := session.artifacts.FacetRevision(
		builder.assemblyOwner,
		api.ArtifactFacetImplementation,
	); revision != initialFacetRevision+1 {
		t.Fatalf("package export facet revision = %d, want %d", revision, initialFacetRevision+1)
	}
	files, err := session.targetFiles()
	if err != nil {
		t.Fatal(err)
	}
	var final []string
	for _, file := range files {
		if file.Kind() == TargetFilePackageAssembly &&
			file.PackageName() == sourcePackage.Name() {
			final = packageExportBindings(file.SourceFile().Statements())
			break
		}
	}
	if !equalStrings(final, want) {
		t.Fatalf("sealed package assembly exports = %v", final)
	}

	mutated := artifactstate.NewGraph(emitordering.CompareArtifactOwners)
	provider := api.MustSourceArtifactOwner(writer)
	if err := mutated.Commit(
		provider,
		artifactstate.NewContract(),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	dependency, err := api.NewArtifactDependency(
		provider,
		api.ArtifactFacetExportSurface,
	)
	if err != nil {
		t.Fatal(err)
	}
	assembly, err := artifactstate.ProjectFacet(
		api.ArtifactFacetImplementation,
		session.factory.Identifier("assembly"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := mutated.Commit(builder.assemblyOwner, assembly, []api.ArtifactDependency{dependency}); err != nil {
		t.Fatal(err)
	}
	var graphError *artifactstate.GraphError
	if err := mutated.VerifyClosure(); !errors.As(err, &graphError) ||
		graphError.Object != builder.assemblyOwner ||
		graphError.Provider != provider ||
		graphError.Facet != api.ArtifactFacetExportSurface {
		t.Fatalf("omitted export-surface mutation error = %#v", err)
	}
}

func packageExportBindings(statements []tsgo.Statement) []string {
	var names []string
	for _, statement := range statements {
		declaration, ok := statement.(tsgo.ExportDeclaration)
		if !ok {
			continue
		}
		exports, ok := declaration.ExportClause().(tsgo.NamedExports)
		if !ok {
			continue
		}
		for _, specifier := range exports.Elements() {
			name, ok := specifier.Name().(tsgo.Identifier)
			if ok {
				names = append(names, name.Text())
			}
		}
	}
	sort.Strings(names)
	return names
}

func packageInitializerForVariable(
	t *testing.T,
	sourcePackage *load.Package,
	variable *types.Var,
) *types.Initializer {
	t.Helper()
	var selected *types.Initializer
	for _, initializer := range sourcePackage.TypesInfo().InitOrder {
		for _, target := range initializer.Lhs {
			if target != variable {
				continue
			}
			if selected != nil {
				t.Fatalf(
					"variable %s belongs to multiple package initializers",
					variable.Name(),
				)
			}
			selected = initializer
		}
	}
	if selected == nil {
		t.Fatalf(
			"variable %s has no package initializer",
			variable.Name(),
		)
	}
	return selected
}
