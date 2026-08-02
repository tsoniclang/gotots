import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import test from "node:test";

import {
  GoInterfaceValue,
  type GoError,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { uint8 } from "@gotots/runtime/scalars.js";

import { Fprintln, Println, Sprintf } from "../src/fmt.js";
import type { Writer } from "../src/io.js";

class FormattedValue extends GoInterfaceValue {
	static readonly comparable = true;
	readonly $go$type = FormattedValue;
  readonly $go$methods: ReadonlySet<object> = new Set<object>();
  readonly $go$formatString: boolean;

  constructor(
    private readonly decimal: string,
    private readonly hexadecimal: string,
  ) {
    super();
    this.$go$formatString = decimal === "value";
  }

  $go$implements(contract: readonly object[]): boolean {
    return contract.length === 0;
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    return 0;
  }

  $go$format(verb: string, _flags: string, _precision: number | undefined): string {
    return verb === "x" ? this.hexadecimal : this.decimal;
  }
}

class CaptureWriter extends GoInterfaceValue implements Writer {
	static readonly comparable = true;
	readonly $go$type = CaptureWriter;
  readonly $go$methods: ReadonlySet<object> = new Set<object>();
  readonly $go$formatString = false;
  content = "";

  $go$implements(contract: readonly object[]): boolean {
    return contract.length === 0;
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    return 0;
  }

  $go$format(_verb: string, _flags: string, _precision: number | undefined): string {
    return "capture-writer";
  }

  Write(buffer: RuntimeSlice<uint8>): [number, GoError | undefined] {
    const bytes: number[] = [];
    for (let index = 0; index < buffer.length; index += 1) {
      bytes.push(buffer.get(index));
    }
    this.content += new TextDecoder().decode(Uint8Array.from(bytes));
    return [buffer.length, undefined];
  }
}

test("fmt parses directives and delegates exact dynamic-value rendering", () => {
  const arguments_ = RuntimeSlice.literal<GoInterfaceValue | undefined>([
    new FormattedValue("value", "value"),
    new FormattedValue("15", "f"),
  ]);
  assert.equal(Sprintf("%s=%04x", arguments_), "value=000f");
});

test("fmt Fprintln writes one Go line through io.Writer", () => {
  const writer = new CaptureWriter();
  const arguments_ = RuntimeSlice.literal<GoInterfaceValue | undefined>([
    new FormattedValue("value", "value"),
    new FormattedValue("15", "f"),
  ]);
  assert.deepEqual(Fprintln(writer, arguments_), [9, undefined]);
  assert.equal(writer.content, "value 15\n");
});

test("fmt Println writes through the selected standard output", (): void => {
  assert.equal(typeof Println, "function");
  const moduleURL = new URL("../src/fmt.js", import.meta.url).href;
  const script = `
    import { Println } from ${JSON.stringify(moduleURL)};
    import { RuntimeSlice } from "@gotots/runtime/slice.js";
    const result = Println(RuntimeSlice.literal([]));
    process.stderr.write(JSON.stringify(result));
  `;
  const result = spawnSync(
    process.execPath,
    ["--input-type=module", "--eval", script],
    { cwd: process.cwd(), encoding: "utf8" },
  );
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, "\n");
  assert.equal(result.stderr, "[1,null]");
});
