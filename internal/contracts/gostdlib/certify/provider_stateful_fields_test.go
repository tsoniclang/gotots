package certify

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestStatefulProfileTargetFieldsExactJoin(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	_, selectedTSGo := resolveTestTools(t, repository)
	client, err := tsgo.StartClientWithTool(selectedTSGo, provider)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	project, err := client.OpenProject(filepath.Join(provider, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	exports, err := project.Exports(filepath.Join(
		provider,
		"src",
		"internal",
		"facets",
		"provider-compress-gzip.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	var target tsgo.ProjectExport
	for _, selected := range exports {
		if selected.Name() == "CanonicalGzipReader" {
			target = selected
			break
		}
	}
	if target.Name() == "" {
		t.Fatal("canonical gzip stateful target is absent")
	}
	fields := []gostdlib.ProviderStructFieldDocument{{Member: "Header"}}
	methods := []gostdlib.ProviderStatefulProfileMethodDocument{
		{Member: "Close"},
		{Member: "Read"},
	}
	operations := []gostdlib.FacetCapability{
		gostdlib.FacetCapabilityAssign,
		gostdlib.FacetCapabilityCopy,
	}
	if err := verifyStatefulProfileTargetMembers(
		target,
		fields,
		methods,
		operations,
	); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		fields []gostdlib.ProviderStructFieldDocument
		want   string
	}{
		{
			name: "omitted",
			want: "unowned public instance member",
		},
		{
			name: "extra",
			fields: []gostdlib.ProviderStructFieldDocument{
				{Member: "Header"},
				{Member: "Invented"},
			},
			want: "omits an owned public instance member",
		},
		{
			name:   "renamed",
			fields: []gostdlib.ProviderStructFieldDocument{{Member: "Renamed"}},
			want:   "unowned public instance member",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := verifyStatefulProfileTargetMembers(
				target,
				test.fields,
				methods,
				operations,
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("field mutation error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestStatefulProfileOperationsExactJoin(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	_, selectedTSGo := resolveTestTools(t, repository)
	client, err := tsgo.StartClientWithTool(selectedTSGo, provider)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	project, err := client.OpenProject(filepath.Join(provider, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	exports, err := project.Exports(filepath.Join(
		provider,
		"src",
		"internal",
		"facets",
		"named-io-fs.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	var target tsgo.ProjectExport
	for _, selected := range exports {
		if selected.Name() == "CanonicalPathError" {
			target = selected
			break
		}
	}
	if target.Name() == "" {
		t.Fatal("canonical path-error target is absent")
	}
	fields := []gostdlib.ProviderStructFieldDocument{
		{Member: "Err"},
		{Member: "Op"},
		{Member: "Path"},
	}
	methods := []gostdlib.ProviderStatefulProfileMethodDocument{
		{Member: "Error"},
		{Member: "Unwrap"},
	}
	operations := []gostdlib.FacetCapability{
		gostdlib.FacetCapabilityAssign,
		gostdlib.FacetCapabilityCopy,
		gostdlib.FacetCapabilityEqual,
		gostdlib.FacetCapabilityHash,
		gostdlib.FacetCapabilityMake,
		gostdlib.FacetCapabilityStorage,
	}
	if err := verifyStatefulProfileTargetMembers(
		target,
		fields,
		methods,
		operations,
	); err != nil {
		t.Fatal(err)
	}
	err = verifyStatefulProfileTargetMembers(
		target,
		fields,
		methods,
		operations[:len(operations)-1],
	)
	if err == nil || !strings.Contains(err.Error(), "unowned public static member") {
		t.Fatalf("dropped storage-operation error = %v", err)
	}
}

func TestStatefulExecutionProfilesRequireExactSynchronousSibling(t *testing.T) {
	cooperative := statefulExecutionProfileFixture(
		gostdlib.EffectAwaitable,
		gostdlib.EffectAsynchronous,
	)
	direct := statefulExecutionProfileFixture(
		gostdlib.EffectSynchronous,
		gostdlib.EffectSynchronous,
	)
	modules := []gostdlib.FacetModuleDocument{{
		StatefulProfiles: []gostdlib.ProviderStatefulProfileDocument{
			cooperative,
			direct,
		},
	}}
	if err := verifyStatefulExecutionProfilePairs(modules); err != nil {
		t.Fatal(err)
	}

	mutations := []struct {
		name     string
		profiles []gostdlib.ProviderStatefulProfileDocument
	}{
		{name: "missing", profiles: []gostdlib.ProviderStatefulProfileDocument{
			cooperative,
		}},
		{name: "interface-shape", profiles: func() []gostdlib.ProviderStatefulProfileDocument {
			mutated := statefulExecutionProfileFixture(
				gostdlib.EffectSynchronous,
				gostdlib.EffectSynchronous,
			)
			mutated.Interfaces[0].ProviderInterface.Methods[0].SourceSignature =
				"func(int) string"
			return []gostdlib.ProviderStatefulProfileDocument{cooperative, mutated}
		}()},
		{name: "method-shape", profiles: func() []gostdlib.ProviderStatefulProfileDocument {
			mutated := statefulExecutionProfileFixture(
				gostdlib.EffectSynchronous,
				gostdlib.EffectSynchronous,
			)
			mutated.Methods[0].SourceSignature = "func(int) string"
			return []gostdlib.ProviderStatefulProfileDocument{cooperative, mutated}
		}()},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			err := verifyStatefulExecutionProfilePairs(
				[]gostdlib.FacetModuleDocument{{StatefulProfiles: mutation.profiles}},
			)
			if err == nil || !strings.Contains(
				err.Error(),
				"suspending profile has no exact synchronous sibling",
			) {
				t.Fatalf("mutation error = %v", err)
			}
		})
	}
}

func statefulExecutionProfileFixture(
	interfaceEffect gostdlib.EffectKind,
	methodEffect gostdlib.EffectKind,
) gostdlib.ProviderStatefulProfileDocument {
	return gostdlib.ProviderStatefulProfileDocument{
		SourceIdentity: "example.com/provider|kind=2|receiver=|name=State",
		Interfaces: []gostdlib.ProviderCallableProfileInterfaceDocument{{
			SourceIdentity: "example.com/provider|kind=2|receiver=|name=Protocol",
			ProviderInterface: gostdlib.ProviderInterfaceDocument{
				Mode: gostdlib.ProviderInterfaceModeBridge,
				Methods: []gostdlib.ProviderInterfaceMethodDocument{{
					SourceIdentity:    "example.com/provider|kind=4|receiver=Protocol|name=Read",
					Kind:              gostdlib.ProviderInterfaceMethodCallable,
					Effect:            interfaceEffect,
					SourceSignature:   "func() string",
					ContractSignature: "func() string",
				}},
			},
		}},
		TypeArguments: []string{
			"example.com/provider|kind=2|receiver=|name=Protocol",
		},
		Operations: []gostdlib.FacetCapability{gostdlib.FacetCapabilityCopy},
		Fields: []gostdlib.ProviderStructFieldDocument{{
			Member:          "Value",
			Ordinal:         0,
			SourceSignature: "string",
		}},
		Methods: []gostdlib.ProviderStatefulProfileMethodDocument{{
			SourceIdentity:  "example.com/provider|kind=4|receiver=*State|name=Read",
			Member:          "Read",
			Effect:          methodEffect,
			SourceSignature: "func() string",
		}},
	}
}
