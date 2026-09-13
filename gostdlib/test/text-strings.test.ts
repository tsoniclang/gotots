import { textValues } from "./text.js";
import { GoString } from "@gotots/runtime/string-value.js";
import assert from "node:assert/strict";
import { Buffer } from "node:buffer";
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
import { StringsReplacerOperations } from "../src/internal/facets/named-strings.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

import {
  Builder,
  Clone,
  Compare,
  Contains,
  ContainsFunc,
  ContainsAny,
  ContainsRune,
  Count,
  Cut,
  CutPrefix,
  CutSuffix,
  EqualFold,
  HasPrefix,
  HasSuffix,
  Index,
  IndexAny,
  IndexByte,
  IndexFunc,
  IndexRune,
  Join,
  LastIndex,
  LastIndexByte,
  LastIndexFunc,
  Lines,
  Map,
  NewReader,
  NewReplacer,
  Reader,
  Repeat,
  Replace,
  ReplaceAll,
  Replacer,
  Split,
  SplitN,
  ToLower,
  ToUpper,
  ToValidUTF8,
  Trim,
  TrimFunc,
  TrimLeft,
  TrimLeftFunc,
  TrimPrefix,
  TrimRight,
  TrimRightFunc,
  TrimSpace,
  TrimSuffix,
} from "../src/strings.js";
import { state as ioState } from "../src/io.js";

test("strings operate on Go UTF-8 bytes rather than JavaScript UTF-16 indexes", () => {
  const text = goText("AéΣ");
  assert.equal(text.length, 5);
  assert.equal(ContainsRune(GoString.fromText(text), 0x03a3), true);
  assert.equal(ContainsAny(GoString.fromText(text), GoString.fromText(goText("λΣ"))), true);
  assert.equal(IndexAny(GoString.fromText(text), GoString.fromText(goText("Σ"))), 3n);
  assert.equal(IndexRune(GoString.fromText(text), 0x03a3), 3n);
  assert.equal(Count(GoString.fromText(text), GoString.fromText("")), 4n);
  assert.deepEqual(textValues(Cut(GoString.fromText(text), GoString.fromText(goText("é")))), ["A", goText("Σ"), true]);
  assert.equal(EqualFold(GoString.fromText(goText("K")), GoString.fromText(goText("K"))), true);
  assert.equal(EqualFold(GoString.fromText(goText("Σ")), GoString.fromText(goText("ς"))), true);
});

test("strings IndexRune agrees with Go on malformed UTF-8", (): void => {
  const directory = mkdtempSync(join(tmpdir(), "gotots-index-rune-"));
  const source = join(directory, "main.go");
  try {
    writeFileSync(source, indexRuneGoProgram);
    const result = spawnSync("go", ["run", source], { encoding: "utf8" });
    assert.equal(result.status, 0, result.stderr);
    const malformed = String.fromCharCode(0x61, 0xff, 0xfe, 0x62);
    const provider = [
      IndexRune(GoString.fromText(goText("AéΣ")), 0x03a3),
      IndexRune(GoString.fromText(malformed), 0xfffd),
      IndexRune(GoString.fromText(goText("AéΣ")), -1),
      IndexRune(GoString.fromText(goText("AéΣ")), 0x03bb),
    ].join(",");
    assert.equal(provider, result.stdout.trim());
  } finally {
    rmSync(directory, { force: true, recursive: true });
  }
});

test("strings transformations preserve invalid bytes and simple Unicode case", () => {
  assert.equal(hostText((ToLower(GoString.fromText(goText("İKΣ"))))?.text()), "ikσ");
  assert.equal(hostText((ToUpper(GoString.fromText(goText("µſ"))))?.text()), "ΜS");
  assert.equal(
    (ToValidUTF8(GoString.fromText(String.fromCharCode(0xff, 0xfe, 0x41, 0xff)), GoString.fromText("?")))?.text(),
    "?A?",
  );
  assert.equal(
    (Map((rune) => rune === 0x61 ? -1 : rune, GoString.fromText("banana")))?.text(),
    "bnn",
  );
  assert.equal((TrimSpace(GoString.fromText(goText("\u3000 value \u00a0"))))?.text(), "value");
});

