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
	// Ledger is the positive per-invocation disposition census: every
	// call, construction, member selection, and index in the verified
	// surface classified into exactly one closed class (an unclassifiable
	// site lands in Violations as undisposed-*, so Violations==0 AND a
	// non-empty ledger together certify every invocation positively).
	Ledger InvocationLedger `json:"ledger"`
	// Mechanisms maps each custom-mechanism identity to the generated
	// files whose ASTs reference its ABI symbols.
	Mechanisms map[string][]string `json:"mechanisms"`
	// Exports maps each generated file to its exported declaration
	// names — the authoritative join target for proof symbol evidence.
	Exports map[string][]string `json:"exports"`
}

// InvocationLedger is the positive-disposition census over the verified
// surface.
type InvocationLedger struct {
	DirectStatic          int `json:"directStatic"`
	TypedFunctionValue    int `json:"typedFunctionValue"`
	ConstructorStatic     int `json:"constructorStatic"`
	MemberSelection       int `json:"memberSelection"`
	TypedIndex            int `json:"typedIndex"`
	UnimplementedBlocking int `json:"unimplementedBlocking"`
}

// Invocations is the total number of positively disposed call sites.
func (l InvocationLedger) Invocations() int {
	return l.DirectStatic + l.TypedFunctionValue + l.ConstructorStatic + l.UnimplementedBlocking
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
const ledger = { directStatic: 0, typedFunctionValue: 0, constructorStatic: 0,
  memberSelection: 0, typedIndex: 0, unimplementedBlocking: 0 };
function skipParens(e) {
  while (ts.isParenthesizedExpression(e)) e = e.expression;
  return e;
}
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

// isLibGlobal reports whether a resolved symbol is the ambient lib global
// of the given name — recognized by RESOLVED DECLARATION ORIGIN, not
// spelling: its name matches AND it is declared OUTSIDE the generated tree
// (the TypeScript lib). An alias (const P = Proxy; type Fn = Function)
// resolves to the same lib symbol and is caught; a coincidental same-name
// declaration inside the generated tree resolves to a different symbol and
// is not. (Computed-member invocation stays syntactic — it is inherently
// syntactic, not an identity question.)
function isLibGlobal(symbol, name) {
  if (!symbol || symbol.getName() !== name) return false;
  const f = declaringFile(symbol);
  return !!f && !f.startsWith(root);
}

// resolvedTypeSymbol resolves a type reference's name to its origin symbol
// (following import/local aliases).
function resolvedTypeSymbol(typeNode) {
  if (!ts.isTypeReferenceNode(typeNode)) return undefined;
  const nameNode = ts.isQualifiedName(typeNode.typeName) ? typeNode.typeName.right : typeNode.typeName;
  return resolveSymbol(nameNode);
}

// isFunctionType reports whether a type node is the lib Function type,
// through any alias form: it resolves the TYPE semantically (following
// import and type aliases) and compares the resolved type's symbol origin
// to the lib Function — so type Fn = Function; x: Fn is caught.
function isFunctionType(typeNode) {
  if (!typeNode || !ts.isTypeNode(typeNode)) return false;
  const t = checker.getTypeFromTypeNode(typeNode);
  if (!t) return false;
  return isLibGlobal(t.getSymbol(), "Function") || isLibGlobal(t.aliasSymbol, "Function");
}

// propertyDeclaredInBox reports whether a resolved property SYMBOL was
// declared by the GoBox (or GoAnyBox) declaration. This is the ONE
// authoritative, form-agnostic carrier test: the checker resolves the v
// property semantically through EVERY generated type form — parentheses,
// intersections, unions, imported and multi-hop aliases — so a box hidden
// behind any of them is identified by the ORIGIN of its own v member, not
// by matching alias syntax or member-name shape.
function propertyDeclaredInBox(propSymbol) {
  if (!propSymbol || !propSymbol.declarations) return false;
  for (const d of propSymbol.declarations) {
    let node = d.parent;
    while (node) {
      if (ts.isTypeAliasDeclaration(node) || ts.isInterfaceDeclaration(node)) {
        const declSym = node.name ? checker.getSymbolAtLocation(node.name) : undefined;
        return declSym === goBoxDeclSymbol || declSym === goAnyBoxDeclSymbol;
      }
      node = node.parent;
    }
  }
  return false;
}

// isBoxType reports whether a RESOLVED type is a box — its v member
// originates in the GoBox declaration — regardless of the type form it
// reached us through.
function isBoxType(t) {
  if (!t) return false;
  return propertyDeclaredInBox(checker.getPropertyOfType(t, "v"));
}

// boxPayloadType returns the payload (v) type of a box referenced by
// typeNode, or undefined if it is not a box. Identity is by the v
// member's declaring origin, so an ordinary {k,r,v,m} domain type is not a
// box and a parenthesized / intersected / multi-hop-aliased box still is.
function boxPayloadType(typeNode) {
  const t = checker.getTypeFromTypeNode(typeNode);
  const v = checker.getPropertyOfType(t, "v");
  if (!propertyDeclaredInBox(v)) return undefined;
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
        // callable. A callee with no call signature that is any/unknown (by
        // FLAGS, not rendered text) or the lib Function type (by symbol
        // identity) is erased dispatch. A Function-annotated callee is also
        // caught at its type annotation (erased-function-type).
        const calleeType = checker.getTypeAtLocation(callee);
        const nonNull = calleeType.getNonNullableType();
        if (nonNull.getCallSignatures().length === 0 && !(nonNull.getFlags() & ts.TypeFlags.Never)) {
          const f = nonNull.getFlags();
          const isFnGlobal = isLibGlobal(nonNull.getSymbol(), "Function") || isLibGlobal(nonNull.aliasSymbol, "Function");
          if ((f & ts.TypeFlags.Any) || (f & ts.TypeFlags.Unknown) || isFnGlobal) {
            report(file, node, source, "erased-callee-type");
          }
        }
      }
      // POSITIVE per-invocation disposition: every call lands in exactly
      // one closed class, or is reported undisposed (fail-closed).
      {
        const inner = skipParens(callee);
        const dispositionName = nameNode ? resolveSymbol(nameNode) : undefined;
        const resolvedName = dispositionName ? dispositionName.getName() : "";
        const calleeType = checker.getTypeAtLocation(callee).getNonNullableType();
        if (resolvedName === "goBodyUnimplemented" || resolvedName === "goExternalUnimplemented") {
          ledger.unimplementedBlocking++;
        } else if (ts.isArrowFunction(inner) || ts.isFunctionExpression(inner)) {
          ledger.directStatic++;
        } else if (calleeType.getCallSignatures().length > 0) {
          const decl = dispositionName && dispositionName.valueDeclaration;
          if (decl && (ts.isFunctionDeclaration(decl) || ts.isMethodDeclaration(decl))) {
            ledger.directStatic++;
          } else {
            // A typed function VALUE (parameter, field, local, vtable
            // member): the target type is exact and closed even though the
            // runtime target is dynamic.
            ledger.typedFunctionValue++;
          }
        } else if (calleeType.getFlags() & ts.TypeFlags.Never) {
          // A call in provably unreachable code: statically disposed.
          ledger.directStatic++;
        } else {
          report(file, node, source, "undisposed-invocation");
        }
      }
    }
    if (ts.isNewExpression(node)) {
      const t = checker.getTypeAtLocation(node.expression).getNonNullableType();
      if (checker.getSignaturesOfType(t, ts.SignatureKind.Construct).length > 0) {
        ledger.constructorStatic++;
      } else {
        report(file, node, source, "undisposed-construction");
      }
    }
    if (ts.isPropertyAccessExpression(node)) {
      // Every member selection resolves to a declared symbol — never a
      // name-computed lookup.
      if (checker.getSymbolAtLocation(node.name)) {
        ledger.memberSelection++;
      } else {
        report(file, node, source, "unresolved-member-selection");
      }
    }
    if (ts.isElementAccessExpression(node) && !ts.isCallExpression(node.parent)) {
      ledger.typedIndex++;
    }
    if (ts.isNewExpression(node) && ts.isIdentifier(node.expression)) {
      const s = resolveSymbol(node.expression);
      if (isLibGlobal(s, "Proxy") || isLibGlobal(s, "Function")) {
        report(file, node, source, "reflection-construct");
      }
    }
    if (ts.isPropertyAccessExpression(node) && ts.isIdentifier(node.expression)) {
      const s = resolveSymbol(node.expression);
      if (isLibGlobal(s, "Reflect")) {
        report(file, node, source, "reflection-construct");
      }
    }
    if (isFunctionType(node)) {
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
        if (isBoxType(boxType)) {
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
    if (ts.isTypeReferenceNode(node) && node.typeArguments && node.typeArguments.length === 2) {
      // A string-keyed function registry (Record<string, Function> or
      // Map<string, Function>) — identified by the lib Record/Map symbol and
      // the lib Function symbol, so an aliased Record/Map/Function is caught.
      const container = resolvedTypeSymbol(node);
      if (isLibGlobal(container, "Record") || isLibGlobal(container, "Map")) {
        const [k, v] = node.typeArguments;
        const kt = checker.getTypeFromTypeNode(k);
        if ((kt.getFlags() & ts.TypeFlags.String) && isFunctionType(v)) {
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
process.stdout.write(JSON.stringify({ violations, mechanisms, exports: exportedNames, ledger }));
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
