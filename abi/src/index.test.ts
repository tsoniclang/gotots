import assert from "node:assert/strict";
import test from "node:test";
import { createCompilerSessionFromFiles, createSourceSemanticsExtension, formatDiagnostics } from "@tsonic/tsts";
import type { Node } from "@tsonic/tsts";
import { createTsonicCoreSourceExtension, tsonicCoreSourceSemanticsModules } from "@tsonic/source-core";
import { readTsonicMemoryLayout, readTsonicRawMemoryOperation, selectTsonicRawLocationOperation, tsonicFixedArrayFactKey } from "@tsonic/source-core/facts";
import { goAbiCompilerContributions, goAbiProviderDeclarations } from "./index.js";

test("ABI certification consumes the same immutable declaration model", () => {
  const declarations = goAbiProviderDeclarations();
  assert.equal(declarations, goAbiProviderDeclarations());
  assert.equal(Object.isFrozen(declarations), true);
  assert.deepEqual(declarations.map(declaration => declaration.name), ["little32", "little64", "big32", "big64"]);
  for (const declaration of declarations) {
    assert.equal(Object.isFrozen(declaration), true);
    assert.equal(Object.isFrozen(declaration.type), true);
    assert.equal(declaration.kind, "value");
    assert.deepEqual(declaration.type, {
      kind: "provider-ref", moduleSpecifier: "@tsonic/core/types.js", exportName: "DataLayout",
    });
  }
  assert.equal(Reflect.set(declarations, "0", declarations[1]), false);
  assert.equal(goAbiProviderDeclarations()[0]?.name, "little32");
});

test("selected Go ABI tokens produce exact shared layout and raw-memory facts", () => {
  const contributions = goAbiCompilerContributions();
  const checked = checkABI(`
      import { little32, little64, big32, big64 } from "@gotots/abi/layout.js";
      import type { uint32 } from "@tsonic/core/types.js";
      import { memoryLayout, addressOf, toRawPointer, reinterpretRawPointer, storePointer } from "@tsonic/core/lang.js";
      const first = memoryLayout<uint32>(little32, 4, 4, 4);
      const second = memoryLayout<uint32>(little64, 4, 4, 4);
      const third = memoryLayout<uint32>(big32, 4, 4, 4);
      const fourth = memoryLayout<uint32>(big64, 4, 4, 4);
      let count: uint32 = 1;
      const raw = toRawPointer(addressOf(count), second);
      const view = reinterpretRawPointer(raw, second);
      if (view !== undefined) storePointer(view, 7);
    `);
  const diagnostics = checked.diagnostics.filter((diagnostic) => diagnostic !== undefined);
  assert.equal(diagnostics.length, 0, formatDiagnostics(diagnostics, "/src"));
  assert.equal(checked.extensionDiagnostics.length, 0,
    checked.extensionDiagnostics.map((diagnostic) => diagnostic.message).join("\n"));
  const layouts: string[] = [];
  const operations: string[] = [];
  const visit = (node: Node): void => {
    const layout = readTsonicMemoryLayout(checked.sourceFacts, node);
    if (layout?.call === node) {
      layouts.push(`${layout.dataLayout.byteOrder}:${layout.dataLayout.addressWidth}:${layout.byteSize}`);
    }
    const operation = readTsonicRawMemoryOperation(checked.sourceFacts, node);
    if (operation?.call === node) operations.push(operation.operation);
    for (const child of checked.ast.children(node)) if (child !== undefined) visit(child);
  };
  const source = checked.getSourceFile("/src/index.ts");
  assert.ok(source);
  visit(source);
  assert.deepEqual(layouts, ["little:32:4", "little:64:4", "big:32:4", "big:64:4"]);
  assert.deepEqual(operations, ["to-raw", "reinterpret"]);
  assert.deepEqual(goAbiCompilerContributions().dataLayouts, contributions.dataLayouts);
});

test("same-spelled authored ABI objects do not acquire provider evidence", () => {
  const checked = checkABI(`
    import type { DataLayout, uint32 } from "@tsonic/core/types.js";
    import { memoryLayout } from "@tsonic/core/lang.js";
    const little64: DataLayout = { __tsonicDataLayout: "DataLayout" };
    export const word = memoryLayout<uint32>(little64, 4, 4, 4);
  `);
  assert.equal(checked.diagnostics.length, 0);
  assert.equal(checked.extensionDiagnostics.length, 1);
  assert.equal(checked.extensionDiagnostics[0]?.extensionCode, "SOURCE_CORE_MEMORY_LAYOUT_NOT_PROVEN");
});

test("an imported ABI without its registration fails at the shared owner", () => {
  const checked = checkABI(`
    import { little64 } from "@gotots/abi/layout.js";
    import type { uint32 } from "@tsonic/core/types.js";
    import { memoryLayout } from "@tsonic/core/lang.js";
    export const word = memoryLayout<uint32>(little64, 4, 4, 4);
  `, false);
  assert.equal(checked.diagnostics.length, 0);
  assert.equal(checked.extensionDiagnostics.length, 1);
  assert.equal(checked.extensionDiagnostics[0]?.extensionCode, "SOURCE_CORE_MEMORY_LAYOUT_NOT_PROVEN");
});

