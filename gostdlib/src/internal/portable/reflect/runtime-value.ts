import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
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
import type { ProviderRawPointer } from "../../runtime/raw-pointer.js";
import {
  createRuntimePointerElementBuilder,
  createRuntimeStructFieldBuilder,
  type RuntimePointerElementBuilder,
  type RuntimePointerValueOperations,
  type RuntimeStructFieldFactory,
  type RuntimeStructFieldOperations,
  type RuntimeStructStorageResolver,
  type RuntimeValueAdapterResolver,
} from "./runtime-value-members.js";
export type {
  RuntimePointerElementBuilder,
  RuntimePointerValueOperations,
  RuntimeStructFieldFactory,
  RuntimeStructFieldOperations,
  RuntimeStructStorageResolver,
  RuntimeValueAdapter,
  RuntimeValueAdapterResolver,
} from "./runtime-value-members.js";

// RuntimeValueLocation is one addressable typed view over original Go
// storage: reads box the current value and writes reach the represented
// storage through the generated accessors. Settability is generated
// evidence (exported-field and addressability facts), never inferred.
export interface RuntimeValueLocation {
  readonly type: () => Type;
  readonly settable: bool;
  readonly get: () => GoInterfaceValue | undefined;
  readonly set: (box: GoInterfaceValue | undefined) => void;
  readonly address?: () => GoInterfaceValue;
}


// RuntimeValueOperations are the settled value operations of one
// reflection-visible Go type. Generated callbacks carry exact typed storage
// facts; common registrations own invariant guards and location mechanics.
// An operation is present exactly when the type's kind admits it; absence is
// kind evidence, never a missing-metadata fallback.
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
    values: readonly (GoInterfaceValue | undefined)[],
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
    key: GoInterfaceValue | undefined,
  ) => readonly [GoInterfaceValue | undefined, bool];
  readonly mapStore?: (
    box: GoInterfaceValue,
    key: GoInterfaceValue | undefined,
    value: GoInterfaceValue | undefined,
    deleteEntry: bool,
  ) => void;
  readonly mapKeys?: (
    box: GoInterfaceValue,
  ) => readonly (GoInterfaceValue | undefined)[];
  readonly makeMap?: () => GoInterfaceValue;
  readonly zero?: () => GoInterfaceValue | undefined;
  readonly boxInt?: (value: int64) => GoInterfaceValue;
  readonly boxUint?: (value: uint64) => GoInterfaceValue;
  readonly boxFloat?: (value: float64) => GoInterfaceValue;
  readonly boxBool?: (value: bool) => GoInterfaceValue;
  readonly newPointer?: () => GoInterfaceValue;
  readonly cloned?: (box: GoInterfaceValue) => GoInterfaceValue;
  readonly unsafePointer?: (
    box: GoInterfaceValue,
  ) => ProviderRawPointer | undefined;
}

const pointerDescriptors: Array<[Type, () => Type]> = [];

let pointerTypeFactory:
  | ((element: Type) => Type | undefined)
  | undefined;

// recordPointerDescriptor remembers one pointer-kind descriptor with its
// lazy element thunk; the thunk is never called during registration so
// forward descriptor references stay valid.
export function recordPointerDescriptor(
  type: Type,
  element: () => Type,
): void {
  pointerDescriptors.push([type, element]);
}

// pointerDescriptorFor resolves the canonical pointer descriptor whose
// element is the given descriptor, evaluating element thunks only at
// lookup time.
export function pointerDescriptorFor(element: Type): Type | undefined {
  for (const entry of pointerDescriptors) {
    if (entry[1]() === element) {
      return entry[0];
    }
  }
  const created = pointerTypeFactory?.(element);
  if (created !== undefined) {
    pointerDescriptors.push([created, () => element]);
  }
  return created;
}

// bindRuntimePointerTypeFactory installs the canonical runtime-type owner for
// pointer descriptors that are composed from a dynamically flowing Type.
export function bindRuntimePointerTypeFactory(
  factory: (element: Type) => Type | undefined,
): void {
  pointerTypeFactory = factory;
}

interface RuntimeValueOperationRegistration {
  readonly factory: () => RuntimeValueOperations;
  value?: RuntimeValueOperations;
}

const operationsByType = new Map<Type, RuntimeValueOperationRegistration>();

// registerRuntimeValueOperations binds one deferred generated value-operation
// record to its canonical runtime type descriptor.
export function registerRuntimeValueOperations(
  type: Type,
  factory: () => RuntimeValueOperations,
): void {
  operationsByType.set(type, { factory });
}

