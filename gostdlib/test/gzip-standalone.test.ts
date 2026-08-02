import assert from "node:assert/strict";
import test from "node:test";
import { gzipSync } from "node:zlib";

import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  NewReader,
  Reader as GzipReader,
} from "../src/compress/gzip.js";
import { ProviderInterfaceValue } from "../src/internal/portable/io/value.js";
import { bytes } from "../src/internal/runtime/slice.js";
import { state, type Reader } from "../src/io.js";

const memoryReaderType = Object.freeze({ comparable: true });

class MemoryReader extends ProviderInterfaceValue implements Reader {
  private offset = 0;

  constructor(private readonly source: Uint8Array) {
    super(memoryReaderType);
  }

  Read(destination: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    if (this.offset >= this.source.length) {
      return [0, state.EOF];
    }
    const count = Math.min(destination.length, this.source.length - this.offset);
    for (let index = 0; index < count; index += 1) {
      destination.set(index, this.source[this.offset + index] ?? 0);
    }
    this.offset += count;
    return [count, this.offset === this.source.length ? state.EOF : undefined];
  }
}

test("gzip reader decodes bytes and reports EOF", () => {
  const encoded = Uint8Array.from(gzipSync("hello gzip"));
  const [reader, creationFailure] = NewReader(new MemoryReader(encoded));
  assert.equal(creationFailure, undefined);
  if (reader === undefined) {
    assert.fail("NewReader returned no reader");
  }
  assert.equal(reader.Header.Name, "");
  assert.equal(reader.Header.Comment, "");
  assert.equal(reader.Header.ModTime.IsZero(), true);

  const decoded = RuntimeSlice.make<uint8>(32, 32, 0);
  const [count, readFailure] = GzipReader.Read(reader, decoded);
  assert.equal(readFailure, undefined);
  assert.equal(
    new TextDecoder().decode(bytes(decoded.slice(0, count, null))),
    "hello gzip",
  );
  assert.deepEqual(GzipReader.Read(reader, decoded), [0, state.EOF]);
  assert.equal(GzipReader.Close(reader), undefined);
});

test("gzip reader rejects an invalid header", () => {
  const [reader, failure] = NewReader(new MemoryReader(Uint8Array.of(1, 2, 3)));
  assert.equal(reader, undefined);
  assert.notEqual(failure, undefined);
});

test("gzip reader reports a corrupt checksum while reading", () => {
  const encoded = Uint8Array.from(gzipSync("checksum"));
  encoded[encoded.length - 8] = (encoded[encoded.length - 8] ?? 0) ^ 0xff;
  const [reader, creationFailure] = NewReader(new MemoryReader(encoded));
  assert.equal(creationFailure, undefined);
  assert.notEqual(reader, undefined);
  const [, readFailure] = GzipReader.Read(
    reader,
    RuntimeSlice.make<uint8>(32, 32, 0),
  );
  assert.equal(readFailure?.Error(), "gzip: invalid checksum");
});