test("Go ABI selections retain exact address domains and nested child layouts", () => {
  const checked = checkABI(`
    import { little32, little64 } from "@gotots/abi/layout.js";
    import type { uint32, uint64, RawPointer } from "@tsonic/core/types.js";
    import { memoryLayout, memoryField, rawPointerToAddressInteger, addressIntegerToRawPointer, reinterpretRawPointer } from "@tsonic/core/lang.js";
    type Pair = { first: uint32; second: uint32 };
    type Outer = { tag: uint32; inner: Pair };
    const word = memoryLayout<uint32>(little64, 4, 4, 4);
    const pair = memoryLayout<Pair>(little64, 8, 4, 8,
      memoryField((value: Pair) => value.first, 0, 4, word),
      memoryField((value: Pair) => value.second, 4, 4, word));
    const outer = memoryLayout<Outer>(little64, 12, 4, 12,
      memoryField((value: Outer) => value.tag, 0, 4, word),
      memoryField((value: Outer) => value.inner, 4, 4, pair));
    declare const raw: RawPointer | undefined;
    const address32 = rawPointerToAddressInteger<uint32>(raw, little32);
    const address64 = rawPointerToAddressInteger<uint64>(raw, little64);
    addressIntegerToRawPointer(address32, little32);
    addressIntegerToRawPointer(address64, little64);
    reinterpretRawPointer(raw, outer);
  `);
  const diagnostics = checked.diagnostics.filter(diagnostic => diagnostic !== undefined);
  assert.equal(diagnostics.length, 0, formatDiagnostics(diagnostics, "/src"));
  assert.equal(checked.extensionDiagnostics.length, 0, checked.extensionDiagnostics.map(diagnostic => diagnostic.message).join("\n"));
  const domains: string[] = [];
  let selectedRecords = 0;
  const visit = (node: Node): void => {
    const operation = readTsonicRawMemoryOperation(checked.sourceFacts, node);
    if (operation?.operation === "raw-to-address-integer" || operation?.operation === "address-integer-to-raw") {
      domains.push(`${operation.operation}:${operation.addressWidth}:${operation.addressRuntimeBase}:${operation.addressSignedness}`);
    }
    const selected = selectTsonicRawLocationOperation(checked.ast, checked.sourceFacts, node);
    if (selected !== undefined) {
      assert.equal(selected.kind, "resolved");
      if (selected.kind === "resolved") {
        assert.equal(selected.layout.byteSize, 12);
        assert.ok(selected.layout.kind === "value");
        const inner = selected.layout.fields[1]?.fieldLayout;
        assert.ok(inner?.kind === "value");
        assert.equal(inner.byteSize, 8);
        assert.equal(inner.fields[1]?.byteOffset, 4);
      }
      selectedRecords++;
    }
    for (const child of checked.ast.children(node)) if (child !== undefined) visit(child);
  };
  const source = checked.getSourceFile("/src/index.ts");
  assert.ok(source);
  visit(source);
  assert.deepEqual(domains, [
    "raw-to-address-integer:32:number:unsigned", "raw-to-address-integer:64:bigint:unsigned",
    "address-integer-to-raw:32:number:unsigned", "address-integer-to-raw:64:bigint:unsigned",
  ]);
  assert.equal(selectedRecords, 1);
});

test("a slice-shaped descriptor retains its address and both integer child layouts", () => {
  const checked = checkABI(`
    import { little64 } from "@gotots/abi/layout.js";
    import type { RawPointer, int64 } from "@tsonic/core/types.js";
    import { memoryLayout, memoryField, reinterpretRawPointer } from "@tsonic/core/lang.js";
    type Header = { data: RawPointer | undefined; length: int64; capacity: int64 };
    const address = memoryLayout<RawPointer | undefined>(little64, 8, 8, 8);
    const count = memoryLayout<int64>(little64, 8, 8, 8);
    const header = memoryLayout<Header>(little64, 24, 8, 24,
      memoryField((value: Header) => value.data, 0, 8, address),
      memoryField((value: Header) => value.length, 8, 8, count),
      memoryField((value: Header) => value.capacity, 16, 8, count));
    declare const raw: RawPointer | undefined;
    reinterpretRawPointer(raw, header);
  `);
  assert.equal(checked.diagnostics.length, 0, formatDiagnostics(checked.diagnostics.filter(diagnostic => diagnostic !== undefined), "/src"));
  assert.deepEqual(checked.extensionDiagnostics, []);
  let selectedHeaders = 0;
  const visit = (node: Node): void => {
    const selected = selectTsonicRawLocationOperation(checked.ast, checked.sourceFacts, node);
    if (selected !== undefined) {
      assert.equal(selected.kind, "resolved");
      if (selected.kind === "resolved") {
        selectedHeaders++;
        assert.ok(selected.layout.kind === "value");
        assert.deepEqual(selected.layout.fields.map(field => [field.byteOffset, field.fieldLayout.byteSize]),
          [[0, 8], [8, 8], [16, 8]]);
        for (const field of selected.layout.fields) {
          assert.equal(field.fieldLayout.dataLayout.fingerprint, selected.layout.dataLayout.fingerprint);
          assert.equal(readTsonicMemoryLayout(checked.sourceFacts, field.fieldLayoutExpression)?.call, field.fieldLayout.call);
        }
      }
    }
    for (const child of checked.ast.children(node)) if (child !== undefined) visit(child);
  };
  const source = checked.getSourceFile("/src/index.ts");
  assert.ok(source);
  visit(source);
  assert.equal(selectedHeaders, 1);
});

