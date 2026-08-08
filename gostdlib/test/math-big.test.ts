import assert from "node:assert/strict";
import test from "node:test";

import { Accuracy, Float, Int, NewInt } from "../src/math/big.js";
import {
  MathBigFloatOperations,
  MathBigIntOperations,
} from "../src/internal/facets/named-math-big.js";

test("math/big parses and formats selected integer bases", (): void => {
  const receiver = new Int();
  assert.deepEqual(Int.SetString(receiver, "0x7fff_ffff_ffff_ffff", 0n), [receiver, true]);
  assert.equal(Int.String(receiver), "9223372036854775807");
  assert.deepEqual(Int.SetString(receiver, "12z", 10n), [undefined, false]);
});

test("math/big exponentiation mutates and returns its receiver", (): void => {
  const receiver = new Int();
  const result = Int.Exp(receiver, NewInt(7n), NewInt(5n), NewInt(13n));
  assert.equal(result, receiver);
  assert.equal(Int.String(receiver), "11");
});

test("math/big reports integer to float accuracy", (): void => {
  const exact = Int.Float64(NewInt(1_048_576n));
  assert.equal(exact[0], 1_048_576);
  assert.equal(exact[1].value, 0);

  const source = new Int();
  assert.deepEqual(Int.SetString(source, "9007199254740993", 10n), [source, true]);
  const rounded = Int.Float64(source);
  assert.equal(rounded[0], 9_007_199_254_740_992);
  assert.equal(rounded[1].value, -1);
});

test("math/big Float selected methods retain receiver identity", (): void => {
  const receiver = new Float();
  assert.equal(Float.SetPrec(receiver, 256n), receiver);
  assert.equal(Float.SetInt(receiver, NewInt(42n)), receiver);
  assert.deepEqual(Float.Float64(receiver), [42, new Accuracy(0)]);
});

test("math/big value operations preserve independent Go struct copies", (): void => {
  const integer = NewInt(41n);
  assert.ok(integer !== undefined);
  const copiedInteger = MathBigIntOperations.$copy(integer);
  const assignedInteger = new Int();
  MathBigIntOperations.$assign(assignedInteger, integer);
  Int.SetString(integer, "99", 10n);
  assert.equal(Int.String(copiedInteger), "41");
  assert.equal(Int.String(assignedInteger), "41");

  const floating = new Float();
  Float.SetPrec(floating, 64n);
  Float.SetInt(floating, NewInt(17n));
  const copiedFloat = MathBigFloatOperations.$copy(floating);
  const assignedFloat = new Float();
  MathBigFloatOperations.$assign(assignedFloat, floating);
  Float.SetInt(floating, NewInt(23n));
  assert.equal(Float.Float64(copiedFloat)[0], 17);
  assert.equal(Float.Float64(assignedFloat)[0], 17);
});
