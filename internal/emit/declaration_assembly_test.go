package emit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestReachedUsesReconstructAndSealDeclarationAssemblies(t *testing.T) {
	program := loadDeclarationAssemblyFixture(t)
	sourcePackage := program.Roots()[0]
	box := sourcePackage.Types().Scope().Lookup("Box").(*types.TypeName)
	item := sourcePackage.Types().Scope().Lookup("Item").(*types.TypeName)
	use := sourcePackage.Types().Scope().Lookup("Use")
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	if err := session.require(box); err != nil {
		t.Fatal(err)
	}
	object, ok := session.scheduler.next()
	if !ok || object != box {
		t.Fatalf("initial declaration = %v, %t; want Box", object, ok)
	}
	if err := session.emit(object); err != nil {
		t.Fatal(err)
	}
	boxDeclaration := declarationForObject(t, session, box)
	if len(boxDeclaration.statements) != 1 ||
		len(session.requirements.appliedFor(sourceArtifactOwner(box))) != 0 ||
		boxDeclaration.reconstructions != 0 {
		t.Fatalf("initial Box assembly = %#v", boxDeclaration)
	}
	initialClass := boxDeclaration.statements[0]

	if err := session.require(use); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)

	boxDeclaration = declarationForObject(t, session, box)
	itemDeclaration := declarationForObject(t, session, item)
	for name, expected := range map[string]struct {
		declaration     *targetDeclaration
		reconstructions uint64
	}{
		"Box":  {declaration: boxDeclaration, reconstructions: 2},
		"Item": {declaration: itemDeclaration, reconstructions: 1},
	} {
		declaration := expected.declaration
		if len(session.requirements.appliedFor(declaration.owner)) != 3 {
			t.Fatalf(
				"%s requirements = %d, want zero/copy/equal",
				name,
				len(session.requirements.appliedFor(declaration.owner)),
			)
		}
		if declaration.reconstructions != expected.reconstructions {
			t.Fatalf(
				"%s reconstructions = %d, want %d",
				name,
				declaration.reconstructions,
				expected.reconstructions,
			)
		}
		if len(declaration.statements) != 1 {
			t.Fatalf("%s assembly statements = %d, want one reconstructed class", name, len(declaration.statements))
		}
	}
	if initialClass == boxDeclaration.statements[0] {
		t.Fatal("Box class node was not reconstructed after late requirements")
	}

	files, err := session.targetFiles()
	if err != nil {
		t.Fatal(err)
	}
	assertOneFinalDeclarationAssembly(t, files, "Box")
	assertOneFinalDeclarationAssembly(t, files, "Item")

	requirement, err := api.NewNamedStructOperationRequirement(
		box,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = session.scheduleDeclarationRequirement(requirement)
	var scheduleError *ScheduleError
	if !errors.As(err, &scheduleError) {
		t.Fatalf("post-seal requirement error = %#v, want ScheduleError", err)
	}
	importRequest, err := api.NewImportRequest(
		session.factory,
		api.ImportPhaseValue,
		"./dependency.js",
		"Value",
		"Value",
	)
	if err != nil {
		t.Fatal(err)
	}
	err = session.applyRootRequests(
		newPlacementOwner(),
		[]api.RootRequest{importRequest},
	)
	if !errors.As(err, &scheduleError) {
		t.Fatalf("post-seal import error = %#v, want ScheduleError", err)
	}
}

func TestObservableChangesReconstructOnlySubscribedDeclarations(t *testing.T) {
	program := loadDeclarationAssemblyFixture(t)
	scope := program.Roots()[0].Types().Scope()
	item := scope.Lookup("Item")
	box := scope.Lookup("Box")
	trigger := scope.Lookup("Trigger")
	caller := scope.Lookup("Caller")
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.require(caller); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)

	expected := map[types.Object]uint64{
		item:    1,
		box:     2,
		trigger: 1,
		caller:  0,
	}
	for object, reconstructions := range expected {
		declaration := declarationForObject(t, session, object)
		if declaration.reconstructions != reconstructions {
			t.Fatalf(
				"%s reconstructions = %d, want %d",
				object.Name(),
				declaration.reconstructions,
				reconstructions,
			)
		}
	}
	if session.artifacts.FacetRevision(
		sourceArtifactOwner(box),
		api.ArtifactFacetStaticSurface,
	) != 2 {
		t.Fatal("Box static surface did not publish its late operations")
	}
	if session.artifacts.FacetRevision(
		sourceArtifactOwner(trigger),
		api.ArtifactFacetCallableSignature,
	) != 1 {
		t.Fatal("Trigger body reconstruction changed its callable signature")
	}
	if session.artifacts.HasPending() {
		t.Fatal("observable propagation did not reach a fixed point")
	}
}

