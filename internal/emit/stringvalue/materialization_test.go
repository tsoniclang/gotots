package stringvalue_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestPointerStringMaterializationRetainsExactIndexDomains(t *testing.T) {
	emission := compileStringFixture(t)
	directory := t.TempDir()
	paths, _, printed := materializeStringProgram(t, directory, emission)
	for _, required := range []string{
		"Number.isSafeInteger(numericStart)",
		"Number.isSafeInteger(numericLength)",
		"Number.isSafeInteger(numericStart + numericLength)",
		"for (let index = 0; index < numericLength; index++)",
		"for (let index = 0n; index < length; index++)",
		"chunk.length === 4096",
		"String.fromCharCode(...chunk)",
		"chunk.length = 0",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("generated materialization lacks %q", required)
		}
	}
	valueFile := targetFile(t, emission, emit.TargetFileSupport, "runtime/string-value.ts")
	var body tsgo.Block
	for _, statement := range valueFile.Statements() {
		class, ok := statement.(tsgo.ClassDeclaration)
		if !ok || class.Name().Text() != "GoStringPointerBacking" {
			continue
		}
		for _, member := range class.Members() {
			method, ok := member.(tsgo.MethodDeclaration)
			if ok && method.Name().(tsgo.Identifier).Text() == "text" {
				body = method.Body().(tsgo.Block)
			}
		}
	}
	if body == nil || !hasNumericMaterializationLoop(body) {
		t.Fatal("materialization has no guarded numeric AST loop")
	}
	factory := tsgo.NewFactory()
	if hasNumericMaterializationLoop(factory.Block(nil, true)) {
		t.Fatal("missing numeric-loop control passed")
	}
	withoutNumericBranch := make([]tsgo.Statement, 0, len(body.Statements()))
	for _, statement := range body.Statements() {
		if _, conditional := statement.(tsgo.IfStatement); !conditional {
			withoutNumericBranch = append(withoutNumericBranch, statement)
		}
	}
	if hasNumericMaterializationLoop(factory.Block(withoutNumericBranch, true)) {
		t.Fatal("unconditional BigInt-loop mutation passed")
	}
	runner := filepath.Join(directory, "materialization.ts")
	writeFile(t, runner, `import { GoStringPointerBacking } from "./runtime/string-value.js";
import type { uint8 } from "@tsonic/core/types.js";

function check(value: boolean): void { if (!value) throw new Error("materialization mismatch"); }
function panics(operation: () => void): boolean {
    try { operation(); return false; } catch { return true; }
}
const values: uint8[] = [9, 0, 128, 255, 65, 10];
const backing = new GoStringPointerBacking({ kind: "indexed", values, offset: 1 });
for (let low = 0; low <= 4; low++) {
    for (let length = 0; length <= 4 - low; length++) {
        const expected = String.fromCharCode(...values.slice(1 + low, 1 + low + length));
        for (const start of [low, BigInt(low)]) {
            for (const count of [length, BigInt(length)]) {
                check(backing.text(start, count) === expected);
            }
        }
    }
}
values[2] = 42;
check(backing.text(1, 1) === "*");
check(backing.text(1n, 1n) === "*");
check(panics(() => backing.text(0.5, 0)));
delete values[2];
check(panics(() => backing.text(0, 3)));
check(panics(() => backing.text(0n, 3n)));

const many: uint8[] = Array.from({ length: 8195 }, (_, index) => (index % 256) as uint8);
const large = new GoStringPointerBacking({ kind: "indexed", values: many, offset: 1 });
for (const length of [0, 1, 4095, 4096, 4097, 8192, 8193]) {
    const expected = String.fromCharCode(...many.slice(1, 1 + length));
    check(large.text(0, length) === expected);
    check(large.text(0n, BigInt(length)) === expected);
}
many[4097] = 33;
check(large.text(4096, 1) === "!");
delete many[4097];
check(panics(() => large.text(0, 4097)));

const positions: bigint[] = [];
const addressed = new GoStringPointerBacking({
    kind: "pointer", offset: 9007199254740993n,
    at(index: number | bigint): never {
        check(typeof index === "bigint");
        positions.push(BigInt(index));
        throw new Error("observed exact position before load");
    },
});
for (const offset of [0, 3, 3n, 9007199254740991n, 9007199254740993n]) {
    check(panics(() => addressed.text(offset, 2)));
    check(positions[positions.length - 1] === 9007199254740993n + BigInt(offset));
}
const reads = positions.length;
check(addressed.text(9007199254740993n, 0n) === "");
check(addressed.text(0, 0) === "");
check(positions.length === reads);
console.log("materialization exact");
`)
	writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
	paths = append(paths, runner)
	compileTypeScript(t, directory, paths)
	if output := run(t, directory, "node", filepath.Join(directory, "out", "materialization.js")); output != "materialization exact\n" {
		t.Fatalf("materialization output = %q", output)
	}
}

func hasNumericMaterializationLoop(body tsgo.Block) bool {
	for _, statement := range body.Statements() {
		conditional, ok := statement.(tsgo.IfStatement)
		if !ok {
			continue
		}
		branch, ok := conditional.ThenStatement().(tsgo.Block)
		if !ok {
			continue
		}
		for _, nested := range branch.Statements() {
			loop, ok := nested.(tsgo.ForStatement)
			if !ok {
				continue
			}
			declarations, ok := loop.Initializer().(tsgo.VariableDeclarationList)
			if !ok || len(declarations.Declarations()) != 1 {
				continue
			}
			initial, ok := declarations.Declarations()[0].Initializer().(tsgo.NumericLiteral)
			if ok && initial.Text() == "0" {
				return true
			}
		}
	}
	return false
}