export function registerRuntimeStructValueOperations<T, S>(
  type: Type,
  resolveAdapter: RuntimeValueAdapterResolver<T>,
  resolveStorage: RuntimeStructStorageResolver<T, S>,
  createFields: RuntimeStructFieldFactory<T, S>,
  clone?: (value: T) => T,
): void {
  registerRuntimeValueOperations(type, () => {
    const adapter = resolveAdapter();
    const fields = createFields(
      createRuntimeStructFieldBuilder<T, S>(resolveStorage),
    );
    const fieldCount = BigInt(fields.length);
    const field = (
      box: GoInterfaceValue,
      index: int64,
    ): RuntimeValueLocation => {
      if (!adapter.$is(box)) {
        return GoPanic.raiseRuntime(
          "reflect: Value.Field received a foreign interface box",
        );
      }
      if (index < 0n || index >= fieldCount) {
        return GoPanic.raiseRuntime("reflect: Field index out of range");
      }
      const selected = fields[Number(index)];
      if (selected === undefined) {
        return GoPanic.raiseRuntime("reflect: Field index out of range");
      }
      const value = box.$go$value;
      const selectedAddress = selected.address;
      const address = selectedAddress === undefined
        ? {}
        : { address: (): GoInterfaceValue => selectedAddress(value) };
      return {
        type: selected.type,
        settable: selected.settable,
        get: (): GoInterfaceValue | undefined => selected.get(value),
        set: (fieldValue: GoInterfaceValue | undefined): void => {
          selected.set(value, fieldValue);
        },
        ...address,
      };
    };
    if (clone === undefined) {
      return { numField: fieldCount, field };
    }
    return {
      numField: fieldCount,
      field,
      cloned: (box: GoInterfaceValue): GoInterfaceValue => {
        if (!adapter.$is(box)) {
          return GoPanic.raiseRuntime(
            "reflect: Value.Interface received a foreign interface box",
          );
        }
        return new adapter(clone(box.$go$value));
      },
    };
  });
}

export function registerRuntimeOpaqueStructValueOperations<T>(
  type: Type,
  resolveAdapter: RuntimeValueAdapterResolver<T>,
  unavailableFields: readonly gostring[],
): void {
  registerRuntimeValueOperations(type, () => {
    const adapter = resolveAdapter();
    const fieldCount = BigInt(unavailableFields.length);
    return {
      numField: fieldCount,
      field: (box: GoInterfaceValue, index: int64): RuntimeValueLocation => {
        if (!adapter.$is(box)) {
          return GoPanic.raiseRuntime(
            "reflect: Value.Field received a foreign interface box",
          );
        }
        if (index < 0n || index >= fieldCount) {
          return GoPanic.raiseRuntime("reflect: Field index out of range");
        }
        const message = unavailableFields[Number(index)];
        return GoPanic.raiseRuntime(
          message ?? "reflect: Field index out of range",
        );
      },
    };
  });
}

export function registerRuntimePointerValueOperations<P>(
  type: Type,
  resolveAdapter: RuntimeValueAdapterResolver<P | undefined>,
  createDescriptor: (
    elements: RuntimePointerElementBuilder<P>,
  ) => RuntimePointerValueOperations<P>,
): void {
  registerRuntimeValueOperations(type, () => {
    const adapter = resolveAdapter();
    const descriptor = createDescriptor(createRuntimePointerElementBuilder<P>());
    const element = descriptor.element;
    const newPointer = descriptor.newPointer;
    const newPointerOperation = newPointer === undefined
      ? {}
      : { newPointer: (): GoInterfaceValue => new adapter(newPointer()) };
    return {
      isNil: (box: GoInterfaceValue): bool => {
        if (!adapter.$is(box)) {
          return GoPanic.raiseRuntime(
            "reflect: Value.IsNil received a foreign interface box",
          );
        }
        return box.$go$value === undefined;
      },
      elem: (
        box: GoInterfaceValue,
      ): RuntimeValueLocation | undefined => {
        if (!adapter.$is(box)) {
          return GoPanic.raiseRuntime(
            "reflect: Value.Elem received a foreign interface box",
          );
        }
        const pointer = box.$go$value;
        if (pointer === undefined) {
          return undefined;
        }
        return {
          type: element.type,
          settable: true,
          get: (): GoInterfaceValue | undefined => element.get(pointer),
          set: (value: GoInterfaceValue | undefined): void => {
            element.set(pointer, value);
          },
          address: (): GoInterfaceValue => box,
        };
      },
      zero: (): GoInterfaceValue => new adapter(undefined),
      ...newPointerOperation,
    };
  });
}

// runtimeValueOperations resolves the generated value-operation record of
// one canonical descriptor.
export function runtimeValueOperations(
  type: Type | undefined,
): RuntimeValueOperations | undefined {
  if (type === undefined) {
    return undefined;
  }
  const registration = operationsByType.get(type);
  if (registration === undefined) {
    return undefined;
  }
  registration.value ??= registration.factory();
  return registration.value;
}

let typeResolver:
  | ((value: GoInterfaceValue) => Type | undefined)
  | undefined;
let typeRecorder:
  | ((value: GoInterfaceValue, type: Type) => void)
  | undefined;

// bindRuntimeTypeResolver is installed once by the runtime-type module so
// the public reflect module can resolve canonical descriptors without a
// runtime module cycle.
export function bindRuntimeTypeResolver(
  resolver: (value: GoInterfaceValue) => Type | undefined,
): void {
  typeResolver = resolver;
}

export function bindRuntimeTypeRecorder(
  recorder: (value: GoInterfaceValue, type: Type) => void,
): void {
  typeRecorder = recorder;
}

// resolveRuntimeType resolves the canonical descriptor of one boxed value.
export function resolveRuntimeType(
  value: GoInterfaceValue,
): Type | undefined {
  return typeResolver === undefined ? undefined : typeResolver(value);
}

export function recordRuntimeType(
  value: GoInterfaceValue,
  type: Type,
): void {
  if (typeRecorder === undefined) {
    return GoPanic.raiseRuntime("reflect: runtime type recorder is absent");
  }
  typeRecorder(value, type);
}
