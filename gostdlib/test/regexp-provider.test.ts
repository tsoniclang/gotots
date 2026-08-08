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
  const expression = MustCompile("(?i)^([a-z]+)-([0-9]+)$");
  assert.equal(Regexp.MatchString(expression, "GoToTS-42"), true);
  assert.deepEqual(
    sliceValues(Regexp.FindStringSubmatch(expression, "GoToTS-42")),
    ["GoToTS-42", "GoToTS", "42"],
  );

  const unicodeEscape = MustCompile("[^\\x{0130}]+");
  assert.equal(Regexp.MatchString(unicodeEscape, "abc"), true);
  assert.deepEqual(
    sliceValues(Regexp.FindStringSubmatch(MustCompile("{(\\d+)}"), "{42}")),
    ["{42}", "42"],
  );

  assert.equal(Regexp.MatchString(MustCompile("[\\s\\S]+"), "a\n"), true);
  assert.equal(Regexp.MatchString(MustCompile("^\\s+$"), goText("\u00a0")), false);
  assert.equal(Regexp.MatchString(MustCompile("^[\\w,\\s-]+$"), "A_,- "), true);
});

test("regexp replacement expands captures and invokes callbacks directly", () => {
  const words = MustCompile("([a-z]+)=([0-9]+)");
  assert.equal(
    Regexp.ReplaceAllString(words, "a=1 b=2", "$2:$1"),
    "1:a 2:b",
  );
  assert.equal(
    Regexp.ReplaceAllStringFunc(
      MustCompile("[0-9]+"),
      "a1b22",
      (match) => `[${match}]`,
    ),
    "a[1]b[22]",
  );
});

test("regexp Split and compile failures follow Go result shapes", () => {
  assert.deepEqual(
    sliceValues(Regexp.Split(MustCompile("\\s+"), "a b  c", -1n)),
    ["a", "b", "c"],
  );
  const [regexp, failure] = Compile("(?=x)");
  assert.equal(regexp, undefined);
  assert.notEqual(failure, undefined);
  assert.throws(() => MustCompile("("));
});

test("regexp value operations preserve independent Go assignment targets", () => {
  const source = MustCompile("^source$");
  const target = MustCompile("^target$");
  const copied = RegexpValueOperations.$copy(source);

  RegexpValueOperations.$assign(target, source);

  assert.notEqual(copied, source);
  assert.notEqual(target, source);
  assert.equal(Regexp.MatchString(copied, "source"), true);
  assert.equal(Regexp.MatchString(target, "source"), true);
  assert.equal(Regexp.MatchString(target, "target"), false);
});

function goText(value: string): string {
  return Buffer.from(value, "utf8").toString("latin1");
}
