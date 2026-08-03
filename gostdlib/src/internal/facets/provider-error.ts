import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, bool } from "@gotots/runtime/scalars.js";

import {
  MessageWrappedErrors,
  WrappedProviderError,
} from "../portable/errors/tree.js";
import { sliceValues } from "../runtime/slice.js";
import type { CanonicalError } from "./provider-io-contract.js";

export type { CanonicalError } from "./provider-io-contract.js";

export interface ProviderErrorInterface extends GoInterfaceValue {
  Error(): string;
}

export interface ProviderErrorIsDirect extends GoInterfaceValue {
  Is(
    target: ProviderErrorInterface | undefined,
    recovery?: GoRecovery,
  ): bool;
}

export interface ProviderErrorUnwrapDirect extends GoInterfaceValue {
  Unwrap(recovery?: GoRecovery): ProviderErrorInterface | undefined;
}

export interface ProviderErrorUnwrapManyDirect extends GoInterfaceValue {
  Unwrap(
    recovery?: GoRecovery,
  ): RuntimeSlice<ProviderErrorInterface | undefined>;
}

export interface ProviderErrorIs extends GoInterfaceValue {
  Is(
    target: CanonicalError | undefined,
    recovery?: GoRecovery,
  ): Awaitable<bool>;
}

export interface ProviderErrorUnwrap extends GoInterfaceValue {
  Unwrap(recovery?: GoRecovery): Awaitable<CanonicalError | undefined>;
}

export interface ProviderErrorUnwrapMany extends GoInterfaceValue {
  Unwrap(
    recovery?: GoRecovery,
  ): Awaitable<RuntimeSlice<CanonicalError | undefined>>;
}

type InterfaceGuard<Value extends GoInterfaceValue> = (
  value: GoInterfaceValue | undefined,
) => value is Value;

export function ErrorsIsDirect(
  failure: ProviderErrorInterface | undefined,
  target: ProviderErrorInterface | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsDirect>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapDirect>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManyDirect>,
): bool {
  if (failure === undefined || target === undefined) {
    return failure === target;
  }
  let current: ProviderErrorInterface | undefined = failure;
  while (current !== undefined) {
    if (target.$go$type.comparable && current.$go$equal(target)) {
      return true;
    }
    if (isCustom(current) && current.Is(target)) {
      return true;
    }
    if (isUnwrap(current)) {
      current = current.Unwrap();
      continue;
    }
    if (current instanceof WrappedProviderError) {
      current = current.Unwrap();
      continue;
    }
    if (isUnwrapMany(current)) {
      for (const cause of sliceValues(current.Unwrap())) {
        if (ErrorsIsDirect(cause, target, isCustom, isUnwrap, isUnwrapMany)) {
          return true;
        }
      }
      return false;
    }
    if (current instanceof MessageWrappedErrors) {
      for (const cause of current.UnwrapAll()) {
        if (ErrorsIsDirect(cause, target, isCustom, isUnwrap, isUnwrapMany)) {
          return true;
        }
      }
    }
    return false;
  }
  return false;
}

export function ErrorsUnwrapDirect(
  failure: ProviderErrorInterface | undefined,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapDirect>,
): ProviderErrorInterface | undefined {
  if (failure instanceof WrappedProviderError) {
    return failure.Unwrap();
  }
  return isUnwrap(failure) ? failure.Unwrap() : undefined;
}

export async function ErrorsIsCanonical(
  failure: CanonicalError | undefined,
  target: CanonicalError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIs>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrap>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapMany>,
): Promise<bool> {
  if (failure === undefined || target === undefined) {
    return failure === target;
  }
  let current: CanonicalError | undefined = failure;
  while (current !== undefined) {
    if (target.$go$type.comparable && current.$go$equal(target)) {
      return true;
    }
    if (isCustom(current) && await current.Is(target)) {
      return true;
    }
    if (isUnwrap(current)) {
      current = await current.Unwrap();
      continue;
    }
    if (current instanceof WrappedProviderError) {
      current = current.Unwrap();
      continue;
    }
    if (isUnwrapMany(current)) {
      for (const cause of sliceValues(await current.Unwrap())) {
        if (await ErrorsIsCanonical(
          cause,
          target,
          isCustom,
          isUnwrap,
          isUnwrapMany,
        )) {
          return true;
        }
      }
      return false;
    }
    if (current instanceof MessageWrappedErrors) {
      for (const cause of current.UnwrapAll()) {
        if (await ErrorsIsCanonical(
          cause,
          target,
          isCustom,
          isUnwrap,
          isUnwrapMany,
        )) {
          return true;
        }
      }
    }
    return false;
  }
  return false;
}

export async function ErrorsUnwrapCanonical(
  failure: CanonicalError | undefined,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrap>,
): Promise<CanonicalError | undefined> {
  if (failure instanceof WrappedProviderError) {
    return failure.Unwrap();
  }
  return isUnwrap(failure) ? failure.Unwrap() : undefined;
}
