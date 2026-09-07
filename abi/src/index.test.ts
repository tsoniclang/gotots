import assert from "node:assert/strict";
import test from "node:test";
import { createCompilerSessionFromFiles, createSourceSemanticsExtension, formatDiagnostics } from "@tsonic/tsts";
import type { Node } from "@tsonic/tsts";
import { createTsonicCoreSourceExtension, tsonicCoreSourceSemanticsModules } from "@tsonic/source-core";
import { readTsonicMemoryLayout, readTsonicRawMemoryOperation, selectTsonicRawLocationOperation } from "@tsonic/source-core/facts";
import { goAbiCompilerContributions } from "./index.js";

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
        assert.equal(selected.layout.fields[1]?.fieldLayout.byteSize, 8);
        assert.equal(selected.layout.fields[1]?.fieldLayout.fields[1]?.byteOffset, 4);
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
