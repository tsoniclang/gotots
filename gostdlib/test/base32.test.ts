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
  Encoding,
  state,
} from "../src/encoding/base32.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

test("base32 appends standard and hexadecimal encodings", (): void => {
  const standard = requireEncoding(state.StdEncoding);
  const hexadecimal = requireEncoding(state.HexEncoding);
  const source = bytes("foobar");

  assert.equal(Encoding.EncodedLen(standard, BigInt(source.length)), 16n);
  assert.equal(text(Encoding.AppendEncode(standard, bytes("x"), source)), "xMZXW6YTBOI======");
  assert.equal(text(Encoding.AppendEncode(hexadecimal, bytes("x"), source)), "xCPNMUOJ1E8======");

  const [decoded, failure] = Encoding.AppendDecode(
    standard,
    bytes("x"),
    bytes("MZXW6YTBOI======"),
  );
  assert.equal(failure, undefined);
  assert.equal(text(decoded), "xfoobar");
});

test("base32 append family agrees with Go on partial failures", (): void => {
  const directory = mkdtempSync(join(tmpdir(), "gotots-base32-"));
  const source = join(directory, "main.go");
  try {
    writeFileSync(source, base32GoProgram);
    const result = spawnSync("go", ["run", source], { encoding: "utf8" });
    assert.equal(result.status, 0, result.stderr);
    assert.equal(providerResult(), result.stdout.trim());
  } finally {
    rmSync(directory, { force: true, recursive: true });
  }
});

function providerResult(): string {
  const encoding = requireEncoding(state.StdEncoding);
  const encoded = Encoding.AppendEncode(encoding, bytes("x"), bytes("foobar"));
  const [decoded, decodedFailure] = Encoding.AppendDecode(
    encoding,
    bytes("x"),
    bytes("MZXW6YTBOI======"),
  );
  const [partial, partialFailure] = Encoding.AppendDecode(
    encoding,
    bytes("x"),
    bytes("MZXW6YTB!"),
  );
  const [short, shortFailure] = Encoding.AppendDecode(
    encoding,
    bytes("x"),
    bytes("M"),
  );
  return [
    text(encoded),
    `${text(decoded)}:${decodedFailure?.Error() ?? ""}`,
    `${text(partial)}:${partialFailure?.Error() ?? ""}`,
    `${text(short)}:${shortFailure?.Error() ?? ""}`,
  ].join("|");
}

function requireEncoding(encoding: Encoding | undefined): Encoding {
  assert.notEqual(encoding, undefined);
  if (encoding === undefined) {
    throw new Error("missing base32 encoding");
  }
  return encoding;
}

function bytes(value: string): RuntimeSlice<number> {
  return RuntimeSlice.literal([...value].map((character): number => character.charCodeAt(0)));
}

function text(value: RuntimeSlice<number>): string {
  return String.fromCharCode(...sliceValues(value));
}

const base32GoProgram = `
package main

import (
  "encoding/base32"
  "fmt"
)

func errorText(err error) string {
  if err == nil {
    return ""
  }
  return err.Error()
}

func main() {
  encoding := base32.StdEncoding
  encoded := encoding.AppendEncode([]byte("x"), []byte("foobar"))
  decoded, decodedErr := encoding.AppendDecode([]byte("x"), []byte("MZXW6YTBOI======"))
  partial, partialErr := encoding.AppendDecode([]byte("x"), []byte("MZXW6YTB!"))
  short, shortErr := encoding.AppendDecode([]byte("x"), []byte("M"))
  fmt.Printf("%s|%s:%s|%s:%s|%s:%s\\n", encoded, decoded, errorText(decodedErr), partial, errorText(partialErr), short, errorText(shortErr))
}
`;
