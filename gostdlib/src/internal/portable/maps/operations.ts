import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoMapValue } from "@gotots/runtime/map.js";
import type { bool } from "@gotots/gostdlib/internal/scalars.js";

import { Seq } from "../iter/sequence.js";

type Convert<Source, Target> = (value: Source) => Target;
type CopyValue<T> = (value: T) => T;
type Equality<L, R> = ((left: L, right: R) => bool) | undefined;
type MakeMap<K, V> = (zero: V) => GoMapValue<K, V>;
type Zero<T> = () => T;

export function Copy<M1, M2, K, V>(
  targetMap: Convert<M1, GoMapValue<K, V>>,
  sourceMap: Convert<M2, GoMapValue<K, V>>,
  copyKey: CopyValue<K>,
  copyValue: CopyValue<V>,
  target: M1,
  source: M2,
): void {
  const targetValue = targetMap(target);
  const sourceValue = sourceMap(source);
  for (const key of sourceValue.keys()) {
    targetValue.store(copyKey(key), copyValue(sourceValue.lookup(key)));
  }
}

export function Clone<M, K, V>(
  resultMap: Convert<GoMapValue<K, V>, M>,
  sourceMap: Convert<M, GoMapValue<K, V>>,
  copyKey: CopyValue<K>,
  copyValue: CopyValue<V>,
  makeMap: MakeMap<K, V>,
  zeroValue: Zero<V>,
  source: M,
): M {
  const sourceValue = sourceMap(source);
  if (sourceValue.isNil()) {
    return source;
  }
  const result = makeMap(zeroValue());
  for (const key of sourceValue.keys()) {
    result.store(copyKey(key), copyValue(sourceValue.lookup(key)));
  }
  return resultMap(result);
}

export function Equal<M1, M2, K, V>(
  leftMap: Convert<M1, GoMapValue<K, V>>,
  rightMap: Convert<M2, GoMapValue<K, V>>,
  equal: (left: V, right: V) => bool,
  left: M1,
  right: M2,
): bool {
  const leftValue = leftMap(left);
  const rightValue = rightMap(right);
  if (leftValue.length() !== rightValue.length()) {
    return false;
  }
  for (const key of leftValue.keys()) {
    const [candidate, present] = rightValue.lookupOk(key);
    if (!present || !equal(leftValue.lookup(key), candidate)) {
      return false;
    }
  }
  return true;
}

export function EqualFunc<M1, M2, K, L, R>(
  leftMap: Convert<M1, GoMapValue<K, L>>,
  rightMap: Convert<M2, GoMapValue<K, R>>,
  copyLeft: CopyValue<L>,
  copyRight: CopyValue<R>,
  left: M1,
  right: M2,
  equal: Equality<L, R>,
): bool {
  if (equal === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const leftValue = leftMap(left);
  const rightValue = rightMap(right);
  if (leftValue.length() !== rightValue.length()) {
    return false;
  }
  for (const key of leftValue.keys()) {
    const [candidate, present] = rightValue.lookupOk(key);
    if (
      !present
      || !equal(
        copyLeft(leftValue.lookup(key)),
        copyRight(candidate),
      )
    ) {
      return false;
    }
  }
  return true;
}

export function Keys<M, K, V>(
  sourceMap: Convert<M, GoMapValue<K, V>>,
  copyKey: CopyValue<K>,
  source: M,
): Seq<K> {
  const sourceValue = sourceMap(source);
  return new Seq<K>((yieldValue): void => {
    if (yieldValue === undefined) {
      GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    for (const key of sourceValue.keys()) {
      if (!yieldValue(copyKey(key))) {
        return;
      }
    }
  });
}

export function Values<M, K, V>(
  sourceMap: Convert<M, GoMapValue<K, V>>,
  copyValue: CopyValue<V>,
  source: M,
): Seq<V> {
  const sourceValue = sourceMap(source);
  return new Seq<V>((yieldValue): void => {
    if (yieldValue === undefined) {
      GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    for (const key of sourceValue.keys()) {
      if (!yieldValue(copyValue(sourceValue.lookup(key)))) {
        return;
      }
    }
  });
}
