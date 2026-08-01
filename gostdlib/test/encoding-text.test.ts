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

test("base64 append operations preserve prefixes and URL encoding", (): void => {
  const standard = requireEncoding(state.StdEncoding);
  const url = requireEncoding(state.URLEncoding);
  assert.equal(
    byteText(Encoding.AppendEncode(standard, byteTextSlice("x"), byteTextSlice("foo"))),
    "xZm9v",
  );
  assert.equal(
    byteText(Encoding.AppendEncode(url, byteSlice([0x78]), byteSlice([0xff, 0xef]))),
    "x_-8=",
  );
  const [decoded, failure] = Encoding.AppendDecode(
    standard,
    byteTextSlice("x"),
    byteTextSlice("Zm9v"),
  );
  assert.equal(failure, undefined);
  assert.equal(byteText(decoded), "xfoo");
});

test("base64 append operations agree with Go on corrupt input", (): void => {
  const directory = mkdtempSync(join(tmpdir(), "gotots-base64-append-"));
  const source = join(directory, "main.go");
  try {
    writeFileSync(source, base64AppendGoProgram);
    const result = spawnSync("go", ["run", source], { encoding: "utf8" });
    assert.equal(result.status, 0, result.stderr);
    assert.equal(base64AppendProviderResult(), result.stdout.trim());
  } finally {
    rmSync(directory, { force: true, recursive: true });
  }
});

test("hex encodes every byte with lower-case digits", () => {
  assert.equal(EncodeHex(RuntimeSlice.literal([0x00, 0x0f, 0x10, 0xff])), "000f10ff");
});

class CapturingWriter extends GoInterfaceValue implements Writer {
  readonly $go$type: object = CapturingWriter;
  readonly $go$methods: ReadonlySet<object> = new Set<object>();
  readonly $go$formatString = false;
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

  $go$format(): string {
    return "capturing writer";
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

function base64AppendProviderResult(): string {
  const standard = requireEncoding(state.StdEncoding);
  const url = requireEncoding(state.URLEncoding);
  const encoded = Encoding.AppendEncode(
    url,
    byteTextSlice("x"),
    byteSlice([0xff, 0xef]),
  );
  const [decoded, decodedFailure] = Encoding.AppendDecode(
    standard,
    byteTextSlice("x"),
    byteTextSlice("Zm9v"),
  );
  const [partial, partialFailure] = Encoding.AppendDecode(
    standard,
    byteTextSlice("x"),
    byteTextSlice("Zm9v!"),
  );
  return [
    byteText(encoded),
    `${byteText(decoded)}:${decodedFailure?.Error() ?? ""}`,
    `${byteText(partial)}:${partialFailure?.Error() ?? ""}`,
  ].join("|");
}

function byteSlice(values: readonly number[]): RuntimeSlice<number> {
  return RuntimeSlice.literal([...values]);
}

function byteTextSlice(value: string): RuntimeSlice<number> {
  return byteSlice([...value].map((character): number => character.charCodeAt(0)));
}

function byteText(value: RuntimeSlice<number>): string {
  return String.fromCharCode(...sliceValues(value));
}

const base64AppendGoProgram = `
package main

import (
  "encoding/base64"
  "fmt"
)

func errorText(err error) string {
  if err == nil {
    return ""
  }
  return err.Error()
}

func main() {
  encoded := base64.URLEncoding.AppendEncode([]byte("x"), []byte{0xff, 0xef})
  decoded, decodedErr := base64.StdEncoding.AppendDecode([]byte("x"), []byte("Zm9v"))
  partial, partialErr := base64.StdEncoding.AppendDecode([]byte("x"), []byte("Zm9v!"))
  fmt.Printf("%s|%s:%s|%s:%s\\n", encoded, decoded, errorText(decodedErr), partial, errorText(partialErr))
}
`;
