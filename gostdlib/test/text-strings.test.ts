import assert from "node:assert/strict";
import { Buffer } from "node:buffer";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";
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
  assert.equal(ContainsRune(text, 0x03a3), true);
  assert.equal(ContainsAny(text, goText("λΣ")), true);
  assert.equal(IndexAny(text, goText("Σ")), 3);
  assert.equal(Count(text, ""), 4);
  assert.deepEqual(Cut(text, goText("é")), ["A", goText("Σ"), true]);
  assert.equal(EqualFold(goText("K"), goText("K")), true);
  assert.equal(EqualFold(goText("Σ"), goText("ς")), true);
});

test("strings transformations preserve invalid bytes and simple Unicode case", () => {
  assert.equal(hostText(ToLower(goText("İKΣ"))), "ikσ");
  assert.equal(hostText(ToUpper(goText("µſ"))), "ΜS");
  assert.equal(
    ToValidUTF8(String.fromCharCode(0xff, 0xfe, 0x41, 0xff), "?"),
    "?A?",
  );
  assert.equal(
    Map((rune) => rune === 0x61 ? -1 : rune, "banana"),
    "bnn",
  );
  assert.equal(TrimSpace(goText("\u3000 value \u00a0")), "value");
});

test("strings selected search, join, replacement, and trim functions retain boundaries", () => {
  assert.equal(Clone("text"), "text");
  assert.equal(Compare("a", "b"), -1);
  assert.equal(Compare("b", "a"), 1);
  assert.equal(Compare("a", "a"), 0);
  assert.equal(Contains("abc", "bc"), true);
  assert.equal(HasPrefix("abc", "ab"), true);
  assert.equal(HasSuffix("abc", "bc"), true);
  assert.equal(Index("ababa", "ba"), 1);
  assert.equal(IndexByte("ab", 0x62), 1);
  assert.equal(IndexFunc("abc", (rune) => rune === 0x62), 1);
  assert.equal(IndexFunc("", undefined), -1);
  assert.throws(() => IndexFunc("x", undefined));
  assert.equal(ContainsFunc(goText("a世界"), (rune) => rune === 0x4e16), true);
  assert.equal(ContainsFunc("", undefined), false);
  assert.throws(() => ContainsFunc("x", undefined));
  assert.equal(LastIndex("ababa", "ba"), 3);
  assert.equal(LastIndexByte("aba", 0x61), 2);
  assert.equal(LastIndexFunc("abca", (rune) => rune === 0x61), 3);
  assert.deepEqual(CutPrefix("prefix-value", "prefix-"), ["value", true]);
  assert.deepEqual(CutSuffix("value.suffix", ".suffix"), ["value", true]);
  assert.equal(Join(RuntimeSlice.literal(["a", "b", "c"]), ":"), "a:b:c");
  assert.equal(ReplaceAll("a-a-a", "a", "b"), "b-b-b");
  assert.equal(Trim("xyvalueyx", "xy"), "value");
  assert.equal(TrimLeft("xyvalue", "xy"), "value");
  assert.equal(TrimRight("valuexy", "xy"), "value");
  assert.equal(TrimFunc("123value456", isDigit), "value");
  assert.equal(TrimLeftFunc("123value", isDigit), "value");
  assert.equal(TrimRightFunc("value456", isDigit), "value");
  assert.equal(TrimPrefix("prefix-value", "prefix-"), "value");
  assert.equal(TrimSuffix("value.suffix", ".suffix"), "value");
});

test("strings splitting, replacement, and repetition follow Go boundaries", () => {
  assert.deepEqual(
    sliceValues(Split(goText("éΣ"), "")).map(hostText),
    ["é", "Σ"],
  );
  assert.equal(Replace(goText("é"), "", ".", -1), `.${goText("é")}.`);
  assert.equal(Replace("aaaa", "aa", "b", 1), "baa");
  assert.equal(Repeat("ab", 3), "ababab");
  assert.throws(() => Repeat("x", -1));
  const builder = new Builder();
  assert.throws(() => Builder.Grow(builder, -1));
});

test("strings named types expose clean static receiver operations", () => {
  const builder = new Builder();
  assert.deepEqual(Builder.WriteString(builder, "go"), [2, undefined]);
  assert.deepEqual(Builder.WriteRune(builder, 0x00e9), [2, undefined]);
  assert.equal(hostText(Builder.String(builder)), "goé");
  assert.equal(Builder.Len(builder), 4);
  Builder.Reset(builder);
  assert.equal(Builder.String(builder), "");

  const reader = NewReader("abc");
  const buffer = RuntimeSlice.make<number>(2, 2, 0);
  assert.deepEqual(Reader.Read(reader, buffer), [2, undefined]);
  assert.deepEqual(sliceValues(buffer), [0x61, 0x62]);
  assert.deepEqual(Reader.Read(reader, buffer), [1, undefined]);
  const [count, end] = Reader.Read(reader, buffer);
  assert.equal(count, 0);
  assert.equal(end, ioState.EOF);

  const replacer = NewReplacer(RuntimeSlice.literal(["", "X", "a", "Y"]));
  assert.equal(Replacer.Replace(replacer, "a"), "XYX");
  assert.throws(() => NewReplacer(RuntimeSlice.literal(["old"])));
});

test("strings Lines yields newline-preserving single-use values", async () => {
  const sequence = Lines("first\nsecond");
  const lines: string[] = [];
  const implementation = sequence.value;
  assert.notEqual(implementation, undefined);
  await implementation?.(async (line) => {
    lines.push(line);
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
