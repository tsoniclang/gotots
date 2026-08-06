package emit

import (
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func drainProgramSession(t *testing.T, session *programSession) {
	t.Helper()
	for {
		if object, ok := session.scheduler.next(); ok {
			if err := session.emit(object); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if owner, requirements, removed, ok :=
			session.requirements.nextBatch(); ok {
			if err := session.applyDeclarationRequirements(
				owner,
				requirements,
				removed,
			); err != nil {
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
		if session.requirements.finalizeRemovals() {
			continue
		}
		if builders := session.packageExports.nextBatch(); len(builders) != 0 {
			for _, builder := range builders {
				if err := session.publishPackageExports(builder); err != nil {
					t.Fatal(err)
				}
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
