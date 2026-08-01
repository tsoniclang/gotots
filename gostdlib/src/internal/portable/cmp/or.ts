import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool } from "@gotots/runtime/scalars.js";

type CopyValue<T> = (value: T) => T;
type EqualValue<T> = (left: T, right: T) => bool;
type FromContainerStorage<T, Storage> = (value: Storage) => T;
type Zero<T> = () => T;

export function Or<T, Storage>(
  copy: CopyValue<T>,
  equal: EqualValue<T>,
  fromStorage: FromContainerStorage<T, Storage>,
  zero: Zero<T>,
  values: RuntimeSlice<Storage>,
): T {
  const zeroValue = zero();
  for (let index = 0; index < values.length; index += 1) {
    const value = copy(fromStorage(values.get(index)));
    if (!equal(value, zeroValue)) {
      return value;
    }
  }
  return zeroValue;
}
