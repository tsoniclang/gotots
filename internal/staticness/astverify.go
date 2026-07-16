// Typed-AST staticness verification: the generated output is parsed
// with the pinned TypeScript compiler and its AST walked structurally,
// so aliases, multiline spellings, and renamed equivalents of erased
// dispatch cannot evade detection the way text scanning permits. The
// same walk derives which custom ABI mechanisms each file actually uses
// (by resolved identifier, not substring), feeding the necessity-record
// join. The text sweep remains a fast pre-check; the gate's verdict is
// the AST result.
package staticness

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ASTReport is the verifier's machine result.
type ASTReport struct {
	Violations []Violation `json:"violations"`
	// Mechanisms maps each custom-mechanism identity to the generated
	// files whose ASTs reference its ABI symbols.
	Mechanisms map[string][]string `json:"mechanisms"`
}

// verifierScript walks every parsed source file. Prohibited structural
// shapes: calls through computed members, erased Function/unknown[]
// callable types, string-keyed function registries, reflection and
// dynamic code, and the deleted erased-dispatch helpers by identifier.
const verifierScript = `
const ts = require(process.argv[2]);
const fs = require("fs");
const path = require("path");
const root = process.argv[3];

const files = [];
(function collect(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) collect(full);
    else if (entry.name.endsWith(".ts")) files.push(full);
  }
})(root);

const violations = [];
const mechanisms = {};
const mechanismSymbols = {
  "slice-carrier": ["GoSlice", "goSliceFrom", "goSliceMake", "goSliceMakeStruct", "goNativeMakeLen", "goNativeMakeCap"],
  "pointer-cell": ["GoCell"],
  "interface-box": ["goIfaceBox", "GoIface", "GoIfaceBox", "goRttiComposite"],
  "keyed-map": ["GoKeyedMap", "goKMapGet", "goKMapSet", "goKMapValues"],
};
const bannedIdentifiers = new Set([
  "goIfaceCall", "goFuncInvoke", "goExternalCall", "goExternalRegister", "eval",
]);

function report(file, node, source, pattern) {
  const { line } = source.getLineAndCharacterOfPosition(node.getStart(source));
  violations.push({
    file: path.relative(root, file).split(path.sep).join("/"),
    line: line + 1,
    pattern,
    detail: node.getText(source).slice(0, 160),
  });
}

function isFunctionTypeName(node) {
  return ts.isTypeReferenceNode(node) && ts.isIdentifier(node.typeName) && node.typeName.text === "Function";
}

for (const file of files) {
  const text = fs.readFileSync(file, "utf8");
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.ES2022, true);
  const relative = path.relative(root, file).split(path.sep).join("/");
  const seenMechanisms = new Set();
  (function walk(node) {
    // Call through a computed member: obj[expr](...) or obj[expr].call/.apply.
    if (ts.isCallExpression(node)) {
      const callee = node.expression;
      if (ts.isElementAccessExpression(callee)) {
        report(file, node, source, "computed-member-call");
      }
      if (
        ts.isPropertyAccessExpression(callee) &&
        (callee.name.text === "call" || callee.name.text === "apply") &&
        ts.isElementAccessExpression(callee.expression)
      ) {
        report(file, node, source, "computed-member-call");
      }
      if (ts.isIdentifier(callee) && bannedIdentifiers.has(callee.text)) {
        report(file, node, source, "erased-dispatch-helper");
      }
      if (ts.isPropertyAccessExpression(callee) && bannedIdentifiers.has(callee.name.text)) {
        report(file, node, source, "erased-dispatch-helper");
      }
    }
    // new Proxy(...) / new Function(...).
    if (ts.isNewExpression(node) && ts.isIdentifier(node.expression)) {
      if (node.expression.text === "Proxy" || node.expression.text === "Function") {
        report(file, node, source, "reflection-construct");
      }
    }
    // Reflect.* usage.
    if (ts.isPropertyAccessExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === "Reflect") {
      report(file, node, source, "reflection-construct");
    }
    // Erased Function type in any type position.
    if (isFunctionTypeName(node)) {
      report(file, node, source, "erased-function-type");
    }
    // Record<string, Function> / Map<string, Function> registries.
    if (ts.isTypeReferenceNode(node) && ts.isIdentifier(node.typeName)) {
      const name = node.typeName.text;
      if ((name === "Record" || name === "Map") && node.typeArguments && node.typeArguments.length === 2) {
        const [k, v] = node.typeArguments;
        if (k.kind === ts.SyntaxKind.StringKeyword && isFunctionTypeName(v)) {
          report(file, node, source, "string-function-registry");
        }
      }
    }
    // Mechanism identifiers (typed usage evidence for necessity records).
    if (ts.isIdentifier(node)) {
      for (const [mechanism, symbols] of Object.entries(mechanismSymbols)) {
        if (symbols.includes(node.text)) seenMechanisms.add(mechanism);
      }
    }
    ts.forEachChild(node, walk);
  })(source);
  for (const mechanism of seenMechanisms) {
    (mechanisms[mechanism] ??= []).push(relative);
  }
}
for (const key of Object.keys(mechanisms)) mechanisms[key].sort();
process.stdout.write(JSON.stringify({ violations, mechanisms }));
`

// VerifyAST writes the generated files to a staging directory, parses
// each with the pinned TypeScript compiler, and returns the structural
// verdicts. typescriptModule is the pinned typescript package directory.
func VerifyAST(files map[string]string, typescriptModule string) (*ASTReport, error) {
	staging, err := os.MkdirTemp("", "gotots-staticness-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)
	for path, content := range files {
		full := filepath.Join(staging, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return nil, err
		}
	}
	script := filepath.Join(staging, "verify.cjs")
	if err := os.WriteFile(script, []byte(verifierScript), 0o644); err != nil {
		return nil, err
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("node toolchain required for the AST staticness verifier: %w", err)
	}
	absTypescript, err := filepath.Abs(typescriptModule)
	if err != nil {
		return nil, err
	}
	command := exec.Command(node, script, absTypescript, staging)
	out, err := command.Output()
	if err != nil {
		detail := ""
		if exit, ok := err.(*exec.ExitError); ok {
			detail = string(exit.Stderr)
		}
		return nil, fmt.Errorf("AST staticness verifier: %w\n%s", err, detail)
	}
	var report ASTReport
	if err := json.Unmarshal(out, &report); err != nil {
		return nil, fmt.Errorf("AST staticness verifier output: %w", err)
	}
	// The verifier's own script file is not generated output.
	filtered := report.Violations[:0]
	for _, v := range report.Violations {
		if v.File != "verify.cjs" {
			filtered = append(filtered, v)
		}
	}
	report.Violations = filtered
	return &report, nil
}