test("a fixed-array index cannot masquerade as a declared physical record field", () => {
  const checked = checkABI(`
    import { little64 } from "@gotots/abi/layout.js";
    import type { FixedArray, uint32 } from "@tsonic/core/types.js";
    import { memoryLayout, memoryField } from "@tsonic/core/lang.js";
    const word = memoryLayout<uint32>(little64, 4, 4, 4);
    export const array = memoryLayout<FixedArray<uint32, 2>>(little64, 8, 4, 8,
      memoryField((value: FixedArray<uint32, 2>) => value[0], 0, 4, word));
  `);
  assert.equal(checked.diagnostics.length, 0, formatDiagnostics(checked.diagnostics.filter(diagnostic => diagnostic !== undefined), "/src"));
  assert.deepEqual(checked.extensionDiagnostics.map(diagnostic => diagnostic.extensionCode), [
    "SOURCE_CORE_MEMORY_FIELD_NOT_PROVEN", "SOURCE_CORE_MEMORY_ARRAY_ELEMENT_REQUIRED",
  ]);
  const extents = new Set<bigint>();
  const visit = (node: Node): void => {
    const array = checked.sourceFacts.getFact(node, tsonicFixedArrayFactKey);
    if (array !== undefined) extents.add(array.length);
    const layout = readTsonicMemoryLayout(checked.sourceFacts, node);
    if (layout !== undefined) assert.equal(layout.byteSize, 4);
    for (const child of checked.ast.children(node)) if (child !== undefined) visit(child);
  };
  const source = checked.getSourceFile("/src/index.ts");
  assert.ok(source);
  visit(source);
  assert.deepEqual([...extents], [2n]);
});

test("Go ABI arrays retain exact children and large zero-sized extents", () => {
  const checked = checkABI(`
    import { little64 } from "@gotots/abi/layout.js";
    import type { uint32 } from "@tsonic/core/types.js";
    import { memoryLayout, memoryArrayLayout } from "@tsonic/core/lang.js";
    const word = memoryLayout<uint32>(little64, 4, 4, 4);
    const pair = memoryArrayLayout(little64, 8, 4, 8, word, 2);
    const zero = memoryLayout<{}>(little64, 0, 1, 0);
    const huge = memoryArrayLayout(little64, 0, 1, 0, zero, 9007199254740993n);
  `);
  assert.equal(checked.diagnostics.length, 0, formatDiagnostics(checked.diagnostics.filter(diagnostic => diagnostic !== undefined), "/src"));
  assert.deepEqual(checked.extensionDiagnostics, []);
  const arrays: { length: bigint; runtimeBase: string; byteSize: number; elementSize: number }[] = [];
  const visit = (node: Node): void => {
    const layout = readTsonicMemoryLayout(checked.sourceFacts, node);
    if (layout?.kind === "array" && layout.call === node) {
      assert.equal(layout.elementLayout.dataLayout.fingerprint, layout.dataLayout.fingerprint);
      assert.equal(readTsonicMemoryLayout(checked.sourceFacts, layout.elementLayoutExpression)?.call, layout.elementLayout.call);
      arrays.push({ length: layout.fixedArray.length, runtimeBase: layout.fixedArray.lengthRuntimeBase,
        byteSize: layout.byteSize, elementSize: layout.elementLayout.byteSize });
    }
    for (const child of checked.ast.children(node)) if (child !== undefined) visit(child);
  };
  const source = checked.getSourceFile("/src/index.ts");
  assert.ok(source);
  visit(source);
  assert.deepEqual(arrays, [
    { length: 2n, runtimeBase: "number", byteSize: 8, elementSize: 4 },
    { length: 9007199254740993n, runtimeBase: "bigint", byteSize: 0, elementSize: 0 },
  ]);
});

function checkABI(text: string, registerLayouts = true) {
  const contributions = goAbiCompilerContributions();
  return createCompilerSessionFromFiles({
    currentDirectory: "/src",
    files: { "/src/index.ts": text },
    compilerOptions: { strict: true, target: "es2022", module: "esnext", moduleResolution: "bundler" },
    extensionHostOptions: { extensions: [
      createSourceSemanticsExtension({ modules: tsonicCoreSourceSemanticsModules() }),
      createTsonicCoreSourceExtension({ dataLayouts: registerLayouts ? contributions.dataLayouts ?? [] : [] }),
      ...contributions.extensions ?? [],
    ] },
  }).checkSource();
}
