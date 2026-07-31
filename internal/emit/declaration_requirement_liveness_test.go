package emit

import (
	"context"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRequirementRemovalWaitsForQuiescentConsumerDiscovery(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/liveness", "liveness")
	provider := types.NewTypeName(token.Pos(1), sourcePackage, "Record", nil)
	firstConsumer := api.MustSourceArtifactOwner(types.NewFunc(
		token.Pos(2),
		sourcePackage,
		"First",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	))
	laterConsumer := api.MustSourceArtifactOwner(types.NewFunc(
		token.Pos(3),
		sourcePackage,
		"Later",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	))
	copyRequirement, err := api.NewNamedStructOperationRequirement(
		provider,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	equalRequirement, err := api.NewNamedStructOperationRequirement(
		provider,
		api.NamedStructOperationEqual,
	)
	if err != nil {
		t.Fatal(err)
	}
	scheduler := newDeclarationRequirementScheduler(
		emitordering.CompareArtifactOwners,
	)
	scheduler.replace(
		firstConsumer,
		[]api.DeclarationRequirement{copyRequirement},
	)
	_, _, _, _ = scheduler.nextBatch()

	scheduler.replace(
		firstConsumer,
		[]api.DeclarationRequirement{equalRequirement},
	)
	owner, requirements, removed, ok := scheduler.nextBatch()
	if !ok ||
		removed ||
		owner != copyRequirement.Owner() ||
		len(requirements) != 2 {
		t.Fatalf(
			"replacement addition = %v %#v removed=%t ok=%t",
			owner,
			requirements,
			removed,
			ok,
		)
	}
	scheduler.replace(
		laterConsumer,
		[]api.DeclarationRequirement{copyRequirement},
	)
	if scheduler.finalizeRemovals() {
		t.Fatal("later consumer did not cancel the pending removal")
	}
	if scheduler.hasPending() {
		t.Fatal("canceled removal left pending scheduler work")
	}
}

func TestEnvironmentProfileDisappearsWithItsLastConsumerRevision(
	t *testing.T,
) {
	directory := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/profileliveness\n\ngo 1.26.4\n",
		"source.go": `package profileliveness

import "slices"

func Sum(values []int32, input <-chan int32) int32 {
	var total int32
	for value := range slices.Values(values) {
		total += value + <-input
	}
	return total
}
`,
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name),
			[]byte(content),
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
	options := DefaultOptions()
	options.ConcurrencySemantics = ConcurrencySemanticsCooperative
	session, err := newProgramSession(program, options)
	if err != nil {
		t.Fatal(err)
	}
	sum := program.Roots()[0].Types().Scope().Lookup("Sum")
	if err := session.require(sum); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)

	var profileRequirement api.DeclarationRequirement
	var consumer api.ArtifactOwner
	for requirement, consumers := range session.requirements.consumers {
		profile, ok := requirement.GenericCallableProfile()
		if !ok ||
			profile.Owner().Pkg() == nil ||
			profile.Owner().Pkg().Path() != "slices" ||
			profile.Owner().Name() != "Values" {
			continue
		}
		if profileRequirement.Valid() || len(consumers) != 1 {
			t.Fatal("Values profile requirement has non-exact consumers")
		}
		profileRequirement = requirement
		for selected := range consumers {
			consumer = selected
		}
	}
	if !profileRequirement.Valid() || !consumer.Valid() {
		t.Fatal("Values profile requirement was not materialized")
	}
	values, _ := profileRequirement.Owner().Source()
	builder := session.environmentBuilders[program.EnvironmentForTypes(values.Pkg())]
	if builder == nil {
		t.Fatal("slices environment builder is absent")
	}
	assertEnvironmentProfileCount(
		t,
		builder.declarations[values].statements,
		"Values$cooperative_",
		1,
	)

	current := session.requirements.byConsumer[consumer]
	next := make(
		[]api.DeclarationRequirement,
		0,
		len(current)-1,
	)
	for requirement := range current {
		if requirement != profileRequirement {
			next = append(next, requirement)
		}
	}
	session.requirements.replace(consumer, next)
	if !session.requirements.finalizeRemovals() {
		t.Fatal("final profile removal was not scheduled at quiescence")
	}
	owner, requirements, removed, ok := session.requirements.nextBatch()
	if !ok || !removed || owner != profileRequirement.Owner() {
		t.Fatalf("profile removal batch = %v %#v %t", owner, requirements, ok)
	}
	if err := session.applyDeclarationRequirements(
		owner,
		requirements,
		removed,
	); err != nil {
		t.Fatal(err)
	}
	assertEnvironmentProfileCount(
		t,
		builder.declarations[values].statements,
		"Values$cooperative_",
		0,
	)
	if session.requirements.wasApplied(profileRequirement) {
		t.Fatal("removed Values profile survived its final consumer")
	}
}

func assertEnvironmentProfileCount(
	t *testing.T,
	statements []tsgo.Statement,
	prefix string,
	want int,
) {
	t.Helper()
	actual := 0
	for _, statement := range statements {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok &&
			function.Name() != nil &&
			strings.HasPrefix(function.Name().Text(), prefix) {
			actual++
		}
	}
	if actual != want {
		t.Fatalf(
			"environment profile %s count = %d, want %d",
			prefix,
			actual,
			want,
		)
	}
}
