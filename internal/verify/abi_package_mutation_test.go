package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestABIAdapterPackageMetadataIsolation(t *testing.T) {
	shared := "@" + "tso" + "nic/"
	for _, relative := range []string{"abi/package.json", "abi/package-lock.json"} {
		t.Run(relative, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(relative)))
			if err != nil {
				t.Fatal(err)
			}
			if err := verifyCompilerDependencyText(relative, source); err != nil {
				t.Fatal(err)
			}
			mutations := []string{
				strings.ReplaceAll(string(source), shared+"tsts", shared+"typescript-runtime"),
				strings.ReplaceAll(string(source), "file:../../tsonic/packages/tsts", "file:../../foreign/checker"),
			}
			var document map[string]json.RawMessage
			if err := json.Unmarshal(source, &document); err != nil {
				t.Fatal(err)
			}
			root := document
			if strings.HasSuffix(relative, "-lock.json") {
				mutations = append(mutations,
					strings.ReplaceAll(string(source), `"dev": true`, `"dev": false`),
					strings.ReplaceAll(string(source), `"link": true`, `"link": false`),
				)
				var packages map[string]json.RawMessage
				if err := json.Unmarshal(document["packages"], &packages); err != nil {
					t.Fatal(err)
				}
				root = make(map[string]json.RawMessage)
				if err := json.Unmarshal(packages[""], &root); err != nil {
					t.Fatal(err)
				}
				root["dependencies"] = json.RawMessage(`{"` + shared + `tsts":"0.1.1"}`)
				encodedRoot, err := json.Marshal(root)
				if err != nil {
					t.Fatal(err)
				}
				packages[""] = encodedRoot
				encodedPackages, err := json.Marshal(packages)
				if err != nil {
					t.Fatal(err)
				}
				document["packages"] = encodedPackages
			} else {
				root["dependencies"] = json.RawMessage(`{"` + shared + `tsts":"0.1.1"}`)
			}
			encoded, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			mutations = append(mutations, string(encoded))
			for index, mutation := range mutations {
				if mutation == string(source) {
					t.Fatalf("mutation %d did not change the metadata", index)
				}
				if err := verifyCompilerDependencyText(relative, []byte(mutation)); err == nil {
					t.Fatalf("mutation %d escaped the ABI package boundary", index)
				}
			}
		})
	}
}
