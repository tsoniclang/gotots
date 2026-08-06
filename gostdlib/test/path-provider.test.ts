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
  assert.equal(JoinPath(RuntimeSlice.literal(["a", ".", "b", "..", "c"])), "a/c");
  assert.equal(JoinPath(RuntimeSlice.literal(["", ""])), "");
  assert.equal(JoinPath(RuntimeSlice.literal(["", "/a"])), "/a");
});

test("filepath exposes the selected Unix lexical contract", () => {
  assert.equal(Separator, 0x2f);
  assert.equal(Clean("//a/./b/../c/"), "/a/c");
  assert.equal(Dir("/a/b.txt"), "/a");
  assert.equal(Ext("/a/b.txt"), ".txt");
  assert.equal(Ext("/a/.profile"), ".profile");
  assert.equal(Ext("/a/name."), ".");
  assert.equal(Ext("/a.b/name"), "");
  assert.equal(FromSlash("a/b"), "a/b");
  assert.equal(IsAbs("/a"), true);
  assert.equal(IsAbs("a"), false);
  assert.equal(Join(RuntimeSlice.literal(["", "a", "..", "b"])), "b");
});

test("filepath host operations return Go byte strings and typed errors", () => {
  const [absolute, absoluteError] = Abs(".");
  assert.equal(absoluteError, undefined);
  assert.equal(hostText(absolute), process.cwd());

  const [resolved, resolvedError] = EvalSymlinks(goText(process.cwd()));
  assert.equal(resolvedError, undefined);
  assert.equal(hostText(resolved), process.cwd());

  const [, missingError] = EvalSymlinks(goText(`${process.cwd()}/missing-gotots-entry`));
  assert.notEqual(missingError, undefined);
});

function goText(value: string): string {
  return Buffer.from(value, "utf8").toString("latin1");
}

function hostText(value: string): string {
  return Buffer.from(value, "latin1").toString("utf8");
}
