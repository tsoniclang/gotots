import assert from "node:assert/strict";
import test from "node:test";

import * as base64 from "../src/encoding/base64.js";
import * as base32 from "../src/encoding/base32.js";
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
  assertExports(bytes, [
    "Buffer",
    "Clone",
    "Compare",
    "Cut",
    "Equal",
    "IndexAny",
    "IndexByte",
    "Join",
    "LastIndexByte",
    "NewBuffer",
    "Trim",
    "TrimLeft",
    "TrimRight",
    "TrimSpace",
  ]);
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
    "AppendBool",
    "AppendFloat",
    "AppendInt",
    "AppendQuote",
    "AppendUint",
    "Atoi",
    "FormatInt",
    "FormatUint",
    "Itoa",
    "ParseFloat",
    "ParseInt",
    "ParseUint",
    "Quote",
    "QuoteRune",
    "Unquote",
    "state",
  ]);
  assertExports(base64, ["Encoding", "NewEncoder", "state"]);
  assertExports(base32, ["Encoding", "state"]);
  assertExports(hex, [
    "AppendDecode",
    "AppendEncode",
    "EncodedLen",
    "EncodeToString",
  ]);
  assertExports(unicode, [
    "In",
    "Is",
    "IsDigit",
    "IsLetter",
    "IsLower",
    "IsNumber",
    "IsPrint",
    "IsSpace",
    "IsUpper",
    "MaxASCII",
    "Range16",
    "Range32",
    "RangeTable",
    "SimpleFold",
    "ToLower",
    "ToUpper",
    "state",
  ]);
  assertExports(utf8, [
    "AppendRune",
    "DecodeRune",
    "DecodeLastRuneInString",
    "DecodeRuneInString",
    "EncodeRune",
    "FullRune",
    "RuneError",
    "RuneCount",
    "RuneSelf",
    "RuneStart",
    "UTFMax",
    "ValidString",
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
    "ContainsFunc",
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
    "IndexRune",
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
    ["AppendDecode", "AppendEncode", "DecodeString", "EncodeToString", "EncodedLen"].sort(),
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
    base32.Encoding,
    regexp.Regexp,
  ]) {
    assert.deepEqual(Object.getOwnPropertyNames(type.prototype), ["constructor"]);
  }
  assert.deepEqual(
    Object.getOwnPropertyNames(base32.Encoding)
      .filter((name) => !["length", "name", "prototype"].includes(name))
      .sort(),
    ["AppendDecode", "AppendEncode", "EncodedLen"].sort(),
  );
  assert.deepEqual(
    Object.getOwnPropertyNames(bytes.Buffer)
      .filter((name) => !["length", "name", "prototype"].includes(name))
      .sort(),
    ["Available", "AvailableBuffer", "Grow", "Len", "Next", "Read", "Write"].sort(),
  );
});

function assertExports(module: object, expected: readonly string[]): void {
  assert.deepEqual(Object.keys(module).sort(), [...expected].sort());
}
