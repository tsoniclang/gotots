package conversion_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestRawAddressConversionsRetainExactSelectedDomain(test *testing.T) {
	for _, architecture := range []string{"386", "amd64"} {
		test.Run(architecture, func(subtest *testing.T) {
			directory := subtest.TempDir()
			writeFile(subtest, filepath.Join(directory, "go.mod"), "module example.com/address\n\ngo 1.26.4\n")
			writeFile(subtest, filepath.Join(directory, "source.go"), `package conversion
import "unsafe"
type Address uintptr
type Raw unsafe.Pointer
func Read(pointer unsafe.Pointer) uintptr { return uintptr(pointer) }
func Restore(address uintptr) unsafe.Pointer { return unsafe.Pointer(address) }
func Named(pointer Raw) Address { return Address(uintptr(pointer)) }
func RestoreNamed(address Address) Raw { return Raw(unsafe.Pointer(uintptr(address))) }
func Zero() unsafe.Pointer { return unsafe.Pointer(uintptr(0)) }
`)
			profile, err := load.NewBuildProfile("linux", architecture, false, nil)
			if err != nil {
				subtest.Fatal(err)
			}
			loaded, err := load.One(context.Background(), load.Request{Directory: directory, Pattern: ".", BuildProfile: profile})
			if err != nil {
				subtest.Fatal(err)
			}
			var roots []emit.Root
			for _, name := range []string{"Read", "Restore", "Named", "RestoreNamed", "Zero"} {
				root, err := emit.NewRoot(loaded.Types().Scope().Lookup(name))
				if err != nil {
					subtest.Fatal(err)
				}
				roots = append(roots, root)
			}
			options := emit.DefaultOptions()
			options.IntegerRepresentation = emit.IntegerRepresentationBigInt
			emission, err := emit.CompileWithOptions(loaded.Program(), roots, options)
			if err != nil {
				subtest.Fatal(err)
			}
			strictTypecheckEmission(subtest, emission)
			_, _, printed := printConversions(subtest, subtest.TempDir(), emission)
			domain, abi := "uint64", "little64"
			if architecture == "386" {
				domain, abi = "uint32", "little32"
			}
			for _, required := range []string{"rawPointerToAddressInteger<" + domain + ">", "addressIntegerToRawPointer<" + domain + ">", abi} {
				if !strings.Contains(printed, required) {
					subtest.Fatalf("address output lacks %q", required)
				}
			}
			for _, forbidden := range []string{"Number(", " as unknown", " as any", "nativeUint"} {
				if strings.Contains(printed, forbidden) {
					subtest.Fatalf("address output contains %q", forbidden)
				}
			}
		})
	}
}
