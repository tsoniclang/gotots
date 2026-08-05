import assert from "node:assert/strict";
import test from "node:test";

import {
  Kind,
  NameOff,
  TFlag,
  Type,
  TypeOff,
  UncommonType,
} from "../src/internal/abi.js";

test("internal/abi Type preserves selected descriptor fields", () => {
  const descriptor = new Type({
    Size_: 16n,
    PtrBytes: 8n,
    Hash: 0x1234,
    TFlag: new TFlag(1),
    Align_: 8,
    FieldAlign_: 8,
    Kind_: new Kind(25),
    Equal: undefined,
    GCData: undefined,
    Str: new NameOff(7),
    PtrToThis: new TypeOff(9),
  });
  assert.equal(descriptor.Size_, 16n);
  assert.equal(descriptor.Kind_.value, 25);
  assert.equal(descriptor.Str.value, 7);
  assert.equal(descriptor.PtrToThis.value, 9);
});

test("internal/abi UncommonType preserves method-table offsets", () => {
  const uncommon = new UncommonType({
    PkgPath: new NameOff(3),
    Mcount: 5,
    Xcount: 2,
    Moff: 64,
  });
  assert.equal(uncommon.PkgPath.value, 3);
  assert.equal(uncommon.Mcount, 5);
  assert.equal(uncommon.Xcount, 2);
  assert.equal(uncommon.Moff, 64);
});
