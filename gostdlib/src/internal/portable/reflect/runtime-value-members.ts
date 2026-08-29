import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";

import type { Type } from "../../../reflect.js";

export interface RuntimeValueAdapter<T> {
  new (value: T): GoInterfaceValue & { readonly $go$value: T };
  $is(
    value: GoInterfaceValue | undefined,
  ): value is GoInterfaceValue & { readonly $go$value: T };
}

export type RuntimeValueAdapterResolver<T> = () => RuntimeValueAdapter<T>;

export interface RuntimeStructFieldOperations<T> {
  readonly type: () => Type;
  readonly settable: boolean;
  readonly get: (value: T) => GoInterfaceValue | undefined;
  readonly set: (
    value: T,
    field: GoInterfaceValue | undefined,
  ) => void;
  readonly address?: (value: T) => GoInterfaceValue;
}

export interface RuntimeStructFieldBuilder<T> {
  readonly value: <F>(
    type: () => Type,
    resolveAdapter: RuntimeValueAdapterResolver<F>,
    get: (value: T) => F,
    set: (value: T, field: F) => void,
    address?: (value: T) => GoInterfaceValue,
  ) => RuntimeStructFieldOperations<T>;
  readonly readonlyValue: <F>(
    type: () => Type,
    resolveAdapter: RuntimeValueAdapterResolver<F>,
    get: (value: T) => F,
    address?: (value: T) => GoInterfaceValue,
  ) => RuntimeStructFieldOperations<T>;
  readonly interfaceValue: <I extends GoInterfaceValue>(
    type: () => Type,
    admit: (field: GoInterfaceValue | undefined) => I | undefined,
    get: (value: T) => I | undefined,
    set: (value: T, field: I | undefined) => void,
    address?: (value: T) => GoInterfaceValue,
  ) => RuntimeStructFieldOperations<T>;
  readonly readonlyInterface: (
    type: () => Type,
    get: (value: T) => GoInterfaceValue | undefined,
    address?: (value: T) => GoInterfaceValue,
  ) => RuntimeStructFieldOperations<T>;
}

export type RuntimeStructFieldFactory<T> = (
  fields: RuntimeStructFieldBuilder<T>,
) => readonly RuntimeStructFieldOperations<T>[];

export interface RuntimePointerElementOperations<P> {
  readonly type: () => Type;
  readonly get: (pointer: P) => GoInterfaceValue | undefined;
  readonly set: (
    pointer: P,
    value: GoInterfaceValue | undefined,
  ) => void;
}

export interface RuntimePointerElementBuilder<P> {
  readonly value: <E>(
    type: () => Type,
    resolveAdapter: RuntimeValueAdapterResolver<E>,
    get: (pointer: P) => E,
    set: (pointer: P, value: E) => void,
  ) => RuntimePointerElementOperations<P>;
  readonly interfaceValue: <I extends GoInterfaceValue>(
    type: () => Type,
    admit: (value: GoInterfaceValue | undefined) => I | undefined,
    get: (pointer: P) => I | undefined,
    set: (pointer: P, value: I | undefined) => void,
  ) => RuntimePointerElementOperations<P>;
}

export interface RuntimePointerValueOperations<P> {
  readonly element: RuntimePointerElementOperations<P>;
  readonly newPointer?: () => P;
}

function unaddressableSetter(): void {
  return GoPanic.raiseRuntime(
    "reflect: Value.Set using unaddressable value",
  );
}

export function createRuntimeStructFieldBuilder<T>(): RuntimeStructFieldBuilder<T> {
  const value = <F>(
    type: () => Type,
    resolveAdapter: RuntimeValueAdapterResolver<F>,
    get: (owner: T) => F,
    set: (owner: T, field: F) => void,
    address?: (owner: T) => GoInterfaceValue,
  ): RuntimeStructFieldOperations<T> => {
    const adapter = resolveAdapter();
    return {
      type,
      settable: true,
      get: (owner: T): GoInterfaceValue => new adapter(get(owner)),
      set: (owner, field): void => {
        const admitted = adapter.$is(field)
          ? field.$go$value
          : GoPanic.raiseRuntime(
            "reflect: Value.Set received a foreign interface box",
          );
        set(owner, admitted);
      },
      ...(address === undefined ? {} : { address }),
    };
  };
  const readonlyValue = <F>(
    type: () => Type,
    resolveAdapter: RuntimeValueAdapterResolver<F>,
    get: (owner: T) => F,
    address?: (owner: T) => GoInterfaceValue,
  ): RuntimeStructFieldOperations<T> => {
    const adapter = resolveAdapter();
    return {
      type,
      settable: false,
      get: (owner: T): GoInterfaceValue => new adapter(get(owner)),
      set: unaddressableSetter,
      ...(address === undefined ? {} : { address }),
    };
  };
  const interfaceValue = <I extends GoInterfaceValue>(
    type: () => Type,
    admit: (field: GoInterfaceValue | undefined) => I | undefined,
    get: (owner: T) => I | undefined,
    set: (owner: T, field: I | undefined) => void,
    address?: (owner: T) => GoInterfaceValue,
  ): RuntimeStructFieldOperations<T> => ({
    type,
    settable: true,
    get,
    set: (owner, field): void => {
      set(owner, admit(field));
    },
    ...(address === undefined ? {} : { address }),
  });
  const readonlyInterface = (
    type: () => Type,
    get: (owner: T) => GoInterfaceValue | undefined,
    address?: (owner: T) => GoInterfaceValue,
  ): RuntimeStructFieldOperations<T> => ({
    type,
    settable: false,
    get,
    set: unaddressableSetter,
    ...(address === undefined ? {} : { address }),
  });
  return { value, readonlyValue, interfaceValue, readonlyInterface };
}

export function createRuntimePointerElementBuilder<P>(): RuntimePointerElementBuilder<P> {
  const value = <E>(
    type: () => Type,
    resolveAdapter: RuntimeValueAdapterResolver<E>,
    get: (pointer: P) => E,
    set: (pointer: P, value: E) => void,
  ): RuntimePointerElementOperations<P> => {
    const adapter = resolveAdapter();
    return {
      type,
      get: (pointer: P): GoInterfaceValue => new adapter(get(pointer)),
      set: (pointer, field): void => {
        const admitted = adapter.$is(field)
          ? field.$go$value
          : GoPanic.raiseRuntime(
            "reflect: Value.Set received a foreign interface box",
          );
        set(pointer, admitted);
      },
    };
  };
  const interfaceValue = <I extends GoInterfaceValue>(
    type: () => Type,
    admit: (value: GoInterfaceValue | undefined) => I | undefined,
    get: (pointer: P) => I | undefined,
    set: (pointer: P, value: I | undefined) => void,
  ): RuntimePointerElementOperations<P> => ({
    type,
    get,
    set: (pointer, field): void => {
      set(pointer, admit(field));
    },
  });
  return { value, interfaceValue };
}
