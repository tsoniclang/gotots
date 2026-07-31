import assert from "node:assert/strict";
import test from "node:test";

import * as base64 from "../src/encoding/base64.js";
import * as hex from "../src/encoding/hex.js";
import * as bytes from "../src/bytes.js";
import * as path from "../src/path.js";
import * as filepath from "../src/path/filepath.js";
import * as regexp from "../src/regexp.js";
import * as strconv from "../src/strconv.js";
import * as strings from "../src/strings.js";
import * as unicode from "../src/unicode.js";
import * as utf16 from "../src/unicode/utf16.js";
import * as utf8 from "../src/unicode/utf8.js";

test("text provider modules expose exactly the clean selected Go names", () => {
  assertExports(bytes, ["Buffer", "Cut", "Equal", "NewBuffer", "TrimSpace"]);
  assertExports(path, ["Join"]);
  assertExports(filepath, [
    "Abs",
    "Clean",
    "Dir",
    "EvalSymlinks",
    "FromSlash",
    "IsAbs",
    "Join",
    "Separator",
  ]);
  assertExports(regexp, ["Compile", "MustCompile", "Regexp"]);
  assertExports(strconv, [
    "Atoi",
    "FormatInt",
    "FormatUint",
    "Itoa",
    "ParseFloat",
    "ParseInt",
    "ParseUint",
    "state",
  ]);
  assertExports(base64, ["Encoding", "NewEncoder", "state"]);
  assertExports(hex, ["EncodeToString"]);
  assertExports(unicode, [
    "Is",
    "IsDigit",
    "IsLower",
    "IsSpace",
    "IsUpper",
    "MaxASCII",
    "Range16",
    "Range32",
    "RangeTable",
    "ToLower",
    "ToUpper",
    "state",
  ]);
  assertExports(utf8, [
    "DecodeLastRuneInString",
    "DecodeRuneInString",
    "RuneError",
    "RuneSelf",
  ]);
  assertExports(utf16, [
    "Decode",
    "DecodeRune",
    "EncodeRune",
    "IsSurrogate",
    "RuneLen",
  ]);
});

test("strings exports every selected clean declaration exactly once", () => {
  assertExports(strings, [
    "Builder",
    "Clone",
    "Compare",
    "Contains",
    "ContainsAny",
    "ContainsRune",
    "Count",
    "Cut",
    "CutPrefix",
    "CutSuffix",
    "EqualFold",
    "HasPrefix",
    "HasSuffix",
    "Index",
    "IndexAny",
    "IndexByte",
    "IndexFunc",
    "Join",
    "LastIndex",
    "LastIndexByte",
    "LastIndexFunc",
    "Lines",
    "Map",
    "NewReader",
    "NewReplacer",
    "Reader",
    "Repeat",
    "Replace",
    "ReplaceAll",
    "Replacer",
    "Split",
    "ToLower",
    "ToUpper",
    "ToValidUTF8",
    "Trim",
    "TrimFunc",
    "TrimLeft",
    "TrimLeftFunc",
    "TrimPrefix",
    "TrimRight",
    "TrimRightFunc",
    "TrimSpace",
    "TrimSuffix",
  ]);
});

test("receiver operations are class-owned static members", () => {
  assert.deepEqual(
    Object.getOwnPropertyNames(strings.Builder)
      .filter((name) => !["length", "name", "prototype"].includes(name))
      .sort(),
    ["Grow", "Len", "Reset", "String", "Write", "WriteByte", "WriteRune", "WriteString"].sort(),
  );
  assert.deepEqual(
    Object.getOwnPropertyNames(strings.Reader)
      .filter((name) => !["length", "name", "prototype"].includes(name)),
    ["Read"],
  );
  assert.deepEqual(
    Object.getOwnPropertyNames(strings.Replacer)
      .filter((name) => !["length", "name", "prototype"].includes(name)),
    ["Replace"],
  );
  assert.deepEqual(
    Object.getOwnPropertyNames(base64.Encoding)
      .filter((name) => !["length", "name", "prototype"].includes(name))
      .sort(),
    ["DecodeString", "EncodeToString", "EncodedLen"].sort(),
  );
  assert.deepEqual(
    Object.getOwnPropertyNames(regexp.Regexp)
      .filter((name) => !["length", "name", "prototype"].includes(name))
      .sort(),
    ["FindStringSubmatch", "MatchString", "ReplaceAllString", "ReplaceAllStringFunc", "Split"].sort(),
  );
  for (const type of [
    strings.Builder,
    strings.Reader,
    strings.Replacer,
    base64.Encoding,
    regexp.Regexp,
  ]) {
    assert.deepEqual(Object.getOwnPropertyNames(type.prototype), ["constructor"]);
  }
});

function assertExports(module: object, expected: readonly string[]): void {
  assert.deepEqual(Object.keys(module).sort(), [...expected].sort());
}
