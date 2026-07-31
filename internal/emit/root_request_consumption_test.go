package emit

import (
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestArtifactRequestConsumptionIsLinearInPersistentGraph(t *testing.T) {
	program := loadDeclarationAssemblyFixture(t)
	scope := program.Roots()[0].Types().Scope()
	box := scope.Lookup("Box").(*types.TypeName)
	use := scope.Lookup("Use")
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	request, err := api.NewNamedStructOperationRequest(
		box,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	requests := []api.RootRequest{request}
	for range 24 {
		requests = api.CombineRequests(requests, requests)
	}

	placement, dependencies, requirements, err :=
		session.consumeArtifactRequests(
			api.MustSourceArtifactOwner(use),
			requests,
		)
	if err != nil {
		t.Fatal(err)
	}
	if len(placement.Requests()) != 0 {
		t.Fatalf("placement requests = %d, want 0", len(placement.Requests()))
	}
	if len(dependencies) != 0 {
		t.Fatalf("dependencies = %d, want 0", len(dependencies))
	}
	if len(requirements) != 1 {
		t.Fatalf("requirements = %d, want 1", len(requirements))
	}
}
