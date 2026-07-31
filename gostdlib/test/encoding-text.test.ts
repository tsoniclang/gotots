import assert from "node:assert/strict";
import test from "node:test";

import { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  Encoding,
  NewEncoder,
  state,
} from "../src/encoding/base64.js";
import { EncodeToString as EncodeHex } from "../src/encoding/hex.js";
import { sliceValues } from "../src/internal/runtime/slice.js";
import type { Writer } from "../src/io.js";

test("base64 standard encoding round-trips bytes and rejects corrupt input", () => {
  const encoding = requireEncoding(state.StdEncoding);
  const source = RuntimeSlice.literal([0x66, 0x6f, 0x6f]);
  assert.equal(Encoding.EncodeToString(encoding, source), "Zm9v");
  assert.equal(Encoding.EncodedLen(encoding, 4), 8);
  const [decoded, failure] = Encoding.DecodeString(encoding, "Zm9v");
  assert.equal(failure, undefined);
  assert.deepEqual(sliceValues(decoded), sliceValues(source));

  const [partial, corrupt] = Encoding.DecodeString(encoding, "Zm8=");
  assert.equal(corrupt, undefined);
  assert.deepEqual(sliceValues(partial), [0x66, 0x6f]);
  const [, missingPadding] = Encoding.DecodeString(encoding, "Zg=");
  assert.notEqual(missingPadding, undefined);
  assert.equal(missingPadding?.Error(), "illegal base64 data at input byte 3");
  const [trailing, trailingFailure] = Encoding.DecodeString(encoding, "Zg===");
  assert.deepEqual(sliceValues(trailing), [0x66]);
  assert.equal(trailingFailure?.Error(), "illegal base64 data at input byte 4");
  const [, shortFailure] = Encoding.DecodeString(encoding, "Zg");
  assert.equal(shortFailure?.Error(), "illegal base64 data at input byte 0");
});

test("base64 stream encoder flushes trailing bytes on Close", () => {
  const writer = new CapturingWriter();
  const encoder = NewEncoder(requireEncoding(state.StdEncoding), writer);
  assert.deepEqual(encoder.Write(RuntimeSlice.literal([0x66])), [1, undefined]);
  assert.equal(writer.text(), "");
  assert.equal(encoder.Close(), undefined);
  assert.equal(writer.text(), "Zg==");
  assert.deepEqual(encoder.Write(RuntimeSlice.literal([0x66, 0x6f, 0x6f])), [3, undefined]);
  assert.equal(writer.text(), "Zg==Zm9v");
});

test("hex encodes every byte with lower-case digits", () => {
  assert.equal(EncodeHex(RuntimeSlice.literal([0x00, 0x0f, 0x10, 0xff])), "000f10ff");
});

class CapturingWriter extends GoInterfaceValue implements Writer {
  readonly $go$type: object = CapturingWriter;
  readonly $go$methods: ReadonlySet<object> = new Set<object>();
  private readonly bytes: number[] = [];

  $go$implements(contract: readonly object[]): boolean {
    return contract.every((token) => this.$go$methods.has(token));
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    return 0;
  }

  Write(buffer: RuntimeSlice<number>): [number, GoError | undefined] {
    this.bytes.push(...sliceValues(buffer));
    return [buffer.length, undefined];
  }

  text(): string {
    return String.fromCharCode(...this.bytes);
  }
}

function requireEncoding(encoding: Encoding | undefined): Encoding {
  assert.notEqual(encoding, undefined);
  if (encoding === undefined) {
    throw new Error("missing standard encoding");
  }
  return encoding;
}
