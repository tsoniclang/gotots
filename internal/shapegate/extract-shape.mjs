// Typed-AST shape extractor: parses TypeScript modules with the PINNED
// compiler and emits per-file shape facts as JSON. This is the ONLY
// authority for generated-shape verdicts; the regex detectors in this
// package are diagnostics. Invoked by the Go gate:
//
//   node extract-shape.mjs <pinned-typescript-dir> <file.ts>...
//
// Output: [{file, declarations:[{kind,name,exported,params}],
//           calls:[{callee,args,line}], aliases:[{name}]}]
import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
import { argv, stdout, exit } from "node:process";

const [tsDir, ...files] = argv.slice(2);
if (!tsDir || files.length === 0) {
  console.error("usage: node extract-shape.mjs <typescript-dir> <file.ts>...");
  exit(1);
}
const require = createRequire(import.meta.url);
const ts = require(tsDir);

function calleeText(expr, source) {
  return expr.getText(source);
}

const out = [];
for (const file of files) {
  const text = readFileSync(file, "utf8");
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.ES2022, true);
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
      facts.calls.push({
        callee: calleeText(node.expression, source),
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
