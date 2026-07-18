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
	// Exports maps each generated file to its exported declaration
	// names — the authoritative join target for proof symbol evidence.
	Exports map[string][]string `json:"exports"`
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
const exportedNames = {};
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
  const relative = path.relative(root, file).split(path.sep).join("/");
  violations.push({
    file: relative,
    line: line + 1,
    pattern: relative.startsWith("language-abi/") ? "abi:" + pattern : pattern,
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

// isErasedType reports whether a RESOLVED type is an erased top: any,
// unknown, the object keyword (NonPrimitive), or the empty object type {}
// (an object type with no properties, no call/construct signatures, and no
// index signatures). A concrete object type (with members) and a bare type
// parameter are NOT erased.
function isErasedType(t) {
  if (!t) return false;
  const f = t.getFlags();
  if (f & ts.TypeFlags.Any || f & ts.TypeFlags.Unknown || f & ts.TypeFlags.NonPrimitive) return true;
  if ((f & ts.TypeFlags.Object) &&
      t.getProperties().length === 0 &&
      checker.getSignaturesOfType(t, ts.SignatureKind.Call).length === 0 &&
      checker.getSignaturesOfType(t, ts.SignatureKind.Construct).length === 0 &&
      checker.getIndexInfosOfType(t).length === 0) {
    return true; // {}
  }
  return false;
}

// The GoBox / GoAnyBox DECLARATION symbols, resolved once from the ABI
// module. A box is identified by DECLARATION IDENTITY (its resolved type
// or alias chain reaches THIS declaration), never by member-name shape, so
// an ordinary domain type that coincidentally has k/r/v/m members is not a
// box, while an alias whose target is GoBox (Wrapped<V> = GoBox<"x",V,M>)
// still is.
let goBoxDeclSymbol, goAnyBoxDeclSymbol;
for (const source of program.getSourceFiles()) {
  const rel = path.relative(root, source.fileName).split(path.sep).join("/");
  if (rel !== "language-abi/goiface.ts") continue;
  for (const stmt of source.statements) {
    if ((ts.isTypeAliasDeclaration(stmt) || ts.isInterfaceDeclaration(stmt)) && stmt.name) {
      const sym = checker.getSymbolAtLocation(stmt.name);
      if (stmt.name.text === "GoBox") goBoxDeclSymbol = sym;
      else if (stmt.name.text === "GoAnyBox") goAnyBoxDeclSymbol = sym;
    }
  }
}

// symbolReachesBoxDecl follows an alias/type-alias chain from sym to the
// GoBox or GoAnyBox declaration symbol (through import aliases and
// type-alias targets), so a box hidden behind one or more aliases is
// still identified by its resolved declaration — not by field names.
function symbolReachesBoxDecl(sym) {
  let s = sym;
  const seen = new Set();
  while (s && !seen.has(s)) {
    seen.add(s);
    if (s === goBoxDeclSymbol || s === goAnyBoxDeclSymbol) return true;
    if (s.flags & ts.SymbolFlags.Alias) { s = checker.getAliasedSymbol(s); continue; }
    const decl = s.declarations && s.declarations[0];
    if (decl && ts.isTypeAliasDeclaration(decl) && decl.type && ts.isTypeReferenceNode(decl.type)) {
      const nameNode = ts.isQualifiedName(decl.type.typeName) ? decl.type.typeName.right : decl.type.typeName;
      s = checker.getSymbolAtLocation(nameNode);
      continue;
    }
    break;
  }
  return false;
}

// isGoBoxType reports whether a RESOLVED type is a GoBox (or an alias
// resolving to one), by DECLARATION identity of its alias/own symbol.
function isGoBoxType(t) {
  if (!t) return false;
  if (t.aliasSymbol && symbolReachesBoxDecl(t.aliasSymbol)) return true;
  if (t.symbol && symbolReachesBoxDecl(t.symbol)) return true;
  return false;
}

// boxPayloadType returns the payload (v) type of a box referenced by
// typeNode, or undefined if it is not a box. Identity is by resolved
// DECLARATION, so an ordinary {k,r,v,m} domain type is not a box.
function boxPayloadType(typeNode) {
  const t = checker.getTypeFromTypeNode(typeNode);
  if (!isGoBoxType(t)) return undefined;
  const v = checker.getPropertyOfType(t, "v");
  if (!v) return undefined;
  // getTypeOfSymbol returns the INSTANTIATED payload type (unknown/object),
  // where getTypeOfSymbolAtLocation would re-resolve to the parameter V.
  return checker.getTypeOfSymbol(v);
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
    // Erased interface payload in generated CORE: a box (GoBox, OR any
    // alias resolving to one) whose payload resolves to an erased top, or
    // an "as" cast recovering from a .v property — each recovers a payload
    // from an erased type (spec 06, ADR-0004). Detection is by RESOLVED
    // type, so an alias such as Wrapped-of-object (type Wrapped V = GoBox
    // of x,V,M) is caught even though its surface name is not GoBox.
    {
      if (ts.isTypeReferenceNode(node)) {
        const payload = boxPayloadType(node);
        if (payload && isErasedType(payload)) {
          report(file, node, source, "erased-iface-payload");
        }
      }
      if (ts.isAsExpression(node) && ts.isPropertyAccessExpression(node.expression) &&
          node.expression.name.text === "v") {
        // Recovering .v off a box: identify the box by RESOLVED declaration
        // identity, never by rendered type text containing "GoBox".
        const boxType = checker.getTypeAtLocation(node.expression.expression);
        if (isGoBoxType(boxType)) {
          report(file, node, source, "erased-payload-recovery-cast");
        }
      }
      // A reference to the erased GoAnyBox alias in generated core: resolve
      // the name to its declaration symbol and compare to the ABI's
      // GoAnyBox declaration, not the spelling.
      if (ts.isQualifiedName(node) && goAnyBoxDeclSymbol) {
        let sym = checker.getSymbolAtLocation(node.right);
        while (sym && (sym.flags & ts.SymbolFlags.Alias)) sym = checker.getAliasedSymbol(sym);
        if (sym === goAnyBoxDeclSymbol) {
          report(file, node, source, "erased-anybox-in-core");
        }
      }
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
  // The file's exported value/type declaration names: the authoritative
  // join target for proof generated-symbol evidence.
  const names = [];
  for (const stmt of source.statements) {
    const mods = ts.canHaveModifiers(stmt) ? ts.getModifiers(stmt) : undefined;
    const exported = mods && mods.some((m) => m.kind === ts.SyntaxKind.ExportKeyword);
    if (!exported) continue;
    if (ts.isFunctionDeclaration(stmt) && stmt.name) names.push(stmt.name.text);
    else if (ts.isClassDeclaration(stmt) && stmt.name) names.push(stmt.name.text);
    else if (ts.isTypeAliasDeclaration(stmt)) names.push(stmt.name.text);
    else if (ts.isVariableStatement(stmt)) {
      for (const d of stmt.declarationList.declarations) {
        if (ts.isIdentifier(d.name)) names.push(d.name.text);
      }
    }
  }
  exportedNames[relative] = names.sort();
}
for (const key of Object.keys(mechanisms)) mechanisms[key].sort();
process.stdout.write(JSON.stringify({ violations, mechanisms, exports: exportedNames }));
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
