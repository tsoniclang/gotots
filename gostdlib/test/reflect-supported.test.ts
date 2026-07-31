import assert from "node:assert/strict";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoPanic } from "@gotots/runtime/panic.js";

import {
  Array,
  Bool,
  Invalid,
  Float64,
  Int,
  Map,
  Slice,
  String,
  Struct,
  StructField,
  StructTag,
  Uint64,
  Value,
  ValueOf,
} from "../src/reflect.js";
import { ProviderError } from "../src/internal/runtime/error.js";

test("reflect kind values retain the selected Go numbering", () => {
  assert.equal(Bool.value, 1);
  assert.equal(Int.value, 2);
  assert.equal(Uint64.value, 11);
  assert.equal(Float64.value, 14);
  assert.equal(Array.value, 17);
  assert.equal(Map.value, 21);
  assert.equal(Slice.value, 23);
  assert.equal(String.value, 24);
  assert.equal(Struct.value, 25);
});

test("reflect StructField.IsExported uses package-path evidence", () => {
  const exported = new StructField({
    Name: "Name",
    PkgPath: "",
    Type: undefined,
    Tag: new StructTag(""),
    Offset: 0,
    Index: RuntimeSlice.literal([0]),
    Anonymous: false,
  });
  const privateField = new StructField({
    Name: "name",
    PkgPath: "example.com/project/model",
    Type: undefined,
    Tag: new StructTag(""),
    Offset: 0,
    Index: RuntimeSlice.literal([0]),
    Anonymous: false,
  });
  assert.equal(exported.IsExported(), true);
  assert.equal(privateField.IsExported(), false);
});

test("reflect StructTag.Get decodes Go quoted values", () => {
  const tag = new StructTag('json:"name,omitempty" xml:"line\\nvalue" octal:"\\141"');
  assert.equal(tag.Get("json"), "name,omitempty");
  assert.equal(tag.Get("xml"), "line\nvalue");
  assert.equal(tag.Get("octal"), "a");
  assert.equal(tag.Get("missing"), "");
});

test("reflect.ValueOf retains a typed interface descriptor", () => {
  assert.ok(ValueOf(new ProviderError("failure")) instanceof Value);
  const invalid = ValueOf(undefined);
  assert.ok(invalid instanceof Value);
  assert.equal(invalid.IsValid(), false);
  assert.equal(invalid.Kind(), Invalid);
  assert.equal(invalid.String(), "<invalid Value>");
});

test("unimplemented reflect operations fail explicitly", () => {
  assert.throws(
    () => ValueOf(new ProviderError("failure")).Kind(),
    (failure): boolean => {
      assert.ok(failure instanceof GoPanic);
      assert.match(
        failure.value.$go$format("v", "", undefined),
        /reflect\.Value\.Kind requires generated reflection metadata/,
      );
      return true;
    },
  );
});
