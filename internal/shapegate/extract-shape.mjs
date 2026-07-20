// Typed-AST shape extractor: parses TypeScript modules through the
// PINNED compiler as a real Program with a TypeChecker, and reports
// calls by RESOLVED declaration identity — an alias such as
// `const f = maps.Keys; f(...)` resolves to Keys and cannot evade the
// gate. This is the ONLY authority for generated-shape verdicts; the
// regex detectors in this package are diagnostics. Invoked by Go:
//
//   node extract-shape.mjs <pinned-typescript-dir> <file.ts>...
import { createRequire } from "node:module";
import { argv, stdout, exit } from "node:process";

const [tsDir, ...files] = argv.slice(2);
if (!tsDir || files.length === 0) {
  console.error("usage: node extract-shape.mjs <typescript-dir> <file.ts>...");
  exit(1);
}
const require = createRequire(import.meta.url);
const ts = require(tsDir);

const options = {
  target: ts.ScriptTarget.ES2022,
  module: ts.ModuleKind.ESNext,
  strict: true,
  noResolve: true,
  skipLibCheck: true,
};
const program = ts.createProgram(files, options);
const checker = program.getTypeChecker();

// resolveCallee follows the callee to its ORIGINAL declaration:
// import aliases through getAliasedSymbol, and const-bound function
// references (`const f = maps.Keys`) through their initializers, to a
// bounded depth so cycles cannot hang the gate.
function resolveCallee(expr) {
  let node = expr;
  for (let depth = 0; depth < 16; depth++) {
    let symbol = checker.getSymbolAtLocation(node);
    if (!symbol) {
      return undefined;
    }
    if (symbol.flags & ts.SymbolFlags.Alias) {
      symbol = checker.getAliasedSymbol(symbol);
    }
    const declaration = symbol.valueDeclaration ?? symbol.declarations?.[0];
    if (!declaration) {
      return { name: symbol.getName(), file: "", line: 0 };
    }
    if (ts.isVariableDeclaration(declaration) && declaration.initializer &&
      (ts.isIdentifier(declaration.initializer) || ts.isPropertyAccessExpression(declaration.initializer))) {
      node = ts.isPropertyAccessExpression(declaration.initializer)
        ? declaration.initializer.name
        : declaration.initializer;
      continue;
    }
    const source = declaration.getSourceFile();
    return {
      name: symbol.getName(),
      file: source.fileName,
      line: source.getLineAndCharacterOfPosition(declaration.getStart(source)).line + 1,
    };
  }
  return undefined;
}

const out = [];
for (const file of files) {
  const source = program.getSourceFile(file);
  if (!source) {
    console.error(`no source file for ${file}`);
    exit(1);
  }
  const facts = { file, declarations: [], calls: [], aliases: [] };
  const isExported = (node) =>
    (ts.getCombinedModifierFlags(node) & ts.ModifierFlags.Export) !== 0;
  const visit = (node) => {
    if (ts.isFunctionDeclaration(node) && node.name) {
      facts.declarations.push({
        kind: "function",
        name: node.name.text,
        exported: isExported(node),
        params: node.parameters.length,
      });
    } else if (ts.isClassDeclaration(node) && node.name) {
      facts.declarations.push({
        kind: "class",
        name: node.name.text,
        exported: isExported(node),
        params: 0,
      });
      for (const member of node.members) {
        if (ts.isMethodDeclaration(member) && member.name && ts.isIdentifier(member.name)) {
          facts.declarations.push({
            kind: "method",
            name: node.name.text + "." + member.name.text,
            exported: isExported(node),
            params: member.parameters.length,
          });
        }
      }
    } else if (ts.isTypeAliasDeclaration(node)) {
      facts.aliases.push({ name: node.name.text, exported: isExported(node) });
    } else if (ts.isCallExpression(node)) {
      const target = ts.isPropertyAccessExpression(node.expression)
        ? node.expression.name
        : node.expression;
      const resolved = resolveCallee(target);
      facts.calls.push({
        callee: node.expression.getText(source),
        resolvedName: resolved ? resolved.name : "",
        resolvedFile: resolved ? resolved.file : "",
        resolvedLine: resolved ? resolved.line : 0,
        args: node.arguments.length,
        line: source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1,
      });
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  out.push(facts);
}
stdout.write(JSON.stringify(out));
