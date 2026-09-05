import assert from "node:assert/strict";
import test from "node:test";
import { createCompilerSessionFromFiles, createSourceSemanticsExtension, formatDiagnostics } from "@tsonic/tsts";
import type { Node } from "@tsonic/tsts";
import { createTsonicCoreSourceExtension, tsonicCoreSourceSemanticsModules } from "@tsonic/source-core";
import { readTsonicMemoryLayout, readTsonicRawMemoryOperation } from "@tsonic/source-core/facts";
import { goAbiCompilerContributions } from "./index.js";

test("selected Go ABI tokens produce exact shared layout and raw-memory facts", () => {
  const contributions = goAbiCompilerContributions();
  const checked = createCompilerSessionFromFiles({
    currentDirectory: "/src",
    files: { "/src/index.ts": `
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
    ` },
    compilerOptions: { strict: true, target: "es2022", module: "esnext", moduleResolution: "bundler" },
    extensionHostOptions: { extensions: [
      createSourceSemanticsExtension({ modules: tsonicCoreSourceSemanticsModules() }),
      createTsonicCoreSourceExtension({ dataLayouts: contributions.dataLayouts ?? [] }),
      ...contributions.extensions ?? [],
    ] },
  }).checkSource();
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
