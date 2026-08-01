import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  mkdtempSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";

import type { TextMarshaler } from "../src/encoding.js";
import {
  AppendDecode,
  AppendEncode,
  EncodedLen,
} from "../src/encoding/hex.js";
import { ProviderInterfaceValue } from "../src/internal/portable/io/value.js";
import { sliceValues } from "../src/internal/runtime/slice.js";
import { Search, Sort, Stable, Strings, type Interface } from "../src/sort.js";

const sortableType = Object.freeze({});

class Sortable extends ProviderInterfaceValue implements Interface {
  constructor(readonly values: { readonly key: number; readonly label: string }[]) {
    super(sortableType);
  }

  Len(): number {
    return this.values.length;
  }

  Less(left: number, right: number): boolean {
    return (this.values[left]?.key ?? 0) < (this.values[right]?.key ?? 0);
  }

  Swap(left: number, right: number): void {
    const saved = this.values[left];
    if (saved === undefined || this.values[right] === undefined) {
      throw new Error("test index is outside the sortable value");
    }
    this.values[left] = this.values[right];
    this.values[right] = saved;
  }
}

test("sort.Search finds the first true index", (): void => {
  assert.equal(Search(10, (index): boolean => index >= 6), 6);
  assert.equal(Search(10, (): boolean => false), 10);
  assert.equal(Search(0, (): boolean => true), 0);
});

test("sort interface operations are in-place and stable when requested", (): void => {
  const ordinary = new Sortable([
    { key: 3, label: "c" },
    { key: 1, label: "a" },
    { key: 2, label: "b" },
  ]);
  Sort(ordinary);
  assert.deepEqual(ordinary.values.map(({ key }): number => key), [1, 2, 3]);

  const stable = new Sortable([
    { key: 2, label: "first" },
    { key: 1, label: "middle" },
    { key: 2, label: "second" },
  ]);
  Stable(stable);
  assert.deepEqual(stable.values.map(({ label }): string => label), [
    "middle",
    "first",
    "second",
  ]);
});

test("sort.Strings mutates the selected runtime slice", (): void => {
  const values = RuntimeSlice.literal(["beta", "alpha", "gamma"]);
  Strings(values);
  assert.deepEqual(
    [values.get(0), values.get(1), values.get(2)],
    ["alpha", "beta", "gamma"],
  );
});

test("encoding.TextMarshaler remains a static contract", (): void => {
  const accepts = (_value: TextMarshaler): void => {};
  assert.equal(typeof accepts, "function");
});

test("encoding/hex appends encoded and partially decoded bytes", (): void => {
  assert.equal(EncodedLen(3), 6);
  const encoded = AppendEncode(
    RuntimeSlice.literal([0x78]),
    RuntimeSlice.literal([0xab, 0xcd]),
  );
  assert.deepEqual(sliceValues(encoded), [0x78, 0x61, 0x62, 0x63, 0x64]);

  const [decoded, decodeFailure] = AppendDecode(
    RuntimeSlice.literal([0x78]),
    RuntimeSlice.literal([0x36, 0x31, 0x36, 0x32]),
  );
  assert.equal(decodeFailure, undefined);
  assert.deepEqual(sliceValues(decoded), [0x78, 0x61, 0x62]);

  const [partial, partialFailure] = AppendDecode(
    RuntimeSlice.literal([0x78]),
    RuntimeSlice.literal([0x36, 0x31, 0x67, 0x30]),
  );
  assert.deepEqual(sliceValues(partial), [0x78, 0x61]);
  assert.equal(partialFailure?.Error().includes("invalid byte"), true);
});

test("encoding/hex append family agrees with Go", (): void => {
  const directory = mkdtempSync(join(tmpdir(), "gotots-hex-"));
  const source = join(directory, "main.go");
  try {
    writeFileSync(source, hexGoProgram);
    const result = spawnSync("go", ["run", source], { encoding: "utf8" });
    assert.equal(result.status, 0, result.stderr);
    assert.equal(hexProviderResult(), result.stdout.trim());
  } finally {
    rmSync(directory, { force: true, recursive: true });
  }
});

function hexProviderResult(): string {
  const encoded = AppendEncode(
    RuntimeSlice.literal([0x78]),
    RuntimeSlice.literal([0xab, 0xcd]),
  );
  const [decoded, decodedFailure] = AppendDecode(
    RuntimeSlice.literal([0x78]),
    RuntimeSlice.literal([0x36, 0x31, 0x36, 0x32]),
  );
  const [partial, partialFailure] = AppendDecode(
    RuntimeSlice.literal([0x78]),
    RuntimeSlice.literal([0x36, 0x31, 0x67, 0x30]),
  );
  const [odd, oddFailure] = AppendDecode(
    RuntimeSlice.literal([0x78]),
    RuntimeSlice.literal([0x36, 0x31, 0x36]),
  );
  const text = (value: RuntimeSlice<number>): string => String.fromCharCode(
    ...sliceValues(value),
  );
  return [
    text(encoded),
    `${text(decoded)}:${decodedFailure?.Error() ?? ""}`,
    `${text(partial)}:${partialFailure?.Error() ?? ""}`,
    `${text(odd)}:${oddFailure?.Error() ?? ""}`,
  ].join("|");
}

const hexGoProgram = `
package main

import (
  "encoding/hex"
  "fmt"
)

func errorText(err error) string {
  if err == nil {
    return ""
  }
  return err.Error()
}

func main() {
  encoded := hex.AppendEncode([]byte("x"), []byte{0xab, 0xcd})
  decoded, decodedErr := hex.AppendDecode([]byte("x"), []byte("6162"))
  partial, partialErr := hex.AppendDecode([]byte("x"), []byte("61g0"))
  odd, oddErr := hex.AppendDecode([]byte("x"), []byte("616"))
  fmt.Printf("%s|%s:%s|%s:%s|%s:%s\\n", encoded, decoded, errorText(decodedErr), partial, errorText(partialErr), odd, errorText(oddErr))
}
`;
