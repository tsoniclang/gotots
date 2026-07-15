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
	"fmt"
	"math"
	"strconv"

	"oracle.fixture/fixture"
)

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
		return strconv.Quote(text)
	}
	return fmt.Sprintf("%v", value)
}

func runCase(name string, kinds []string, callCase func() []any) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("%s: panic: %v\n", name, recovered)
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
	loader := `import { pathToFileURL } from "node:url";
import { access } from "node:fs/promises";

export async function resolve(specifier, context, nextResolve) {
  if (specifier.startsWith(".") && specifier.endsWith(".js")) {
    const parent = context.parentURL ? new URL(context.parentURL) : pathToFileURL(process.cwd() + "/");
    const candidate = new URL(specifier.slice(0, -3) + ".ts", parent);
    try {
      await access(candidate);
      return nextResolve(candidate.href, context);
    } catch {}
  }
  return nextResolve(specifier, context);
}
`
	register := `import { register } from "node:module";
register(new URL("./loader.mjs", import.meta.url));
`
	var driver strings.Builder
	driver.WriteString(`import * as fixture from "./generated/core/oracle.fixture/fixture/package.ts";
import { GoPanic } from "./generated/language-abi/gopanic.ts";

const floatView = new DataView(new ArrayBuffer(8));

function formatValue(value: unknown, kind: string): string {
  if (kind === "float" && typeof value === "number") {
    floatView.setFloat64(0, value, false);
    const bits = floatView.getBigUint64(0, false).toString(16).padStart(16, "0");
    return "float64(0x" + bits + ")";
  }
  if (typeof value === "string") {
    return JSON.stringify(value);
  }
  return String(value);
}

function runCase(name: string, kinds: readonly string[], callCase: () => readonly unknown[]): void {
  let values: readonly unknown[];
  try {
    values = callCase();
  } catch (thrown) {
    if (thrown instanceof GoPanic) {
      console.log(name + ": panic: " + thrown.goMessage);
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
