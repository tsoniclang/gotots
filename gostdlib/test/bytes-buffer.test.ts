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

import {
  Buffer as ByteBuffer,
  NewBuffer,
} from "../src/bytes.js";
import { sliceValues } from "../src/internal/runtime/slice.js";
import { state as ioState } from "../src/io.js";

test("Buffer.Read advances exactly and preserves empty-read boundaries", (): void => {
  const source = RuntimeSlice.literal([1, 2, 3]);
  const buffer = NewBuffer(source);
  const empty = RuntimeSlice.nil<number>();
  const target = RuntimeSlice.literal([9, 9]);

  assert.deepEqual(ByteBuffer.Read(buffer, empty), [0, undefined]);
  assert.deepEqual(ByteBuffer.Read(buffer, target), [2, undefined]);
  assert.deepEqual(sliceValues(target), [1, 2]);
  assert.deepEqual(ByteBuffer.Read(buffer, target), [1, undefined]);
  assert.deepEqual(sliceValues(target), [3, 2]);
  assert.deepEqual(ByteBuffer.Read(buffer, target), [0, ioState.EOF]);
  assert.deepEqual(ByteBuffer.Read(buffer, empty), [0, undefined]);

  assert.deepEqual(
    ByteBuffer.Read(new ByteBuffer(), RuntimeSlice.nil<number>()),
    [0, undefined],
  );
  assert.throws((): void => {
    ByteBuffer.Read(undefined, target);
  });
});

test("Buffer exposes unread length, growth, next, and write behavior", (): void => {
  const buffer = NewBuffer(RuntimeSlice.literal([1, 2, 3]));
  assert.equal(ByteBuffer.Len(buffer), 3);
  assert.deepEqual(sliceValues(ByteBuffer.Next(buffer, 2)), [1, 2]);
  assert.equal(ByteBuffer.Len(buffer), 1);
  ByteBuffer.Grow(buffer, 4);
  assert.equal(ByteBuffer.Available(buffer) >= 4, true);
  assert.equal(ByteBuffer.AvailableBuffer(buffer).length, 0);
  assert.deepEqual(ByteBuffer.Write(buffer, RuntimeSlice.literal([4, 5])), [2, undefined]);
  assert.deepEqual(sliceValues(ByteBuffer.Next(buffer, 10)), [3, 4, 5]);
});

test("Buffer and NewBuffer agree with Go", (): void => {
  assert.equal(providerResult(), runGo(goProgram));
});

function providerResult(): string {
  const buffer = NewBuffer(RuntimeSlice.literal([1, 2, 3]));
  const target = RuntimeSlice.literal([9, 9]);
  const rows: {
    n: number;
    error: string;
    destination: number[];
  }[] = [];

  for (let readIndex = 0; readIndex < 3; readIndex += 1) {
    const [count, failure] = ByteBuffer.Read(buffer, target);
    rows.push({
      n: count,
      error: failure?.Error() ?? "",
      destination: sliceValues(target),
    });
  }

  const [emptyCount, emptyFailure] = ByteBuffer.Read(
    buffer,
    RuntimeSlice.nil<number>(),
  );
  rows.push({
    n: emptyCount,
    error: emptyFailure?.Error() ?? "",
    destination: [],
  });
  return JSON.stringify(rows);
}

function runGo(program: string): string {
  const directory = mkdtempSync(join(tmpdir(), "gotots-bytes-buffer-"));
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
  "bytes"
  "encoding/json"
  "fmt"
)

type row struct {
  N int \`json:"n"\`
  Error string \`json:"error"\`
  Destination []int \`json:"destination"\`
}

func errorText(err error) string {
  if err == nil {
    return ""
  }
  return err.Error()
}

func integers(values []byte) []int {
  result := make([]int, len(values))
  for index, value := range values {
    result[index] = int(value)
  }
  return result
}

func main() {
  buffer := bytes.NewBuffer([]byte{1, 2, 3})
  target := []byte{9, 9}
  rows := make([]row, 0, 4)
  for index := 0; index < 3; index++ {
    count, err := buffer.Read(target)
    rows = append(rows, row{
      N: count,
      Error: errorText(err),
      Destination: integers(target),
    })
  }
  count, err := buffer.Read(nil)
  rows = append(rows, row{N: count, Error: errorText(err), Destination: []int{}})
  encoded, err := json.Marshal(rows)
  if err != nil {
    panic(err)
  }
  fmt.Println(string(encoded))
}
`;
