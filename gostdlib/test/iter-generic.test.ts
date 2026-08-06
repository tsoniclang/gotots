import assert from "node:assert/strict";
import test from "node:test";

import { Seq, Seq2 } from "../src/iter.js";

test("iter.Seq exposes the selected push iterator contract", (): void => {
  const sequence = new Seq<number>((yieldValue): void => {
    if (yieldValue === undefined) {
      return;
    }
    for (const value of [2, 4, 6]) {
      if (!yieldValue(value)) {
        return;
      }
    }
  });
  const values: number[] = [];
  sequence.value?.((value): boolean => {
    values.push(value);
    return value < 4;
  });
  assert.deepEqual(values, [2, 4]);
});

test("iter.Seq2 preserves paired values and early stop", (): void => {
  const sequence = new Seq2<string, number>((yieldValue): void => {
    if (yieldValue !== undefined) {
      if (!yieldValue("a", 1)) {
        return;
      }
      yieldValue("b", 2);
    }
  });
  const values: [string, number][] = [];
  sequence.value?.((key, value): boolean => {
    values.push([key, value]);
    return false;
  });
  assert.deepEqual(values, [["a", 1]]);
});
