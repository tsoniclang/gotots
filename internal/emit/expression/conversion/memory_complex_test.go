package conversion_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestRawComplexLayoutRetainsPhysicalComponents(test *testing.T) {
	for _, selected := range []struct {
		name      string
		component string
		offset    string
		alignment string
		size      string
		arch      string
	}{
		{"complex64", "float32", "4", "4", "8", "amd64"},
		{"complex128", "float64", "8", "8", "16", "amd64"},
		{"complex64", "float32", "4", "4", "8", "386"},
		{"complex128", "float64", "8", "4", "16", "386"},
	} {
		test.Run(selected.name+"/"+selected.arch, func(test *testing.T) {
			profile, err := load.NewBuildProfile("linux", selected.arch, false, nil)
			if err != nil {
				test.Fatal(err)
			}
			loaded := loadMemoryStorageProfile(test, `
type Named `+selected.name+`
type Alias = `+selected.name+`
type Record struct { Value Named }
func Convert(value *`+selected.name+`) *`+selected.name+` { return (*`+selected.name+`)(unsafe.Pointer(value)) }
func ConvertNamed(value *Named) *Named { return (*Named)(unsafe.Pointer(value)) }
func ConvertAlias(value *Alias) *Alias { return (*Alias)(unsafe.Pointer(value)) }
func ConvertRecord(value *Record) *Record { return (*Record)(unsafe.Pointer(value)) }
`, profile)
			var roots []emit.Root
			for _, name := range []string{"Convert", "ConvertNamed", "ConvertAlias", "ConvertRecord"} {
				root, err := emit.NewRoot(loaded.Types().Scope().Lookup(name))
				if err != nil {
					test.Fatal(err)
				}
				roots = append(roots, root)
			}
			emission, err := emit.Compile(loaded.Program(), roots)
			if err != nil {
				test.Fatal(err)
			}
			strictTypecheckEmission(test, emission)
			_, _, printed := printConversions(test, test.TempDir(), emission)
			class := "GoC" + selected.name[1:]
			storage := class + "Storage"
			for _, required := range []string{
				"export const " + storage + ":",
				"real: field<" + selected.component + ">()",
				"imag: field<" + selected.component + ">()",
				"memoryLayout<typeof " + storage + ">",
				", " + selected.size + ", " + selected.alignment + ", " + selected.size + ", memoryField",
				".real, 0, " + selected.alignment + ", memoryLayout<" + selected.component + ">",
				".imag, " + selected.offset + ", " + selected.alignment + ", memoryLayout<" + selected.component + ">",
				"goC" + selected.name[1:] + "ToStorage",
				"goC" + selected.name[1:] + "FromStorage",
				"return { real: value.real, imag: value.imag }",
				"return " + class + ".make(value.real, value.imag)",
			} {
				if !strings.Contains(printed, required) {
					test.Fatalf("physical complex output lacks %q", required)
				}
			}
			if strings.Count(printed, "export const "+storage+":") != 1 {
				test.Fatal("complex storage schema is not demand-deduplicated")
			}
			if strings.Contains(printed, "memoryLayout<"+class+">") {
				test.Fatal("logical complex class acquired a physical layout")
			}
		})
	}
}
