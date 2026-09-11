import { GoString } from "@gotots/runtime/string-value.js";
import { textValues } from "./text.js";
import assert from "node:assert/strict";
import { Buffer } from "node:buffer";
import test from "node:test";

import {
  Compile,
  MustCompile,
  Regexp,
} from "../src/regexp.js";
import { RegexpValueOperations } from "../src/internal/facets/provider-regexp.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

test("regexp compiles selected RE2 forms and returns submatches", () => {
  const expression = MustCompile(GoString.fromText("(?i)^([a-z]+)-([0-9]+)$"));
  assert.equal(Regexp.MatchString(expression, GoString.fromText("GoToTS-42")), true);
  assert.deepEqual(
    textValues(sliceValues(Regexp.FindStringSubmatch(expression, GoString.fromText("GoToTS-42")))),
    ["GoToTS-42", "GoToTS", "42"],
  );

  const unicodeEscape = MustCompile(GoString.fromText("[^\\x{0130}]+"));
  assert.equal(Regexp.MatchString(unicodeEscape, GoString.fromText("abc")), true);
  assert.deepEqual(
    textValues(sliceValues(Regexp.FindStringSubmatch(MustCompile(GoString.fromText("{(\\d+)}")), GoString.fromText("{42}")))),
    ["{42}", "42"],
  );

  assert.equal(Regexp.MatchString(MustCompile(GoString.fromText("[\\s\\S]+")), GoString.fromText("a\n")), true);
  assert.equal(Regexp.MatchString(MustCompile(GoString.fromText("^\\s+$")), GoString.fromText(goText("\u00a0"))), false);
  assert.equal(Regexp.MatchString(MustCompile(GoString.fromText("^[\\w,\\s-]+$")), GoString.fromText("A_,- ")), true);
});

test("regexp replacement expands captures and invokes callbacks directly", () => {
  const words = MustCompile(GoString.fromText("([a-z]+)=([0-9]+)"));
  assert.equal(
    (Regexp.ReplaceAllString(words, GoString.fromText("a=1 b=2"), GoString.fromText("$2:$1")))?.text(),
    "1:a 2:b",
  );
  assert.equal(
    (Regexp.ReplaceAllStringFunc(
      MustCompile(GoString.fromText("[0-9]+")),
      GoString.fromText("a1b22"),
      (match) => GoString.fromText(`[${match.text()}]`),
    ))?.text(),
    "a[1]b[22]",
  );
});

test("regexp Split and compile failures follow Go result shapes", () => {
  assert.deepEqual(
    textValues(sliceValues(Regexp.Split(MustCompile(GoString.fromText("\\s+")), GoString.fromText("a b  c"), -1n))),
    ["a", "b", "c"],
  );
  const [regexp, failure] = Compile(GoString.fromText("(?=x)"));
  assert.equal(regexp, undefined);
  assert.notEqual(failure, undefined);
  assert.throws(() => MustCompile(GoString.fromText("(")));
});

test("regexp value operations preserve independent Go assignment targets", () => {
  const source = MustCompile(GoString.fromText("^source$"));
  const target = MustCompile(GoString.fromText("^target$"));
  const copied = RegexpValueOperations.$copy(source);

  RegexpValueOperations.$assign(target, source);

  assert.notEqual(copied, source);
  assert.notEqual(target, source);
  assert.equal(Regexp.MatchString(copied, GoString.fromText("source")), true);
  assert.equal(Regexp.MatchString(target, GoString.fromText("source")), true);
  assert.equal(Regexp.MatchString(target, GoString.fromText("target")), false);
});

function goText(value: string): string {
  return Buffer.from(value, "utf8").toString("latin1");
}
