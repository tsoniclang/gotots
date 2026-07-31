import assert from "node:assert/strict";
import test from "node:test";

import { GoMap, type GoMapValue } from "@gotots/runtime/map.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  MapsKeysCooperative,
} from "../src/internal/facets/generic-maps.js";
import {
  SlicesCollectCooperative,
} from "../src/internal/facets/generic-slices.js";
import {
  RuntimeMetricsDescriptionOperations,
  RuntimeMetricsSampleOperations,
} from "../src/internal/facets/named-runtime-metrics.js";
import {
  SyncPoolOperations,
} from "../src/internal/facets/named-sync.js";
import {
  UnicodeRange16Operations,
} from "../src/internal/facets/named-unicode.js";
import {
  BufioReaderRead,
} from "../src/internal/facets/recovery-io.js";
import { NewReader } from "../src/bufio.js";
import { Description, Sample, Value } from "../src/runtime/metrics.js";
import { Seq } from "../src/iter.js";

test("named-struct facets expose only selected static operations", (): void => {
  const range = UnicodeRange16Operations.$make(1, 4, 1);
  assert.deepEqual([range.Lo, range.Hi, range.Stride], [1, 4, 1]);

  const pool = SyncPoolOperations.$zero();
  assert.equal(SyncPoolOperations.$fromStorage(
    SyncPoolOperations.$storageOf(pool),
  ), pool);

  const sample = RuntimeMetricsSampleOperations.$copy(
    new Sample("/metric", new Value()),
  );
  assert.equal(sample.Name, "/metric");
  const description = RuntimeMetricsDescriptionOperations.$copy(
    new Description("/metric", "detail"),
  );
  assert.equal(description.Description, "detail");
});

test("recovery facets preserve the direct provider ABI", (): void => {
  const reader = NewReader({
    Read(destination): [number, undefined] {
      destination.set(0, 65);
      return [1, undefined];
    },
    $go$type: Object.freeze({}),
    $go$methods: new Set<object>(),
    $go$implements(): boolean { return true; },
    $go$equal(other): boolean { return this === other; },
    $go$hash(): number { return 0; },
  });
  const destination = RuntimeSlice.make<number>(1, 1, 0);
  assert.deepEqual(BufioReaderRead(reader, destination), [1, undefined]);
  assert.equal(destination.get(0), 65);
});

test("generic facets adapt cooperative provider implementations", async (): Promise<void> => {
  const source = GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]);
  const keys: string[] = [];
  await MapsKeysCooperative<GoMapValue<string, number>, string, number>(
    source,
  ).value?.(async (key): Promise<boolean> => {
    keys.push(key);
    return true;
  });
  assert.deepEqual(keys.sort(), ["a", "b"]);

  const sequence = new Seq<number, (
    yieldValue: ((value: number) => Promise<boolean>) | undefined,
  ) => Promise<void>>(async (yieldValue): Promise<void> => {
    await yieldValue?.(2);
    await yieldValue?.(3);
  });
  const values = await SlicesCollectCooperative(sequence);
  assert.deepEqual([values.get(0), values.get(1)], [2, 3]);
});
