import { GoString } from "@gotots/runtime/string-value.js";
import assert from "node:assert/strict";
import { Buffer } from "node:buffer";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { Join as JoinPath } from "../src/path.js";
import {
  Abs,
  Clean,
  Dir,
  Ext,
  EvalSymlinks,
  FromSlash,
  IsAbs,
  Join,
  Separator,
} from "../src/path/filepath.js";

test("path joining applies Go slash-path cleaning", () => {
  assert.equal((JoinPath(RuntimeSlice.literal([GoString.fromText("a"), GoString.fromText("."), GoString.fromText("b"), GoString.fromText(".."), GoString.fromText("c")])))?.text(), "a/c");
  assert.equal((JoinPath(RuntimeSlice.literal([GoString.fromText(""), GoString.fromText("")])))?.text(), "");
  assert.equal((JoinPath(RuntimeSlice.literal([GoString.fromText(""), GoString.fromText("/a")])))?.text(), "/a");
});

test("filepath exposes the selected Unix lexical contract", () => {
  assert.equal(Separator, 0x2f);
  assert.equal((Clean(GoString.fromText("//a/./b/../c/")))?.text(), "/a/c");
  assert.equal((Dir(GoString.fromText("/a/b.txt")))?.text(), "/a");
  assert.equal((Ext(GoString.fromText("/a/b.txt")))?.text(), ".txt");
  assert.equal((Ext(GoString.fromText("/a/.profile")))?.text(), ".profile");
  assert.equal((Ext(GoString.fromText("/a/name.")))?.text(), ".");
  assert.equal((Ext(GoString.fromText("/a.b/name")))?.text(), "");
  assert.equal((FromSlash(GoString.fromText("a/b")))?.text(), "a/b");
  assert.equal(IsAbs(GoString.fromText("/a")), true);
  assert.equal(IsAbs(GoString.fromText("a")), false);
  assert.equal((Join(RuntimeSlice.literal([GoString.fromText(""), GoString.fromText("a"), GoString.fromText(".."), GoString.fromText("b")])))?.text(), "b");
});

test("filepath host operations return Go byte strings and typed errors", () => {
  const [absolute, absoluteError] = Abs(GoString.fromText("."));
  assert.equal(absoluteError, undefined);
  assert.equal(hostText((absolute)?.text()), process.cwd());

  const [resolved, resolvedError] = EvalSymlinks(GoString.fromText(goText(process.cwd())));
  assert.equal(resolvedError, undefined);
  assert.equal(hostText((resolved)?.text()), process.cwd());

  const [, missingError] = EvalSymlinks(GoString.fromText(goText(`${process.cwd()}/missing-gotots-entry`)));
  assert.notEqual(missingError, undefined);
});

function goText(value: string): string {
  return Buffer.from(value, "utf8").toString("latin1");
}

function hostText(value: string): string {
  return Buffer.from(value, "latin1").toString("utf8");
}