func TestDeclarationRequirementRejectsSameSpellingWithoutExactOwner(
	t *testing.T,
) {
	program := loadDeclarationAssemblyFixture(t)
	sourcePackage := program.Roots()[0]
	box := sourcePackage.Types().Scope().Lookup("Box").(*types.TypeName)
	forged := types.NewTypeName(box.Pos(), box.Pkg(), box.Name(), box.Type())
	requirement, err := api.NewNamedStructOperationRequirement(
		forged,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	err = session.scheduleDeclarationRequirement(requirement)
	var scheduleError *ScheduleError
	if !errors.As(err, &scheduleError) {
		t.Fatalf("forged-owner error = %#v, want ScheduleError", err)
	}
}

func TestDeclarationAssembliesAreByteStableAcrossRootOrder(t *testing.T) {
	program := loadDeclarationAssemblyFixture(t)
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	first, err := Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(roots)
	second, err := Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	firstFiles := first.Files()
	secondFiles := second.Files()
	if len(firstFiles) != len(secondFiles) {
		t.Fatalf("target file counts = %d and %d", len(firstFiles), len(secondFiles))
	}
	for index := range firstFiles {
		if firstFiles[index].OutputPath() != secondFiles[index].OutputPath() {
			t.Fatalf("target file %d paths differ", index)
		}
		firstBytes, err := tsgo.EncodeSourceFile(firstFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		secondBytes, err := tsgo.EncodeSourceFile(secondFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(firstBytes, secondBytes) {
			t.Fatalf(
				"target file %s changed with root order",
				firstFiles[index].OutputPath(),
			)
		}
	}
}

func TestDeclarationAssembliesCannotSealWithPendingWork(t *testing.T) {
	program := loadDeclarationAssemblyFixture(t)
	box := program.Roots()[0].Types().Scope().Lookup("Box").(*types.TypeName)
	requirement, err := api.NewNamedStructOperationRequirement(
		box,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.require(box); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	if err := session.scheduleDeclarationRequirement(requirement); err != nil {
		t.Fatal(err)
	}
	if session.scheduler.hasPending() ||
		session.packageInitializations.hasPending() ||
		!session.requirements.hasPending() {
		t.Fatal("pending-seal fixture is not isolated to declaration requirements")
	}
	_, err = session.targetFiles()
	var scheduleError *ScheduleError
	if !errors.As(err, &scheduleError) {
		t.Fatalf("pending-seal error = %#v, want ScheduleError", err)
	}
	if session.sealed {
		t.Fatal("failed pending-work seal closed the session")
	}
}

func TestDeclarationAssemblyCostDoesNotGrowPerUseSite(t *testing.T) {
	useCounts := []int{8, 16, 32}
	measurements := make([]declarationAssemblyMeasurement, len(useCounts))
	for index, useCount := range useCounts {
		measurements[index] = measureDeclarationAssembly(t, useCount)
		t.Logf(
			"use sites=%d requirements=%d reconstructions=%d definition roots=%d assembly bytes=%d source-file bytes=%d",
			useCount,
			measurements[index].requirements,
			measurements[index].reconstructions,
			measurements[index].definitionRoots,
			measurements[index].assemblyBytes,
			measurements[index].fileBytes,
		)
		if measurements[index].requirements != 3 ||
			measurements[index].reconstructions != 1 ||
			measurements[index].definitionRoots != 1 {
			t.Fatalf(
				"use sites %d metrics = %#v, want 3 requirements, 1 reconstruction, 1 definition root",
				useCount,
				measurements[index],
			)
		}
		if index != 0 &&
			measurements[index].assemblyBytes != measurements[0].assemblyBytes {
			t.Fatalf(
				"assembly bytes grew with use sites: %d then %d",
				measurements[0].assemblyBytes,
				measurements[index].assemblyBytes,
			)
		}
	}
	firstDelta := measurements[1].fileBytes - measurements[0].fileBytes
	secondDelta := measurements[2].fileBytes - measurements[1].fileBytes
	if firstDelta <= 0 ||
		secondDelta*10 < firstDelta*17 ||
		secondDelta*10 > firstDelta*23 {
		t.Fatalf(
			"file bytes = %d/%d/%d; doubling-use deltas %d/%d are not linear",
			measurements[0].fileBytes,
			measurements[1].fileBytes,
			measurements[2].fileBytes,
			firstDelta,
			secondDelta,
		)
	}
}

type declarationAssemblyMeasurement struct {
	requirements    int
	reconstructions uint64
	definitionRoots int
	assemblyBytes   int
	fileBytes       int
}

func measureDeclarationAssembly(
	t *testing.T,
	useCount int,
) declarationAssemblyMeasurement {
	t.Helper()
	program := loadDeclarationAssemblyScalingFixture(t, useCount)
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range roots {
		if err := session.require(root.object); err != nil {
			t.Fatal(err)
		}
	}
	drainProgramSession(t, session)
	record := program.Roots()[0].Types().Scope().
		Lookup("Record").(*types.TypeName)
	declaration := declarationForObject(t, session, record)
	measurement := declarationAssemblyMeasurement{
		requirements: len(session.requirements.appliedFor(
			sourceArtifactOwner(record),
		)),
		reconstructions: declaration.reconstructions,
		definitionRoots: len(declaration.statements),
	}
	for _, statement := range declaration.statements {
		encoded, err := tsgo.EncodeNode(statement)
		if err != nil {
			t.Fatal(err)
		}
		measurement.assemblyBytes += len(encoded)
	}
	files, err := session.targetFiles()
	if err != nil {
		t.Fatal(err)
	}
	assertOneFinalDeclarationAssembly(t, files, "Record")
	for _, file := range files {
		if file.Kind() != TargetFileSource {
			continue
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		measurement.fileBytes += len(encoded)
	}
	return measurement
}

func loadDeclarationAssemblyScalingFixture(
	t *testing.T,
	useCount int,
) *load.Program {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/assemblyscale\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	var source strings.Builder
	source.WriteString("package assemblyscale\n\ntype Record struct { Value int32 }\n")
	for index := range useCount {
		fmt.Fprintf(&source, `
func Use%d(left, right Record) Record {
	var result Record
	result = left
	if left == right {
		return result
	}
	return right
}
`, index)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(source.String()),
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

func loadDeclarationAssemblyFixture(t *testing.T) *load.Program {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/assembly\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(`package assembly

type Item struct {
	Value int32
}

type Box struct {
	Item Item
}

func Use(left, right Box) Box {
	var result Box
	result = left
	if left == right {
		return result
	}
	return right
}

func Trigger() int32 {
	var left Box
	var right Box
	left, right = right, left
	if left == right {
		return 1
	}
	return 0
}

func Caller() int32 {
	return Trigger()
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

func drainProgramSession(t *testing.T, session *programSession) {
	t.Helper()
	for {
		if object, ok := session.scheduler.next(); ok {
			if err := session.emit(object); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if requirements, ok := session.requirements.nextBatch(); ok {
			if err := session.applyDeclarationRequirements(requirements); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if object, ok := session.artifacts.NextDirty(); ok {
			if err := session.reconstructScheduledArtifact(object); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if sourcePackage, ok := session.packageInitializations.next(); ok {
			if err := session.emitPackageInitialization(sourcePackage); err != nil {
				t.Fatal(err)
			}
			continue
		}
		return
	}
}

func declarationForObject(
	t *testing.T,
	session *programSession,
	object types.Object,
) *targetDeclaration {
	t.Helper()
	site := session.sites[object]
	builder, err := session.builder(site)
	if err != nil {
		t.Fatal(err)
	}
	index, ok := builder.indexByOwner[sourceArtifactOwner(object)]
	if !ok {
		t.Fatalf("declaration %s is absent", object.Name())
	}
	return &builder.declarations[index]
}

func assertOneFinalDeclarationAssembly(
	t *testing.T,
	files []TargetFile,
	owner string,
) {
	t.Helper()
	classCount := 0
	operationCounts := map[string]int{
		"$zero":  0,
		"$copy":  0,
		"$equal": 0,
	}
	for _, file := range files {
		if file.Kind() != TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			switch statement := statement.(type) {
			case tsgo.ClassDeclaration:
				if statement.Name().Text() != owner {
					continue
				}
				classCount++
				for _, member := range statement.Members() {
					method, ok := member.(tsgo.MethodDeclaration)
					if !ok {
						continue
					}
					operationCounts[method.Name().(tsgo.Identifier).Text()]++
				}
			case tsgo.FunctionDeclaration:
				if strings.HasPrefix(statement.Name().Text(), owner+"$") {
					t.Fatalf("top-level operation helper %s remains", statement.Name().Text())
				}
			}
		}
	}
	if classCount != 1 {
		t.Fatalf("%s final class count = %d, want one", owner, classCount)
	}
	for name, count := range operationCounts {
		if count != 1 {
			t.Fatalf("%s.%s final definition count = %d, want one", owner, name, count)
		}
	}
}
