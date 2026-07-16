package oracle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/goenv"
)

// kindsLiteral renders a case's result kinds as a source-code list literal
// shared by both driver languages ({"float", "value"} vs ["float","value"]).
func kindsLiteral(kinds []string, open, close string) string {
	parts := make([]string, len(kinds))
	for i, kind := range kinds {
		parts[i] = fmt.Sprintf("%q", kind)
	}
	return open + strings.Join(parts, ", ") + close
}

// runGoDriver generates and executes the Go-side driver: each case prints
// one line "Name: v1 v2" or "Name: panic: <message>". Floats print as
// exact IEEE-754 bit patterns so number formatting can never mask or
// invent a difference; strings print Go-quoted; everything else prints its
// exact decimal/boolean form.
func runGoDriver(resolved *goenv.Resolved, env []string, workDir string, cases []fixtureCase) (string, error) {
	var driver strings.Builder
	driver.WriteString(`package main

import (
	"encoding/hex"
	"fmt"
	"math"

	"oracle.fixture/fixture"
)

// formatText prints ASCII-printable text directly and anything else as
// exact bytes — the same rule the TS driver applies, so formatting can
// never mask or invent a byte difference.
func formatText(text string) string {
	printable := true
	for i := 0; i < len(text); i++ {
		if text[i] < 0x20 || text[i] > 0x7e || text[i] == '"' || text[i] == '\\' {
			printable = false
			break
		}
	}
	if printable {
		return "\"" + text + "\""
	}
	return "hex:" + hex.EncodeToString([]byte(text))
}

func formatValue(value any, kind string) string {
	if kind == "float" {
		switch f := value.(type) {
		case float64:
			return fmt.Sprintf("float64(0x%016x)", math.Float64bits(f))
		case float32:
			return fmt.Sprintf("float64(0x%016x)", math.Float64bits(float64(f)))
		}
	}
	if text, ok := value.(string); ok {
		return formatText(text)
	}
	return fmt.Sprintf("%v", value)
}

func runCase(name string, kinds []string, callCase func() []any) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("%s: panic: %s\n", name, formatText(fmt.Sprintf("%v", recovered)))
		}
	}()
	values := callCase()
	line := name + ":"
	for index, value := range values {
		line += " " + formatValue(value, kinds[index])
	}
	fmt.Println(line)
}

func main() {
`)
	for _, c := range cases {
		kinds := kindsLiteral(c.ResultKinds, "[]string{", "}")
		if len(c.ResultKinds) == 0 {
			fmt.Fprintf(&driver, "\trunCase(%q, %s, func() []any { fixture.%s(); return nil })\n", c.Name, kinds, c.Name)
			continue
		}
		names := make([]string, len(c.ResultKinds))
		for i := range names {
			names[i] = fmt.Sprintf("r%d", i)
		}
		fmt.Fprintf(&driver, "\trunCase(%q, %s, func() []any { %s := fixture.%s(); return []any{%s} })\n",
			c.Name, kinds, strings.Join(names, ", "), c.Name, strings.Join(names, ", "))
	}
	driver.WriteString("}\n")

	if err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte(driver.String()), 0o644); err != nil {
		return "", err
	}
	return runCommand(resolved.GoExecutable, []string{"run", "."}, workDir, env)
}

// runNodeDriver generates and executes the TypeScript-side driver under
// Node's native type stripping. Generated product output uses .js
// specifiers; a documented resolver shim maps them onto the .ts sources
// for execution — a host-execution normalization, never a mutation of
// generated output.
func runNodeDriver(nodeExecutable, workDir string, cases []fixtureCase) (string, error) {
	loader := LoaderSource
	register := `import { register } from "node:module";
register(new URL("./loader.mjs", import.meta.url));
`
	var driver strings.Builder
	driver.WriteString(`import * as fixture from "./generated/core/oracle.fixture/fixture/package.ts";
import { GoPanic } from "./generated/language-abi/gopanic.ts";

const floatView = new DataView(new ArrayBuffer(8));

// formatText mirrors the Go driver's rule exactly: ASCII-printable text
// prints directly, anything else as the byte-carrier's exact bytes.
function formatText(text: string): string {
  let printable = true;
  for (let i = 0; i < text.length; i++) {
    const unit = text.charCodeAt(i);
    if (unit < 0x20 || unit > 0x7e || unit === 0x22 || unit === 0x5c) {
      printable = false;
      break;
    }
  }
  if (printable) {
    return '"' + text + '"';
  }
  let out = "hex:";
  for (let i = 0; i < text.length; i++) {
    const unit = text.charCodeAt(i);
    if (unit > 0xff) {
      return "not-a-byte-string:" + JSON.stringify(text);
    }
    out += unit.toString(16).padStart(2, "0");
  }
  return out;
}

function formatValue(value: unknown, kind: string): string {
  if (kind === "float" && typeof value === "number") {
    floatView.setFloat64(0, value, false);
    const bits = floatView.getBigUint64(0, false).toString(16).padStart(16, "0");
    return "float64(0x" + bits + ")";
  }
  if (typeof value === "string") {
    return formatText(value);
  }
  return String(value);
}

function runCase(name: string, kinds: readonly string[], callCase: () => readonly unknown[]): void {
  let values: readonly unknown[];
  try {
    values = callCase();
  } catch (thrown) {
    if (thrown instanceof GoPanic) {
      console.log(name + ": panic: " + formatText(thrown.goMessage));
      return;
    }
    throw thrown;
  }
  let line = name + ":";
  for (let index = 0; index < values.length; index++) {
    line += " " + formatValue(values[index], kinds[index]!);
  }
  console.log(line);
}

`)
	for _, c := range cases {
		kinds := kindsLiteral(c.ResultKinds, "[", "]")
		switch len(c.ResultKinds) {
		case 0:
			fmt.Fprintf(&driver, "runCase(%q, %s, () => { fixture.%s(); return []; });\n", c.Name, kinds, c.Name)
		case 1:
			fmt.Fprintf(&driver, "runCase(%q, %s, () => [fixture.%s()]);\n", c.Name, kinds, c.Name)
		default:
			fmt.Fprintf(&driver, "runCase(%q, %s, () => fixture.%s());\n", c.Name, kinds, c.Name)
		}
	}

	files := map[string]string{
		"loader.mjs":   loader,
		"register.mjs": register,
		"driver.ts":    driver.String(),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(workDir, name), []byte(content), 0o644); err != nil {
			return "", err
		}
	}
	return runCommand(nodeExecutable, []string{"--no-warnings", "--import", "./register.mjs", "driver.ts"}, workDir, nil)
}
