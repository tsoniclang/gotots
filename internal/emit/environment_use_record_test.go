package emit

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"testing"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/load"
)

func writeUseRecordFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func environmentUseSession(t *testing.T) (*programSession, *load.Program) {
	t.Helper()
	project := t.TempDir()
	writeUseRecordFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/userecord\n\ngo 1.26.4\n",
	)
	writeUseRecordFile(t, filepath.Join(project, "source.go"), `package userecord

import "strings"

func Trim(input string) string {
	return strings.TrimSpace(input)
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	return session, program
}

func environmentLookup(
	t *testing.T,
	program *load.Program,
	packagePath string,
	name string,
) types.Object {
	t.Helper()
	for _, environmentPackage := range program.EnvironmentPackages() {
		if environmentPackage.Path() != packagePath {
			continue
		}
		object := environmentPackage.Types().Scope().Lookup(name)
		if object == nil {
			t.Fatalf("environment package %q has no %q", packagePath, name)
		}
		return object
	}
	t.Fatalf("environment package %q was not loaded", packagePath)
	return nil
}

// TestUseRecordJoinsMonotonicDemandOnCanonicalScheduler proves a type-only
// use followed by a callable use upgrades the same canonical scheduler
// record monotonically, and that every demand class enters that record.
func TestUseRecordJoinsMonotonicDemandOnCanonicalScheduler(t *testing.T) {
	session, program := environmentUseSession(t)
	trimSpace := environmentLookup(t, program, "strings", "TrimSpace")
	if err := session.RequireUse(
		trimSpace,
		environmentidentity.UseDemandTypeContract,
		gostdlib.NoUseSelection(),
	); err != nil {
		t.Fatal(err)
	}
	record := session.scheduler.records[trimSpace]
	if record == nil {
		t.Fatal("environment use created no canonical scheduler record")
	}
	if !slices.Equal(
		record.demandList(),
		[]environmentidentity.UseDemand{
			environmentidentity.UseDemandTypeContract,
		},
	) {
		t.Fatalf("initial demands = %v", record.demandList())
	}
	if err := session.RequireUse(
		trimSpace,
		environmentidentity.UseDemandCallable,
		gostdlib.NoUseSelection(),
	); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(
		record.demandList(),
		[]environmentidentity.UseDemand{
			environmentidentity.UseDemandTypeContract,
			environmentidentity.UseDemandCallable,
		},
	) {
		t.Fatalf(
			"callable use did not upgrade type-only demand: %v",
			record.demandList(),
		)
	}
	for _, demand := range []environmentidentity.UseDemand{
		environmentidentity.UseDemandValue,
		environmentidentity.UseDemandState,
		environmentidentity.UseDemandInitializer,
		environmentidentity.UseDemandInterfaceCapability,
		environmentidentity.UseDemandCallbackCapability,
		environmentidentity.UseDemandRuntimeFacet,
	} {
		if err := session.RequireUse(
			trimSpace,
			demand,
			gostdlib.NoUseSelection(),
		); err != nil {
			t.Fatal(err)
		}
	}
	if len(record.demandList()) != 8 {
		t.Fatalf(
			"joined demand classes = %v, want all eight",
			record.demandList(),
		)
	}
	if record.route != environmentidentity.RouteBoundary {
		t.Fatalf(
			"compile-only provider route = %v, want boundary",
			record.route,
		)
	}
	if !record.emitting {
		t.Fatal("selection route did not mark the record emitting")
	}
}

// TestUseRecordRejectsSecondImplementationRoute proves one scalar route:
// repeated identical observations succeed and a different route for the
// same declaration fails immediately.
func TestUseRecordRejectsSecondImplementationRoute(t *testing.T) {
	session, program := environmentUseSession(t)
	trimSpace := environmentLookup(t, program, "strings", "TrimSpace")
	if err := session.RequireUse(
		trimSpace,
		environmentidentity.UseDemandCallable,
		gostdlib.NoUseSelection(),
	); err != nil {
		t.Fatal(err)
	}
	if err := session.RequireUse(
		trimSpace,
		environmentidentity.UseDemandCallable,
		gostdlib.NoUseSelection(),
	); err != nil {
		t.Fatalf("repeated identical route observation failed: %v", err)
	}
	if err := session.ObserveImplementation(
		trimSpace,
		environmentidentity.UseDemandCallable,
		environmentidentity.RouteGeneratedFacet,
	); err == nil {
		t.Fatal("second implementation route for one declaration succeeded")
	}
	record := session.scheduler.records[trimSpace]
	if record.route != environmentidentity.RouteBoundary {
		t.Fatalf(
			"conflicting observation mutated the settled route: %v",
			record.route,
		)
	}
}

// TestObservedImplementationRecordIsNonEmitting proves intrinsic/generated
// routes create canonical non-emitting scheduler records rather than a
// separate ledger, and never schedule an environment declaration artifact.
func TestObservedImplementationRecordIsNonEmitting(t *testing.T) {
	session, program := environmentUseSession(t)
	trimSpace := environmentLookup(t, program, "strings", "TrimSpace")
	if err := session.ObserveImplementation(
		trimSpace,
		environmentidentity.UseDemandCallable,
		environmentidentity.RouteGeneratedFacet,
	); err != nil {
		t.Fatal(err)
	}
	record := session.scheduler.records[trimSpace]
	if record == nil {
		t.Fatal("observation created no canonical scheduler record")
	}
	if record.emitting || record.queued || record.emitted {
		t.Fatalf(
			"observed implementation record schedules emission: %+v",
			record,
		)
	}
	if record.route != environmentidentity.RouteGeneratedFacet {
		t.Fatalf("observed route = %v", record.route)
	}
	if err := session.RequireUse(
		trimSpace,
		environmentidentity.UseDemandCallable,
		gostdlib.NoUseSelection(),
	); err == nil {
		t.Fatal("selection route joined an observed compiler-owned declaration")
	}
}

// TestUnobservedEnvironmentSettlementFailsProjection proves that removing
// the use observation from a settled environment declaration fails the
// final projection at its owning gate.
func TestUnobservedEnvironmentSettlementFailsProjection(t *testing.T) {
	session, program := environmentUseSession(t)
	trimSpace := environmentLookup(t, program, "strings", "TrimSpace")
	environmentPackage := program.EnvironmentForTypes(trimSpace.Pkg())
	if environmentPackage == nil {
		t.Fatal("strings package is not environment-owned")
	}
	if _, err := session.requireEnvironmentPackage(
		environmentPackage,
	); err != nil {
		t.Fatal(err)
	}
	// Mutation control: bypass RequireUse so the declaration settles with no
	// recorded use, then demand the projection.
	session.scheduler.enqueue(trimSpace)
	for {
		object, ok := session.scheduler.next()
		if !ok {
			break
		}
		if err := session.emit(object); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := session.environmentObligations(); err == nil {
		t.Fatal("unobserved environment settlement passed the projection gate")
	}
}

// TestProjectionExactJoinsSchedulerRecords proves the immutable projection
// and the canonical scheduler records join exactly in both directions.
func TestProjectionExactJoinsSchedulerRecords(t *testing.T) {
	project := t.TempDir()
	writeUseRecordFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/joinproof\n\ngo 1.26.4\n",
	)
	writeUseRecordFile(t, filepath.Join(project, "source.go"), `package joinproof

import "strings"

func Trim(input string) string {
	return strings.TrimSpace(input)
}

func Describe(reader *strings.Reader) bool {
	return reader == nil
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	for _, name := range []string{"Trim", "Describe"} {
		root, rootErr := NewRoot(scope.Lookup(name))
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		if err := session.requireRoot(root); err != nil {
			t.Fatal(err)
		}
	}
	drainProgramSession(t, session)
	obligations, err := session.environmentObligations()
	if err != nil {
		t.Fatal(err)
	}
	projected := make(map[string]int)
	for _, obligation := range obligations {
		projected[obligation.Identity()]++
	}
	recorded := 0
	for object, record := range session.scheduler.records {
		if record.route == environmentidentity.RouteInvalid {
			continue
		}
		recorded++
		description, describeErr := environmentidentity.Describe(object)
		if describeErr != nil {
			t.Fatal(describeErr)
		}
		if projected[description.Identity()] == 0 {
			t.Fatalf(
				"scheduler record %q is absent from the projection",
				description.Identity(),
			)
		}
	}
	if recorded == 0 {
		t.Fatal("no environment records were settled")
	}
	total := 0
	for _, count := range projected {
		total += count
	}
	if total != len(obligations) {
		t.Fatalf("projection identity join lost rows: %d != %d", total, len(obligations))
	}
	if len(projected) != recorded {
		t.Fatalf(
			"projection rows (%d identities) do not exact-join scheduler records (%d)",
			len(projected),
			recorded,
		)
	}
}
