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

import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  NewScanner,
  Scanner,
} from "../src/bufio.js";
import { New } from "../src/errors.js";
import { ProviderInterfaceValue } from "../src/internal/portable/io/value.js";
import { state as ioState, type Reader } from "../src/io.js";

const chunkReaderType = Object.freeze({});

class ChunkReader extends ProviderInterfaceValue implements Reader {
  #offset = 0;

  constructor(
    private readonly source: string,
    private readonly chunkSize: number,
    private readonly terminalFailure: GoError = ioState.EOF,
  ) {
    super(chunkReaderType);
  }

  Read(destination: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    if (this.#offset >= this.source.length) {
      return [0, this.terminalFailure];
    }
    const count = Math.min(
      destination.length,
      this.chunkSize,
      this.source.length - this.#offset,
    );
    for (let index = 0; index < count; index += 1) {
      destination.set(index, this.source.charCodeAt(this.#offset + index));
    }
    this.#offset += count;
    return [count, undefined];
  }
}

test("bufio Scanner preserves line semantics across read boundaries", (): void => {
  const scanner = NewScanner(new ChunkReader("alpha\r\n\ncharlie", 2));
  const lines: string[] = [];
  while (Scanner.Scan(scanner)) {
    lines.push(Scanner.Text(scanner));
  }
  assert.deepEqual(lines, ["alpha", "", "charlie"]);
  assert.equal(Scanner.Err(scanner), undefined);
  assert.equal(Scanner.Scan(scanner), false);
});

test("bufio Scanner agrees with Go for chunked default line scanning", (): void => {
  const directory = mkdtempSync(join(tmpdir(), "gotots-scanner-"));
  const source = join(directory, "main.go");
  try {
    writeFileSync(source, scannerGoProgram);
    const result = spawnSync("go", ["run", source], { encoding: "utf8" });
    assert.equal(result.status, 0, result.stderr);
    assert.equal(scannerProviderResult(), result.stdout.trim());
  } finally {
    rmSync(directory, { force: true, recursive: true });
  }
});

test("bufio Scanner reports non-EOF and bounded-token failures", (): void => {
  const readFailure = New("read failed");
  const failing = NewScanner(new ChunkReader("tail", 2, readFailure));
  assert.equal(Scanner.Scan(failing), true);
  assert.equal(Scanner.Text(failing), "tail");
  assert.equal(Scanner.Scan(failing), false);
  assert.equal(Scanner.Err(failing), readFailure);

  const oversized = NewScanner(new ChunkReader("x".repeat(64 * 1024), 4096));
  assert.equal(Scanner.Scan(oversized), false);
  assert.equal(Scanner.Err(oversized)?.Error(), "bufio.Scanner: token too long");
});

function scannerProviderResult(): string {
  const scanner = NewScanner(new ChunkReader("alpha\r\n\ncharlie", 2));
  const tokens: string[] = [];
  while (Scanner.Scan(scanner)) {
    tokens.push(JSON.stringify(Scanner.Text(scanner)));
  }
  return `${tokens.join("|")}:${Scanner.Err(scanner)?.Error() ?? ""}`;
}

const scannerGoProgram = `
package main

import (
  "bufio"
  "fmt"
  "io"
  "strings"
)

type chunkReader struct {
  source string
  offset int
}

func (r *chunkReader) Read(target []byte) (int, error) {
  if r.offset >= len(r.source) {
    return 0, io.EOF
  }
  count := 2
  if count > len(target) {
    count = len(target)
  }
  if count > len(r.source)-r.offset {
    count = len(r.source)-r.offset
  }
  copy(target, r.source[r.offset:r.offset+count])
  r.offset += count
  return count, nil
}

func main() {
  scanner := bufio.NewScanner(&chunkReader{source: "alpha\\r\\n\\ncharlie"})
  var tokens []string
  for scanner.Scan() {
    tokens = append(tokens, fmt.Sprintf("%q", scanner.Text()))
  }
  errorText := ""
  if scanner.Err() != nil {
    errorText = scanner.Err().Error()
  }
  fmt.Printf("%s:%s\\n", strings.Join(tokens, "|"), errorText)
}
`;
