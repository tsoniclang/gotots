package emit

import (
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func drainProgramSession(t *testing.T, session *programSession) {
	t.Helper()
	if err := session.settle(); err != nil {
		t.Fatal(err)
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
