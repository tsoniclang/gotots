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
import { GoPanic } from "@gotots/runtime/panic.js";
import type { uint8 } from "@gotots/runtime/scalars.js";

import * as binary from "../src/encoding/binary.js";
import {
  BigEndianOrder,
  LittleEndianOrder,
} from "../src/internal/portable/encoding/binary/byte-order.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

test("byte orders round-trip selected integer widths", () => {
  for (const order of [binary.state.BigEndian, binary.state.LittleEndian]) {
    const buffer = RuntimeSlice.make<uint8>(8, 8, 0);
    order.PutUint16(buffer, 0xabcd);
    assert.equal(order.Uint16(buffer), 0xabcd);
    order.PutUint32(buffer, 0x89ab_cdef);
    assert.equal(order.Uint32(buffer), 0x89ab_cdef);
    order.PutUint64(buffer, 0x1f_ffff_ffffn);
    assert.equal(order.Uint64(buffer), 0x1f_ffff_ffffn);
  }
});

test("AppendUint32 remains an unexported-receiver operation", (): void => {
  assert.equal(Object.hasOwn(binary, "AppendUint32"), false);
  const prefix = RuntimeSlice.literal<uint8>([0xaa]);
  assert.deepEqual(
    sliceValues(new BigEndianOrder().AppendUint32(prefix, 0x0102_0304)),
    [0xaa, 1, 2, 3, 4],
  );
  assert.deepEqual(
    sliceValues(binary.state.BigEndian.AppendUint16(prefix, 0x0102)),
    [0xaa, 1, 2],
  );
  assert.deepEqual(
    sliceValues(binary.state.LittleEndian.AppendUint64(prefix, 0x0102_0304n)),
    [0xaa, 4, 3, 2, 1, 0, 0, 0, 0],
  );
  assert.deepEqual(
    sliceValues(new LittleEndianOrder().AppendUint32(prefix, 0x0102_0304)),
    [0xaa, 4, 3, 2, 1],
  );
});

test("structured I/O preserves Go nil-interface panics", (): void => {
  for (const operation of [binary.Read, binary.Write]) {
    assert.throws(
      () => operation(undefined, undefined, undefined),
      (failure: unknown): boolean => {
        assert.ok(failure instanceof GoPanic);
        assert.equal(
          failure.value.$go$format("v", "", undefined),
          "invalid memory address or nil pointer dereference",
        );
        return true;
      },
    );
  }
});

test("internal AppendUint32 agrees with Go receiver methods", (): void => {
  const prefix = RuntimeSlice.literal<uint8>([0xaa]);
  const provider = JSON.stringify({
    big: sliceValues(new BigEndianOrder().AppendUint32(prefix, 0x0102_0304)),
    little: sliceValues(new LittleEndianOrder().AppendUint32(prefix, 0x0102_0304)),
  });
  assert.equal(provider, runGo(goProgram));
});

function runGo(program: string): string {
  const directory = mkdtempSync(join(tmpdir(), "gotots-binary-"));
  const source = join(directory, "main.go");
  try {
    writeFileSync(source, program);
    const result = spawnSync("go", ["run", source], {
      encoding: "utf8",
    });
    assert.equal(result.status, 0, result.stderr);
    return result.stdout.trim();
  } finally {
    rmSync(directory, {
      force: true,
      recursive: true,
    });
  }
}

const goProgram = `
package main

import (
  "encoding/binary"
  "encoding/json"
  "fmt"
)

func main() {
  prefix := []byte{0xaa}
  result := struct {
    Big []int \`json:"big"\`
    Little []int \`json:"little"\`
  }{
    Big: integers(binary.BigEndian.AppendUint32(prefix, 0x01020304)),
    Little: integers(binary.LittleEndian.AppendUint32(prefix, 0x01020304)),
  }
  encoded, err := json.Marshal(result)
  if err != nil {
    panic(err)
  }
  fmt.Println(string(encoded))
}

func integers(values []byte) []int {
  result := make([]int, len(values))
  for index, value := range values {
    result[index] = int(value)
  }
  return result
}
`;
