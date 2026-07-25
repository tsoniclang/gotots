package tsgo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinnedContractMatchesCheckedInInputs(t *testing.T) {
	contract, err := VerifyPinnedContract(schemaDirectory())
	if err != nil {
		t.Fatal(err)
	}
	if contract.Revision() != "c78d39e7075b4fc641b12b1f35d905c54cdc13ef" {
		t.Fatalf("revision = %q", contract.Revision())
	}
	if contract.ProtocolVersion() != 5 {
		t.Fatalf("protocol version = %d", contract.ProtocolVersion())
	}
	if contract.ToolVersion() != "v0.0.0-20260613021236-c78d39e7075b" {
		t.Fatalf("tool version = %q", contract.ToolVersion())
	}
	if len(contract.Files()) != 7 {
		t.Fatalf("pinned files = %d, want 7", len(contract.Files()))
	}
}

func TestPinnedContractFilesAreIsolated(t *testing.T) {
	contract, err := VerifyPinnedContract(schemaDirectory())
	if err != nil {
		t.Fatal(err)
	}
	files := contract.Files()
	files[0] = ContractFile{}
	if contract.Files()[0].Path() == "" {
		t.Fatal("Files exposed mutable backing storage")
	}
}

func TestCheckedInInputsMatchPinnedToolModule(t *testing.T) {
	contract, err := VerifyPinnedContract(schemaDirectory())
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(
		"go",
		"list",
		"-m",
		"-f",
		"{{.Dir}}",
		contract.Module()+"@"+contract.ToolVersion(),
	)
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	moduleDirectory := strings.TrimSpace(string(output))

	for _, file := range contract.Files() {
		checkedIn, err := os.ReadFile(filepath.Join(
			schemaDirectory(),
			filepath.FromSlash(file.Path()),
		))
		if err != nil {
			t.Fatal(err)
		}
		upstream, err := os.ReadFile(filepath.Join(
			moduleDirectory,
			filepath.FromSlash(file.SourcePath()),
		))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(checkedIn, upstream) {
			t.Fatalf("%s differs from %s", file.Path(), file.SourcePath())
		}
	}
}

func TestPinnedContractRejectsInputDrift(t *testing.T) {
	fixture := copySchemaDirectory(t)
	path := filepath.Join(fixture, "upstream", "protocol.ts")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[0] ^= 1
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = VerifyPinnedContract(fixture)
	var contractError *ContractError
	if !errors.As(err, &contractError) {
		t.Fatalf("error = %v, want ContractError", err)
	}
}

func TestPinnedContractRejectsUnlistedInput(t *testing.T) {
	fixture := copySchemaDirectory(t)
	if err := os.WriteFile(
		filepath.Join(fixture, "upstream", "extra.ts"),
		[]byte("export {};"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := VerifyPinnedContract(fixture); err == nil {
		t.Fatal("VerifyPinnedContract accepted an unlisted input")
	}
}

func TestPinnedContractRejectsUnknownManifestField(t *testing.T) {
	fixture := copySchemaDirectory(t)
	path := filepath.Join(fixture, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifestValue map[string]any
	if err := json.Unmarshal(data, &manifestValue); err != nil {
		t.Fatal(err)
	}
	manifestValue["unexpected"] = true
	data, err = json.Marshal(manifestValue)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := VerifyPinnedContract(fixture); err == nil {
		t.Fatal("VerifyPinnedContract accepted an unknown manifest field")
	}
}

func schemaDirectory() string {
	return filepath.Join("..", "..", "..", "schema", "tsgo")
}

func copySchemaDirectory(t *testing.T) string {
	t.Helper()
	source := schemaDirectory()
	destination := filepath.Join(t.TempDir(), "tsgo")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return destination
}
