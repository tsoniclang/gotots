// Package declparity extracts the DECLARED surface of generated
// TypeScript modules by PARSING them with the pinned TypeScript
// compiler — the independent side of the stage-05 declaration parity
// join. Nothing here consults the translator's internal identity
// machinery: the parsed structure is compared against the census's Go
// declaration shapes.
package declparity

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// TSDecl is one parsed top-level declaration of a generated module.
type TSDecl struct {
	Kind       string `json:"kind"` // function | class | type | let | const
	ParamCount int    `json:"paramCount,omitempty"`
	// ParamNames are the declared parameter names in order ("?" for a
	// non-identifier binding pattern) — the factory-protocol structure
	// check reads the zero$/eq$/clone$/set$/key$/rt$ groups from these.
	ParamNames []string `json:"paramNames,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Methods    []string `json:"methods,omitempty"`
	TypeParams int      `json:"typeParams,omitempty"`
	// TypeParamNames are the declared type-parameter names (bounds
	// excluded).
	TypeParamNames []string `json:"typeParamNames,omitempty"`
	Exported       bool     `json:"exported"`
}

// Extract parses every given generated file with the pinned TypeScript
// compiler and returns file → declared name → declaration structure.
func Extract(files map[string]string, typescriptModule string) (map[string]map[string]TSDecl, error) {
	if len(files) == 0 {
		return map[string]map[string]TSDecl{}, nil
	}
	staging, err := os.MkdirTemp("", "gotots-declparity-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)
	list := make([]string, 0, len(files))
	for path, content := range files {
		full := filepath.Join(staging, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return nil, err
		}
		list = append(list, path)
	}
	listPath := filepath.Join(staging, "files.json")
	listData, err := json.Marshal(list)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(listPath, listData, 0o644); err != nil {
		return nil, err
	}
	script := filepath.Join(staging, "extract.cjs")
	if err := os.WriteFile(script, []byte(extractorScript), 0o644); err != nil {
		return nil, err
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("node toolchain required for the declaration extractor: %w", err)
	}
	absTypescript, err := filepath.Abs(typescriptModule)
	if err != nil {
		return nil, err
	}
	command := exec.Command(node, script, absTypescript, staging, listPath)
	out, err := command.Output()
	if err != nil {
		detail := ""
		if exitError, ok := err.(*exec.ExitError); ok {
			detail = ": " + string(exitError.Stderr)
		}
		return nil, fmt.Errorf("declaration extractor failed: %w%s", err, detail)
	}
	var parsed map[string]map[string]TSDecl
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("declaration extractor output: %w", err)
	}
	return parsed, nil
}

// extractorScript parses each listed file with the pinned TypeScript
// compiler (syntax only — the strict typecheck is stage 04's job) and
// reports every top-level declaration's structure.
const extractorScript = `"use strict";
const fs = require("fs");
const path = require("path");
const ts = require(path.join(process.argv[2], "lib", "typescript.js"));
const staging = process.argv[3];
const files = JSON.parse(fs.readFileSync(process.argv[4], "utf8"));
const result = {};
for (const rel of files) {
  const text = fs.readFileSync(path.join(staging, rel), "utf8");
  const sf = ts.createSourceFile(rel, text, ts.ScriptTarget.Latest, false, ts.ScriptKind.TS);
  const decls = {};
  const isExported = (node) =>
    !!(ts.getCombinedModifierFlags(node) & ts.ModifierFlags.Export);
  for (const stmt of sf.statements) {
    if (ts.isFunctionDeclaration(stmt) && stmt.name) {
      decls[stmt.name.text] = {
        kind: "function",
        paramCount: stmt.parameters.length,
        paramNames: stmt.parameters.map((p) => (ts.isIdentifier(p.name) ? p.name.text : "?")),
        typeParams: stmt.typeParameters ? stmt.typeParameters.length : 0,
        typeParamNames: stmt.typeParameters ? stmt.typeParameters.map((tp) => tp.name.text) : [],
        exported: isExported(stmt),
      };
    } else if (ts.isClassDeclaration(stmt) && stmt.name) {
      const fields = [];
      const methods = [];
      for (const member of stmt.members) {
        if (ts.isPropertyDeclaration(member) && ts.isIdentifier(member.name)) {
          fields.push(member.name.text);
        } else if (ts.isMethodDeclaration(member) && ts.isIdentifier(member.name)) {
          methods.push(member.name.text);
        }
      }
      decls[stmt.name.text] = {
        kind: "class",
        fields,
        methods,
        typeParams: stmt.typeParameters ? stmt.typeParameters.length : 0,
        exported: isExported(stmt),
      };
    } else if (ts.isTypeAliasDeclaration(stmt)) {
      decls[stmt.name.text] = {
        kind: "type",
        typeParams: stmt.typeParameters ? stmt.typeParameters.length : 0,
        exported: isExported(stmt),
      };
    } else if (ts.isVariableStatement(stmt)) {
      const kind = stmt.declarationList.flags & ts.NodeFlags.Const ? "const" : "let";
      for (const decl of stmt.declarationList.declarations) {
        if (ts.isIdentifier(decl.name)) {
          decls[decl.name.text] = { kind, exported: isExported(stmt) };
        }
      }
    }
  }
  result[rel] = decls;
}
process.stdout.write(JSON.stringify(result));
`
