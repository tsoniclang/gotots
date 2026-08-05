import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  float64,
  gostring,
  int64,
  uint8,
  uint64,
} from "@gotots/gostdlib/internal/scalars.js";

import type { Type } from "../../../reflect.js";

// RuntimeValueLocation is one addressable typed view over original Go
// storage: reads box the current value and writes reach the represented
// storage through the generated accessors. Settability is generated
// evidence (exported-field and addressability facts), never inferred.
export interface RuntimeValueLocation {
  readonly type: () => Type;
  readonly settable: bool;
  readonly get: () => GoInterfaceValue;
  readonly set: (box: GoInterfaceValue) => void;
}

// RuntimeValueOperations are the generated per-type typed value callbacks
// of one reflection-visible Go type. Every callback receives the canonical
// interface box, performs its exact generated adapter guard, and projects
// or mutates through the represented storage. A callback is present
// exactly when the type's kind admits the operation; absence is kind
// evidence, never a missing-metadata fallback.
export interface RuntimeValueOperations {
  readonly int?: (box: GoInterfaceValue) => int64;
  readonly uint?: (box: GoInterfaceValue) => uint64;
  readonly float?: (box: GoInterfaceValue) => float64;
  readonly bool?: (box: GoInterfaceValue) => bool;
  readonly string?: (box: GoInterfaceValue) => gostring;
  readonly isNil?: (box: GoInterfaceValue) => bool;
  readonly isZero?: (box: GoInterfaceValue) => bool;
  readonly boxString?: (value: gostring) => GoInterfaceValue;
  readonly elem?: (
    box: GoInterfaceValue,
  ) => RuntimeValueLocation | undefined;
  readonly numField?: int64;
  readonly field?: (
    box: GoInterfaceValue,
    index: int64,
  ) => RuntimeValueLocation;
  readonly len?: (box: GoInterfaceValue) => int64;
  readonly cap?: (box: GoInterfaceValue) => int64;
  readonly index?: (
    box: GoInterfaceValue,
    index: int64,
  ) => RuntimeValueLocation;
  readonly append?: (
    box: GoInterfaceValue,
    values: readonly GoInterfaceValue[],
  ) => GoInterfaceValue;
  readonly makeSlice?: (
    length: int64,
    capacity: int64,
  ) => GoInterfaceValue;
  readonly resliced?: (
    box: GoInterfaceValue,
    length: int64,
  ) => GoInterfaceValue;
  readonly grown?: (
    box: GoInterfaceValue,
    count: int64,
  ) => GoInterfaceValue;
  readonly bytes?: (box: GoInterfaceValue) => RuntimeSlice<uint8>;
  readonly boxBytes?: (value: RuntimeSlice<uint8>) => GoInterfaceValue;
  readonly mapIndex?: (
    box: GoInterfaceValue,
    key: GoInterfaceValue,
  ) => GoInterfaceValue | undefined;
  readonly mapStore?: (
    box: GoInterfaceValue,
    key: GoInterfaceValue,
    value: GoInterfaceValue | undefined,
  ) => void;
  readonly mapKeys?: (
    box: GoInterfaceValue,
  ) => readonly GoInterfaceValue[];
  readonly makeMap?: () => GoInterfaceValue;
}

const operationsByType = new Map<Type, RuntimeValueOperations>();

// registerRuntimeValueOperations binds one generated value-operation record
// to its canonical runtime type descriptor.
export function registerRuntimeValueOperations(
  type: Type,
  operations: RuntimeValueOperations,
): void {
  operationsByType.set(type, operations);
}

// runtimeValueOperations resolves the generated value-operation record of
// one canonical descriptor.
export function runtimeValueOperations(
  type: Type | undefined,
): RuntimeValueOperations | undefined {
  return type === undefined ? undefined : operationsByType.get(type);
}

let typeResolver:
  | ((value: GoInterfaceValue) => Type | undefined)
  | undefined;

// bindRuntimeTypeResolver is installed once by the runtime-type module so
// the public reflect module can resolve canonical descriptors without a
// runtime module cycle.
export function bindRuntimeTypeResolver(
  resolver: (value: GoInterfaceValue) => Type | undefined,
): void {
  typeResolver = resolver;
}

// resolveRuntimeType resolves the canonical descriptor of one boxed value.
export function resolveRuntimeType(
  value: GoInterfaceValue,
): Type | undefined {
  return typeResolver === undefined ? undefined : typeResolver(value);
}
