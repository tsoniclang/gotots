package verify

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func abiDependencySurface(relative string, source []byte) (string, error) {
	text := string(source)
	shared := "@" + "tso" + "nic/"
	if relative == "abi/package.json" {
		var document map[string]json.RawMessage
		if err := json.Unmarshal(source, &document); err != nil {
			return "", err
		}
		var peers map[string]string
		if err := json.Unmarshal(document["peerDependencies"], &peers); err != nil {
			return "", err
		}
		for name := range peers {
			if name != shared+"source-core" && name != shared+"target-api" && name != shared+"tsts" {
				return "", &wallError{source: relative, reason: "ABI adapter has an unapproved shared peer"}
			}
		}
		delete(document, "peerDependencies")
		remaining, err := json.Marshal(document)
		return string(remaining), err
	}
	if !strings.HasPrefix(relative, "abi/src/") || !strings.HasSuffix(relative, ".ts") {
		return text, nil
	}
	if strings.HasSuffix(relative, ".test.ts") {
		for _, module := range []string{"tsts", "source-core", "source-core/facts"} {
			text = strings.ReplaceAll(text, `"`+shared+module+`"`, `"abi-test-contract"`)
		}
		return text, nil
	}
	for _, module := range []string{"tsts", "source-core", "target-api/provider"} {
		pattern := `(?m)^import\s+type\s*\{[^{};]+\}\s*from\s*["']` + regexp.QuoteMeta(shared+module) + `["'];\s*$`
		text = regexp.MustCompile(pattern).ReplaceAllString(text, "")
	}
	construction := `(?m)^import\s*\{\s*createSourceSemanticsVirtualModuleProvider\s*\}\s*from\s*["']` +
		regexp.QuoteMeta(shared+"source-core/extension") + `["'];\s*$`
	return regexp.MustCompile(construction).ReplaceAllString(text, ""), nil
}

func TestABIAdapterDependencyIsolation(t *testing.T) {
	shared := "@" + "tso" + "nic/"
	typeImport := `import type { CompilerExtension } from "` + shared + `tsts";`
	construction := `import { createSourceSemanticsVirtualModuleProvider } from "` + shared + `source-core/extension";`
	checker := `import { createCompilerSessionFromFiles } from "` + shared + `tsts";`
	for _, source := range []string{typeImport, construction} {
		if err := verifyCompilerDependencyText("abi/src/index.ts", []byte(source)); err != nil {
			t.Fatal(err)
		}
		if err := verifyCompilerDependencyText("internal/emit/leak.ts", []byte(source)); err == nil {
			t.Fatal("ABI adapter dependency entered compiler source")
		}
	}
	if err := verifyCompilerDependencyText("abi/src/index.test.ts", []byte(checker)); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{
		checker,
		`import { createTsonicCoreSourceExtension } from "` + shared + `source-core";`,
		`import { location } from "` + shared + `typescript-runtime";`,
		`import type { Target } from "` + shared + `target-typescript";`,
	} {
		if err := verifyCompilerDependencyText("abi/src/index.ts", []byte(mutation)); err == nil {
			t.Fatalf("ABI production dependency escaped isolation: %s", mutation)
		}
	}
	for _, module := range []string{"typescript-runtime", "target-typescript", "host"} {
		manifest := `{"peerDependencies":{"` + shared + module + `":"0.1.1"}}`
		if err := verifyCompilerDependencyText("abi/package.json", []byte(manifest)); err == nil {
			t.Fatalf("ABI manifest admitted executable peer %s", module)
		}
	}
	manifest := `{"peerDependencies":{"` + shared + `tsts":"0.1.1"}}`
	if err := verifyCompilerDependencyText("abi/package.json", []byte(manifest)); err != nil {
		t.Fatal(err)
	}
	manifest = `{"dependencies":{"` + shared + `tsts":"0.1.1"},"peerDependencies":{}}`
	if err := verifyCompilerDependencyText("abi/package.json", []byte(manifest)); err == nil {
		t.Fatal("ABI executable dependency escaped peer-only surface")
	}
}
