package load

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestPackageOwnerUsesExactModuleAndToolchainEvidence(t *testing.T) {
	membership := toolchainPackageMembership{
		standard: map[string]struct{}{
			"builtin": {},
			"fmt":     {},
			"unsafe":  {},
		},
		command: map[string]struct{}{
			"cmd/compile": {},
		},
	}
	tests := []struct {
		name        string
		selected    *packages.Package
		environment bool
		owner       PackageOwner
		key         string
	}{
		{
			"main module",
			&packages.Package{PkgPath: "example.test/app", Module: &packages.Module{Path: "example.test/app", Main: true}},
			false,
			PackageOwnerWorkspace,
			moduleContractKey("example.test/app", ""),
		},
		{
			"module dependency",
			&packages.Package{PkgPath: "example.test/dep", Module: &packages.Module{Path: "example.test/dep", Version: "v1.2.3"}},
			false,
			PackageOwnerModule,
			moduleContractKey("example.test/dep", "v1.2.3"),
		},
		{
			"external module",
			&packages.Package{PkgPath: "example.test/host", Module: &packages.Module{Path: "example.test/host", Version: "v2.0.0"}},
			true,
			PackageOwnerExternal,
			moduleContractKey("example.test/host", "v2.0.0"),
		},
		{"standard library", &packages.Package{PkgPath: "fmt"}, false, PackageOwnerStandardLibrary, "toolchain"},
		{"toolchain command", &packages.Package{PkgPath: "cmd/compile"}, false, PackageOwnerToolchain, "toolchain"},
		{"language package", &packages.Package{PkgPath: "unsafe"}, false, PackageOwnerLanguage, "toolchain"},
		{
			"cmd prefix foil",
			&packages.Package{PkgPath: "cmd/example"},
			false,
			PackageOwnerWorkspace,
			"workspace-package:cmd/example",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			owner, key, err := classifyPackageOwner(
				test.selected,
				"toolchain",
				membership,
				test.environment,
			)
			if err != nil {
				t.Fatal(err)
			}
			if owner != test.owner || key != test.key {
				t.Fatalf("owner = %v/%q, want %v/%q", owner, key, test.owner, test.key)
			}
		})
	}
}

func TestPackageOwnerRejectsUncorroboratedEnvironmentPackage(t *testing.T) {
	_, _, err := classifyPackageOwner(
		&packages.Package{PkgPath: "host/implicit"},
		"toolchain",
		toolchainPackageMembership{
			standard: map[string]struct{}{"fmt": {}},
			command:  map[string]struct{}{"cmd/compile": {}},
		},
		true,
	)
	if err == nil {
		t.Fatal("uncorroborated environment package was admitted")
	}
}
