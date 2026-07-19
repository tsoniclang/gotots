package declparity

import (
	"path/filepath"
	"testing"
)

func TestExtractParsesDeclarationSurface(t *testing.T) {
	files := map[string]string{
		"core/x/package.ts": `import * as gort$ from "../abi.js";
export function Node$Pos($r: Node | undefined, extra: number): number { return extra; }
export function Generic<T>(v: T, zero$T: () => T, eq$T: (a: T, b: T) => boolean, clone$T: (v: T) => T, set$T: ((d: T, s: T) => void) | undefined): T { return v; }
export class Node {
  Kind: number;
  Flags: number;
  constructor(Kind: number, Flags: number) { this.Kind = Kind; this.Flags = Flags; }
  goClone$(): Node { return new Node(this.Kind, this.Flags); }
}
export type Path = string;
export type Cache<T> = Map<string, T>;
export let version: string = "";
export const answer = 42;
`,
	}
	typescriptModule := filepath.Join("..", "..", "product", "node_modules", "typescript")
	parsed, err := Extract(files, typescriptModule)
	if err != nil {
		t.Fatal(err)
	}
	decls := parsed["core/x/package.ts"]
	if decls == nil {
		t.Fatal("file missing from extraction")
	}
	if d := decls["Node$Pos"]; d.Kind != "function" || d.ParamCount != 2 || d.TypeParams != 0 {
		t.Fatalf("Node$Pos: %+v", d)
	}
	if d := decls["Generic"]; d.Kind != "function" || d.ParamCount != 5 || d.TypeParams != 1 {
		t.Fatalf("Generic: %+v", d)
	}
	d := decls["Node"]
	if d.Kind != "class" || len(d.Fields) != 2 || d.Fields[0] != "Kind" || len(d.Methods) != 1 {
		t.Fatalf("Node: %+v", d)
	}
	if d := decls["Path"]; d.Kind != "type" || d.TypeParams != 0 {
		t.Fatalf("Path: %+v", d)
	}
	if d := decls["Cache"]; d.Kind != "type" || d.TypeParams != 1 {
		t.Fatalf("Cache: %+v", d)
	}
	if d := decls["version"]; d.Kind != "let" {
		t.Fatalf("version: %+v", d)
	}
	if d := decls["answer"]; d.Kind != "const" {
		t.Fatalf("answer: %+v", d)
	}
}