test("strings selected search, join, replacement, and trim functions retain boundaries", () => {
  assert.equal((Clone(GoString.fromText("text")))?.text(), "text");
  assert.equal(Compare(GoString.fromText("a"), GoString.fromText("b")), -1n);
  assert.equal(Compare(GoString.fromText("b"), GoString.fromText("a")), 1n);
  assert.equal(Compare(GoString.fromText("a"), GoString.fromText("a")), 0n);
  assert.equal(Contains(GoString.fromText("abc"), GoString.fromText("bc")), true);
  assert.equal(HasPrefix(GoString.fromText("abc"), GoString.fromText("ab")), true);
  assert.equal(HasSuffix(GoString.fromText("abc"), GoString.fromText("bc")), true);
  assert.equal(Index(GoString.fromText("ababa"), GoString.fromText("ba")), 1n);
  assert.equal(IndexByte(GoString.fromText("ab"), 0x62), 1n);
  assert.equal(IndexFunc(GoString.fromText("abc"), (rune) => rune === 0x62), 1n);
  assert.equal(IndexFunc(GoString.fromText(""), undefined), -1n);
  assert.throws(() => IndexFunc(GoString.fromText("x"), undefined));
  assert.equal(ContainsFunc(GoString.fromText(goText("a世界")), (rune) => rune === 0x4e16), true);
  assert.equal(ContainsFunc(GoString.fromText(""), undefined), false);
  assert.throws(() => ContainsFunc(GoString.fromText("x"), undefined));
  assert.equal(LastIndex(GoString.fromText("ababa"), GoString.fromText("ba")), 3n);
  assert.equal(LastIndexByte(GoString.fromText("aba"), 0x61), 2n);
  assert.equal(LastIndexFunc(GoString.fromText("abca"), (rune) => rune === 0x61), 3n);
  assert.deepEqual(textValues(CutPrefix(GoString.fromText("prefix-value"), GoString.fromText("prefix-"))), ["value", true]);
  assert.deepEqual(textValues(CutSuffix(GoString.fromText("value.suffix"), GoString.fromText(".suffix"))), ["value", true]);
  assert.equal((Join(RuntimeSlice.literal([GoString.fromText("a"), GoString.fromText("b"), GoString.fromText("c")]), GoString.fromText(":")))?.text(), "a:b:c");
  assert.equal((ReplaceAll(GoString.fromText("a-a-a"), GoString.fromText("a"), GoString.fromText("b")))?.text(), "b-b-b");
  assert.equal((Trim(GoString.fromText("xyvalueyx"), GoString.fromText("xy")))?.text(), "value");
  assert.equal((TrimLeft(GoString.fromText("xyvalue"), GoString.fromText("xy")))?.text(), "value");
  assert.equal((TrimRight(GoString.fromText("valuexy"), GoString.fromText("xy")))?.text(), "value");
  assert.equal((TrimFunc(GoString.fromText("123value456"), isDigit))?.text(), "value");
  assert.equal((TrimLeftFunc(GoString.fromText("123value"), isDigit))?.text(), "value");
  assert.equal((TrimRightFunc(GoString.fromText("value456"), isDigit))?.text(), "value");
  assert.equal((TrimPrefix(GoString.fromText("prefix-value"), GoString.fromText("prefix-")))?.text(), "value");
  assert.equal((TrimSuffix(GoString.fromText("value.suffix"), GoString.fromText(".suffix")))?.text(), "value");
});

