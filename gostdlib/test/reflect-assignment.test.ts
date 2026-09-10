import assert from "node:assert/strict";
import test from "node:test";

import { ValueOf } from "../src/reflect.js";
import { ReflectValueOperations } from "../src/internal/facets/named-reflect.js";
import { ProviderError } from "../src/internal/runtime/error.js";

test("reflection assignment replaces the descriptor without changing its copies", () => {
  const first = new ProviderError("first");
  const second = new ProviderError("second");
  const original = ValueOf(first);
  assert.equal(original.CanAddr(), false);
  const copied = ReflectValueOperations.$copy(original);
  const target = copied;
  const incoming = ValueOf(second);
  assert.notEqual(copied, original);
  assert.equal(copied.$unbox(), first);
  ReflectValueOperations.$assign(copied, incoming);
  assert.equal(copied, target);
  assert.equal(copied.$unbox(), second);
  assert.equal(original.$unbox(), first);
  ReflectValueOperations.$assign(incoming, ReflectValueOperations.$zero());
  assert.equal(copied.$unbox(), second);
  assert.equal(incoming.IsValid(), false);
  ReflectValueOperations.$assign(copied, copied);
  assert.equal(copied.$unbox(), second);
  ReflectValueOperations.$assign(copied, incoming);
  assert.equal(copied.IsValid(), false);
  assert.equal(copied.CanAddr(), false);
  assert.equal(original.$unbox(), first);
});
