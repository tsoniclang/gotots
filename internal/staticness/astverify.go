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

// A real whole-program verifier: the pinned compiler builds a Program,
// its TypeChecker resolves every symbol, aliases are followed via
// getAliasedSymbol, and every invocation must receive a positive
// disposition — an unresolvable or erased-typed callee is a violation.
const program = ts.createProgram(files, {
  target: ts.ScriptTarget.ES2022,
  module: ts.ModuleKind.ESNext,
  moduleResolution: ts.ModuleResolutionKind.Bundler,
  strict: true,
  noEmit: true,
  allowImportingTsExtensions: true,
});
const checker = program.getTypeChecker();

const violations = [];
const mechanisms = {};
const mechanismByFile = {
  "goslice.ts": "slice-carrier",
  "goiface.ts": "interface-box",
};
const mechanismBySymbol = {
  "GoCell": "pointer-cell",
  "goKeyed": "keyed-map",
  "GoKeyedMap": "keyed-map",
  "goKMapGet": "keyed-map",
  "goKMapSet": "keyed-map",
  "goKMapValues": "keyed-map",
};
const bannedNames = new Set([
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

// resolve follows import and local aliases to the original symbol.
function resolveSymbol(node) {
  let symbol = checker.getSymbolAtLocation(node);
  if (!symbol) return undefined;
  while (symbol.flags & ts.SymbolFlags.Alias) {
    const next = checker.getAliasedSymbol(symbol);
    if (!next || next === symbol) break;
    symbol = next;
  }
  // A const initialized from another symbol: follow the initializer.
  const decl = symbol.valueDeclaration;
  if (decl && ts.isVariableDeclaration(decl) && decl.initializer &&
      (ts.isIdentifier(decl.initializer) || ts.isPropertyAccessExpression(decl.initializer))) {
    const target = resolveSymbol(ts.isIdentifier(decl.initializer) ? decl.initializer : decl.initializer.name);
    if (target) return target;
  }
  return symbol;
}

function declaringFile(symbol) {
  const decls = symbol.getDeclarations();
  if (!decls || decls.length === 0) return undefined;
  return decls[0].getSourceFile().fileName;
}

function isFunctionTypeName(node) {
  return ts.isTypeReferenceNode(node) && ts.isIdentifier(node.typeName) && node.typeName.text === "Function";
}

for (const source of program.getSourceFiles()) {
  const file = source.fileName;
  if (!file.startsWith(root) || file.endsWith(".d.ts")) continue;
  if (file.endsWith("verify.cjs")) continue;
  const relative = path.relative(root, file).split(path.sep).join("/");
  const seenMechanisms = new Set();
  (function walk(node) {
    if (ts.isCallExpression(node)) {
      const callee = node.expression;
      // Computed-member invocation is name-selected dispatch.
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
      // Symbol-resolved ban: follows aliases, so ` + "`" + `const g = goIfaceCall;
      // g(...)` + "`" + ` is caught by the resolved symbol's name.
      const nameNode = ts.isPropertyAccessExpression(callee) ? callee.name
        : ts.isIdentifier(callee) ? callee : undefined;
      if (nameNode) {
        const symbol = resolveSymbol(nameNode);
        if (symbol && bannedNames.has(symbol.getName())) {
          report(file, node, source, "erased-dispatch-helper");
        }
        // Positive disposition: the callee's type must be a concrete
        // callable — any/unknown/Function-typed callees are erased.
        const calleeType = checker.getTypeAtLocation(callee);
        const nonNull = calleeType.getNonNullableType();
        if (nonNull.getCallSignatures().length === 0 && !(nonNull.getFlags() & ts.TypeFlags.Never)) {
          const display = checker.typeToString(nonNull);
          if (display === "any" || display === "unknown" || display === "Function") {
            report(file, node, source, "erased-callee-type");
          }
        }
      }
    }
    if (ts.isNewExpression(node) && ts.isIdentifier(node.expression)) {
      if (node.expression.text === "Proxy" || node.expression.text === "Function") {
        report(file, node, source, "reflection-construct");
      }
    }
    if (ts.isPropertyAccessExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === "Reflect") {
      report(file, node, source, "reflection-construct");
    }
    if (isFunctionTypeName(node)) {
      report(file, node, source, "erased-function-type");
    }
    if (ts.isTypeReferenceNode(node) && ts.isIdentifier(node.typeName)) {
      const name = node.typeName.text;
      if ((name === "Record" || name === "Map") && node.typeArguments && node.typeArguments.length === 2) {
        const [k, v] = node.typeArguments;
        if (k.kind === ts.SyntaxKind.StringKeyword && isFunctionTypeName(v)) {
          report(file, node, source, "string-function-registry");
        }
      }
    }
    // Mechanism detection by RESOLVED symbol: a use site counts only
    // when the symbol's declaring file is the ABI module — a local
    // declaration named like an ABI symbol never counts, and an
    // imported alias always does. The ABI's own definitions are not
    // use sites.
    if (ts.isIdentifier(node)) {
      const symbol = resolveSymbol(node);
      if (symbol) {
        const declared = declaringFile(symbol);
        if (declared && declared !== file && declared.startsWith(root)) {
          const base = path.basename(declared);
          const byFile = mechanismByFile[base];
          if (byFile) seenMechanisms.add(byFile);
          const bySymbol = mechanismBySymbol[symbol.getName()];
          if (bySymbol && base.startsWith("go")) seenMechanisms.add(bySymbol);
        }
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
