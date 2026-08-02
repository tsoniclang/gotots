import assert from "node:assert/strict";
import test from "node:test";

import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  NewReader,
  NewWriter,
  Reader as BufferedReader,
  Writer as BufferedWriter,
} from "../src/bufio.js";
import { Is, New, state as errorState } from "../src/errors.js";
import {
  ErrorsAsTypeKernel as AsType,
} from "../src/internal/facets/generic-errors-kernel.js";
import { ProviderInterfaceValue } from "../src/internal/portable/io/value.js";
import {
  bytes,
  byteSlice,
  sliceValues,
} from "../src/internal/runtime/slice.js";
import {
  ReadFull,
  state,
  type Reader,
  type Writer,
} from "../src/io.js";

const memoryReaderType = Object.freeze({ comparable: true });
const memoryWriterType = Object.freeze({ comparable: true });

class MemoryReader extends ProviderInterfaceValue implements Reader {
  private offset = 0;

  constructor(
    private readonly source: Uint8Array,
    private readonly chunkSize = source.length,
  ) {
    super(memoryReaderType);
  }

  Read(destination: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    if (this.offset >= this.source.length) {
      return [0, state.EOF];
    }
    const count = Math.min(
      destination.length,
      this.chunkSize,
      this.source.length - this.offset,
    );
    for (let index = 0; index < count; index += 1) {
      destination.set(index, this.source[this.offset + index] ?? 0);
    }
    this.offset += count;
    return [count, this.offset === this.source.length ? state.EOF : undefined];
  }
}

class MemoryWriter extends ProviderInterfaceValue implements Writer {
  readonly values: number[] = [];

  constructor() {
    super(memoryWriterType);
  }

  Write(source: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    this.values.push(...sliceValues(source));
    return [source.length, undefined];
  }
}

class ShortWriter extends ProviderInterfaceValue implements Writer {
  constructor() {
    super(memoryWriterType);
  }

  Write(source: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    return [Math.max(0, source.length - 1), undefined];
  }
}

test("errors preserve sentinel identity", () => {
  const first = New("first");
  const second = New("first");
  assert.equal(Is(first, first), true);
  assert.equal(Is(first, second), false);
  assert.equal(first.Error(), "first");
  const [selected, ok] = AsType<GoError | undefined>(
    (failure): [GoError | undefined, boolean] => [
      failure,
      failure === first,
    ],
    first,
  );
  assert.equal(selected, first);
  assert.equal(ok, true);
  assert.equal(Is(errorState.ErrUnsupported, errorState.ErrUnsupported), true);
  assert.equal(Is(errorState.ErrUnsupported, New("unsupported operation")), false);
  assert.equal(errorState.ErrUnsupported.Error(), "unsupported operation");
  assert.equal(state.ErrShortWrite.Error(), "short write");
  assert.equal(state.ErrShortBuffer.Error(), "short buffer");
  assert.equal(state.ErrUnexpectedEOF.Error(), "unexpected EOF");
  assert.equal(state.ErrNoProgress.Error(), "multiple Read calls return no data or error");
  assert.notEqual(state.EOF, state.ErrUnexpectedEOF);
});

test("ReadFull handles exact EOF and reports unexpected EOF for short input", () => {
  const exact = RuntimeSlice.make<uint8>(4, 4, 0);
  const [exactCount, exactFailure] = ReadFull(
    new MemoryReader(Uint8Array.of(1, 2, 3, 4)),
    exact,
  );
  assert.equal(exactCount, 4);
  assert.equal(exactFailure, undefined);
  assert.deepEqual([...bytes(exact)], [1, 2, 3, 4]);

  const short = RuntimeSlice.make<uint8>(4, 4, 0);
  const [shortCount, shortFailure] = ReadFull(
    new MemoryReader(Uint8Array.of(1, 2, 3)),
    short,
  );
  assert.equal(shortCount, 3);
  assert.equal(shortFailure, state.ErrUnexpectedEOF);
});

test("Discard accepts all bytes", () => {
  const source = byteSlice([1, 2, 3]);
  assert.deepEqual(state.Discard.Write(source), [3, undefined]);
});

test("buffered reader preserves delimiter and EOF behavior", () => {
  const reader = NewReader(new MemoryReader(new TextEncoder().encode("one\ntwo"), 2));
  assert.notEqual(reader, undefined);

  const [first, firstFailure] = BufferedReader.ReadBytes(reader, 10);
  assert.equal(new TextDecoder().decode(bytes(first)), "one\n");
  assert.equal(firstFailure, undefined);

  const [second, secondFailure] = BufferedReader.ReadBytes(reader, 10);
  assert.equal(new TextDecoder().decode(bytes(second)), "two");
  assert.equal(secondFailure, state.EOF);
});

test("buffered writer delays writes until Flush", () => {
  const target = new MemoryWriter();
  const writer = NewWriter(target);
  assert.notEqual(writer, undefined);

  assert.deepEqual(BufferedWriter.Write(writer, byteSlice([1, 2, 3])), [3, undefined]);
  assert.deepEqual(target.values, []);
  assert.equal(BufferedWriter.WriteByte(writer, 4), undefined);
  assert.equal(BufferedWriter.Flush(writer), undefined);
  assert.deepEqual(target.values, [1, 2, 3, 4]);
});

test("buffered writer converts an unexplained partial write to short write", () => {
  const writer = NewWriter(new ShortWriter());
  assert.notEqual(writer, undefined);
  assert.deepEqual(BufferedWriter.Write(writer, byteSlice([1, 2, 3])), [3, undefined]);
  assert.equal(BufferedWriter.Flush(writer)?.Error(), "short write");
});
