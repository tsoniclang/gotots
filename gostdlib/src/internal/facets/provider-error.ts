import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, bool } from "@gotots/gostdlib/internal/scalars.js";

import {
  MessageWrappedErrors,
  WrappedProviderError,
} from "../portable/errors/tree.js";
import { ErrnoError } from "../portable/syscall/errno.js";
import { sliceValues } from "../runtime/slice.js";
import type { CanonicalError } from "./provider-io-contract.js";
import type { InterfaceGuard } from "./provider-support.js";

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

export function AsProviderErrorIsDirect(
  value: ProviderErrorInterface,
): ProviderErrorIsDirect | undefined {
  return value instanceof ErrnoError ? value : undefined;
}

export function AsProviderErrorUnwrapDirect(
  value: ProviderErrorInterface,
): ProviderErrorUnwrapDirect | undefined {
  return value instanceof WrappedProviderError ? value : undefined;
}

export function AsProviderErrorUnwrapManyDirect(
  value: ProviderErrorInterface,
): ProviderErrorUnwrapManyDirect | undefined {
  return value instanceof MessageWrappedErrors ? value : undefined;
}

export function AsProviderErrorIs(
  value: CanonicalError,
): ProviderErrorIs | undefined {
  return value instanceof ErrnoError ? value : undefined;
}

export function AsProviderErrorUnwrap(
  value: CanonicalError,
): ProviderErrorUnwrap | undefined {
  return value instanceof WrappedProviderError ? value : undefined;
}

export function AsProviderErrorUnwrapMany(
  value: CanonicalError,
): ProviderErrorUnwrapMany | undefined {
  return value instanceof MessageWrappedErrors ? value : undefined;
}

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
    const directCustom = AsProviderErrorIsDirect(current);
    if (directCustom !== undefined && directCustom.Is(target)) {
      return true;
    }
    if (isCustom(current) && current.Is(target)) {
      return true;
    }
    const directUnwrap = AsProviderErrorUnwrapDirect(current);
    if (directUnwrap !== undefined) {
      current = directUnwrap.Unwrap();
      continue;
    }
    const generatedUnwrap = isUnwrap(current) ? current : undefined;
    if (generatedUnwrap !== undefined) {
      current = generatedUnwrap.Unwrap();
      continue;
    }
    const directUnwrapMany = AsProviderErrorUnwrapManyDirect(current);
    if (directUnwrapMany !== undefined) {
      for (const cause of sliceValues(directUnwrapMany.Unwrap())) {
        if (ErrorsIsDirect(cause, target, isCustom, isUnwrap, isUnwrapMany)) {
          return true;
        }
      }
      return false;
    }
    const generatedUnwrapMany = isUnwrapMany(current) ? current : undefined;
    if (generatedUnwrapMany !== undefined) {
      for (const cause of sliceValues(generatedUnwrapMany.Unwrap())) {
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
  if (failure === undefined) {
    return undefined;
  }
  const direct = AsProviderErrorUnwrapDirect(failure);
  return direct !== undefined
    ? direct.Unwrap()
    : isUnwrap(failure) ? failure.Unwrap() : undefined;
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
    const directCustom = AsProviderErrorIs(current);
    if (directCustom !== undefined && await directCustom.Is(target)) {
      return true;
    }
    if (isCustom(current) && await current.Is(target)) {
      return true;
    }
    const directUnwrap = AsProviderErrorUnwrap(current);
    if (directUnwrap !== undefined) {
      current = await directUnwrap.Unwrap();
      continue;
    }
    const generatedUnwrap = isUnwrap(current) ? current : undefined;
    if (generatedUnwrap !== undefined) {
      current = await generatedUnwrap.Unwrap();
      continue;
    }
    const directUnwrapMany = AsProviderErrorUnwrapMany(current);
    if (directUnwrapMany !== undefined) {
      for (const cause of sliceValues(await directUnwrapMany.Unwrap())) {
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
    const generatedUnwrapMany = isUnwrapMany(current) ? current : undefined;
    if (generatedUnwrapMany !== undefined) {
      for (const cause of sliceValues(await generatedUnwrapMany.Unwrap())) {
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
  return isUnwrap(failure) ? failure.Unwrap() : undefined;
}