test("strings splitting, replacement, and repetition follow Go boundaries", () => {
  assert.deepEqual(
    sliceValues(Split(GoString.fromText(goText("éΣ")), GoString.fromText(""))).map(value => hostText(value.text())),
    ["é", "Σ"],
  );
  assert.equal(SplitN(GoString.fromText("a,b"), GoString.fromText(","), 0n).isNil(), true);
  assert.deepEqual(textValues(sliceValues(SplitN(GoString.fromText("a,b,c"), GoString.fromText(","), 2n))), ["a", "b,c"]);
  assert.deepEqual(textValues(sliceValues(SplitN(GoString.fromText("a,b,c"), GoString.fromText(","), -1n))), ["a", "b", "c"]);
  assert.deepEqual(
    sliceValues(SplitN(GoString.fromText(goText("éΣx")), GoString.fromText(""), 2n)).map(value => hostText(value.text())),
    ["é", "Σx"],
  );
  assert.equal((Replace(GoString.fromText(goText("é")), GoString.fromText(""), GoString.fromText("."), -1n))?.text(), `.${goText("é")}.`);
  assert.equal((Replace(GoString.fromText("aaaa"), GoString.fromText("aa"), GoString.fromText("b"), 1n))?.text(), "baa");
  assert.equal((Repeat(GoString.fromText("ab"), 3n))?.text(), "ababab");
  assert.throws(() => Repeat(GoString.fromText("x"), -1n));
  const builder = new Builder();
  assert.throws(() => Builder.Grow(builder, -1n));
});

test("strings named types expose clean static receiver operations", () => {
  const builder = new Builder();
  assert.deepEqual(Builder.WriteString(builder, GoString.fromText("go")), [2n, undefined]);
  assert.deepEqual(Builder.WriteRune(builder, 0x00e9), [2n, undefined]);
  assert.equal(hostText((Builder.String(builder))?.text()), "goé");
  assert.equal(Builder.Len(builder), 4n);
  Builder.Reset(builder);
  assert.equal((Builder.String(builder))?.text(), "");

  const reader = NewReader(GoString.fromText("abc"));
  const buffer = RuntimeSlice.make<number>(2, 2, 0);
  assert.deepEqual(Reader.Read(reader, buffer), [2n, undefined]);
  assert.deepEqual(sliceValues(buffer), [0x61, 0x62]);
  assert.deepEqual(Reader.Read(reader, buffer), [1n, undefined]);
  const [count, end] = Reader.Read(reader, buffer);
  assert.equal(count, 0n);
  assert.equal(end, ioState.EOF);

  const replacer = NewReplacer(RuntimeSlice.literal([GoString.fromText(""), GoString.fromText("X"), GoString.fromText("a"), GoString.fromText("Y")]));
  assert.equal((Replacer.Replace(replacer, GoString.fromText("a")))?.text(), "XYX");
  assert.throws(() => NewReplacer(RuntimeSlice.literal([GoString.fromText("old")])));
});

test("Replacer value operations preserve shallow Go assignment", () => {
  const source = NewReplacer(RuntimeSlice.literal([GoString.fromText("a"), GoString.fromText("source")]));
  const originalTarget = NewReplacer(RuntimeSlice.literal([GoString.fromText("a"), GoString.fromText("target")]));
  const target = StringsReplacerOperations.$copy(originalTarget);

  StringsReplacerOperations.$assign(target, source);

  assert.equal((Replacer.Replace(target, GoString.fromText("a")))?.text(), "source");
  assert.equal((Replacer.Replace(originalTarget, GoString.fromText("a")))?.text(), "target");
});

test("strings Lines yields newline-preserving single-use values", () => {
  const sequence = Lines(GoString.fromText("first\nsecond"));
  const lines: string[] = [];
  const implementation = sequence.value;
  assert.notEqual(implementation, undefined);
  implementation?.((line) => {
    lines.push(line.text());
    return true;
  });
  assert.deepEqual(lines, ["first\n", "second"]);
});

function goText(value: string): string {
  return Buffer.from(value, "utf8").toString("latin1");
}

function hostText(value: string): string {
  return Buffer.from(value, "latin1").toString("utf8");
}

function isDigit(rune: number): boolean {
  return rune >= 0x30 && rune <= 0x39;
}

const indexRuneGoProgram = `
package main

import (
  "fmt"
  "strings"
  "unicode/utf8"
)

func main() {
  text := "AéΣ"
  malformed := string([]byte{'a', 0xff, 0xfe, 'b'})
  fmt.Printf("%d,%d,%d,%d\\n",
    strings.IndexRune(text, 'Σ'),
    strings.IndexRune(malformed, utf8.RuneError),
    strings.IndexRune(text, -1),
    strings.IndexRune(text, 'λ'),
  )
}
`;
