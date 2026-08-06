package load

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJoinsEmbedDirectivesToToolchainSelectedFiles(t *testing.T) {
	project := t.TempDir()
	writeEmbedFixture(t, project, "go.mod", `module example.com/embedfixture

go 1.26.4
`)
	writeEmbedFixture(t, project, "source.go", `package embedfixture

import "embed"

//go:embed "payload file.txt"
var text string

//go:embed bytes.bin
var data []byte

//go:embed assets
var tree embed.FS

//go:embed all:assets
var allTree embed.FS
`)
	writeEmbedFixture(t, project, "payload file.txt", "selected text\n")
	writeEmbedFixture(t, project, "bytes.bin", "\x00\x01\xff")
	writeEmbedFixture(t, project, "assets/a.txt", "alpha")
	writeEmbedFixture(t, project, "assets/.hidden.txt", "hidden")
	writeEmbedFixture(t, project, "assets/nested/b.txt", "beta")

	loaded, err := One(context.Background(), Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertEmbeddedValue(
		t,
		loaded,
		"text",
		EmbedString,
		[]string{"payload file.txt"},
		[][]byte{[]byte("selected text\n")},
	)
	assertEmbeddedValue(
		t,
		loaded,
		"data",
		EmbedBytes,
		[]string{"bytes.bin"},
		[][]byte{{0, 1, 0xff}},
	)
	assertEmbeddedValue(
		t,
		loaded,
		"tree",
		EmbedFileSystem,
		[]string{"assets/a.txt", "assets/nested/b.txt"},
		[][]byte{[]byte("alpha"), []byte("beta")},
	)
	assertEmbeddedValue(
		t,
		loaded,
		"allTree",
		EmbedFileSystem,
		[]string{"assets/.hidden.txt", "assets/a.txt", "assets/nested/b.txt"},
		[][]byte{[]byte("hidden"), []byte("alpha"), []byte("beta")},
	)

	text := loaded.Types().Scope().Lookup("text").(*types.Var)
	first, ok := loaded.Embed(text)
	if !ok {
		t.Fatal("text embed evidence is absent")
	}
	files := first.Files()
	files[0].content[0] = 'X'
	second, ok := loaded.Embed(text)
	if !ok || string(second.Files()[0].Bytes()) != "selected text\n" {
		t.Fatal("embed evidence exposes mutable backing storage")
	}
}

func assertEmbeddedValue(
	t *testing.T,
	loaded *Package,
	name string,
	kind EmbedKind,
	names []string,
	contents [][]byte,
) {
	t.Helper()
	variable, ok := loaded.Types().Scope().Lookup(name).(*types.Var)
	if !ok {
		t.Fatalf("%s variable is absent", name)
	}
	value, ok := loaded.Embed(variable)
	if !ok || value.Kind() != kind {
		t.Fatalf("%s embed kind = %v, %v; want %v", name, value.Kind(), ok, kind)
	}
	files := value.Files()
	if len(files) != len(names) {
		t.Fatalf("%s embed files = %d, want %d", name, len(files), len(names))
	}
	for index, file := range files {
		if file.Name() != names[index] ||
			string(file.Bytes()) != string(contents[index]) {
			t.Fatalf(
				"%s embed file %d = %q %q, want %q %q",
				name,
				index,
				file.Name(),
				file.Bytes(),
				names[index],
				contents[index],
			)
		}
	}
}

func writeEmbedFixture(
	t *testing.T,
	root string,
	name string,
	content string,
) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
