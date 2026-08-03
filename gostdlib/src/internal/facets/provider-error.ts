import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, bool } from "@gotots/runtime/scalars.js";

import {
  MessageWrappedErrors,
  WrappedProviderError,
} from "../portable/errors/tree.js";
import { sliceValues } from "../runtime/slice.js";
import type { CanonicalErrorAsync } from "./provider-io-contract.js";

export type { CanonicalErrorAsync } from "./provider-io-contract.js";

export interface ProviderErrorInterface extends GoInterfaceValue {
  Error(): string;
}

export interface ProviderErrorIsSync extends GoInterfaceValue {
  Is(target: GoError | undefined, recovery?: GoRecovery): bool;
}

export interface ProviderErrorIsAsync extends GoInterfaceValue {
  Is(target: GoError | undefined, recovery?: GoRecovery): Awaitable<bool>;
}

export interface ProviderErrorUnwrapSync extends GoInterfaceValue {
  Unwrap(recovery?: GoRecovery): GoError | undefined;
}

export interface ProviderErrorUnwrapAsync extends GoInterfaceValue {
  Unwrap(recovery?: GoRecovery): Awaitable<GoError | undefined>;
}

export interface ProviderErrorUnwrapManySync extends GoInterfaceValue {
  Unwrap(recovery?: GoRecovery): RuntimeSlice<GoError | undefined>;
}

export interface ProviderErrorUnwrapManyAsync extends GoInterfaceValue {
  Unwrap(recovery?: GoRecovery): Awaitable<RuntimeSlice<GoError | undefined>>;
}

export interface ProviderErrorIsSyncAsyncError extends GoInterfaceValue {
  Is(target: CanonicalErrorAsync | undefined, recovery?: GoRecovery): bool;
}

export interface ProviderErrorIsAsyncAsyncError extends GoInterfaceValue {
  Is(
    target: CanonicalErrorAsync | undefined,
    recovery?: GoRecovery,
  ): Awaitable<bool>;
}

export interface ProviderErrorUnwrapSyncAsyncError extends GoInterfaceValue {
  Unwrap(recovery?: GoRecovery): CanonicalErrorAsync | undefined;
}

export interface ProviderErrorUnwrapAsyncAsyncError extends GoInterfaceValue {
  Unwrap(recovery?: GoRecovery): Awaitable<CanonicalErrorAsync | undefined>;
}

export interface ProviderErrorUnwrapManySyncAsyncError extends GoInterfaceValue {
  Unwrap(
    recovery?: GoRecovery,
  ): RuntimeSlice<CanonicalErrorAsync | undefined>;
}

export interface ProviderErrorUnwrapManyAsyncAsyncError extends GoInterfaceValue {
  Unwrap(
    recovery?: GoRecovery,
  ): Awaitable<RuntimeSlice<CanonicalErrorAsync | undefined>>;
}

type InterfaceGuard<Value extends GoInterfaceValue> = (
  value: GoInterfaceValue | undefined,
) => value is Value;

export function ErrorsIsCanonicalSync(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsSync>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapSync>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManySync>,
): bool {
  return errorsIsSync(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

function errorsIsSync(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsSync>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapSync>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManySync>,
): bool {
  if (failure === undefined || target === undefined) {
    return failure === target;
  }
  let current: GoError | undefined = failure;
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
        if (errorsIsSync(cause, target, isCustom, isUnwrap, isUnwrapMany)) {
          return true;
        }
      }
      return false;
    }
    if (current instanceof MessageWrappedErrors) {
      for (const cause of current.UnwrapAll()) {
        if (errorsIsSync(cause, target, isCustom, isUnwrap, isUnwrapMany)) {
          return true;
        }
      }
    }
    return false;
  }
  return false;
}

type ProviderErrorIs = ProviderErrorIsSync | ProviderErrorIsAsync;
type ProviderErrorUnwrap = ProviderErrorUnwrapSync | ProviderErrorUnwrapAsync;
type ProviderErrorUnwrapMany =
  | ProviderErrorUnwrapManySync
  | ProviderErrorUnwrapManyAsync;

async function errorsIsAsync(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIs>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrap>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapMany>,
): Promise<bool> {
  if (failure === undefined || target === undefined) {
    return failure === target;
  }
  let current: GoError | undefined = failure;
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
        if (await errorsIsAsync(cause, target, isCustom, isUnwrap, isUnwrapMany)) {
          return true;
        }
      }
      return false;
    }
    if (current instanceof MessageWrappedErrors) {
      for (const cause of current.UnwrapAll()) {
        if (await errorsIsAsync(cause, target, isCustom, isUnwrap, isUnwrapMany)) {
          return true;
        }
      }
    }
    return false;
  }
  return false;
}

export async function ErrorsIsCanonicalAsyncIs(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsAsync>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapSync>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManySync>,
): Promise<bool> {
  return errorsIsAsync(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncUnwrap(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsSync>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapAsync>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManySync>,
): Promise<bool> {
  return errorsIsAsync(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncUnwrapMany(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsSync>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapSync>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManyAsync>,
): Promise<bool> {
  return errorsIsAsync(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncIsUnwrap(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsAsync>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapAsync>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManySync>,
): Promise<bool> {
  return errorsIsAsync(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncIsUnwrapMany(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsAsync>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapSync>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManyAsync>,
): Promise<bool> {
  return errorsIsAsync(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncUnwrapBoth(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsSync>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapAsync>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManyAsync>,
): Promise<bool> {
  return errorsIsAsync(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncAll(
  failure: GoError | undefined,
  target: GoError | undefined,
  isCustom: InterfaceGuard<ProviderErrorIsAsync>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapAsync>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManyAsync>,
): Promise<bool> {
  return errorsIsAsync(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

type AsyncErrorGuard<Value extends GoInterfaceValue> = (
  value: GoInterfaceValue | undefined,
) => value is Value;

export function ErrorsIsCanonicalAsyncErrorSync(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderErrorIsSyncAsyncError>,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapSyncAsyncError>,
  isUnwrapMany: AsyncErrorGuard<ProviderErrorUnwrapManySyncAsyncError>,
): bool {
  return errorsIsAsyncErrorSync(
    failure,
    target,
    isCustom,
    isUnwrap,
    isUnwrapMany,
  );
}

function errorsIsAsyncErrorSync(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderErrorIsSyncAsyncError>,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapSyncAsyncError>,
  isUnwrapMany: AsyncErrorGuard<ProviderErrorUnwrapManySyncAsyncError>,
): bool {
  if (failure === undefined || target === undefined) {
    return failure === target;
  }
  let current: CanonicalErrorAsync | undefined = failure;
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
    if (isUnwrapMany(current)) {
      for (const cause of sliceValues(current.Unwrap())) {
        if (errorsIsAsyncErrorSync(
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
    return false;
  }
  return false;
}

type ProviderAsyncErrorIs =
  | ProviderErrorIsSyncAsyncError
  | ProviderErrorIsAsyncAsyncError;
type ProviderAsyncErrorUnwrap =
  | ProviderErrorUnwrapSyncAsyncError
  | ProviderErrorUnwrapAsyncAsyncError;
type ProviderAsyncErrorUnwrapMany =
  | ProviderErrorUnwrapManySyncAsyncError
  | ProviderErrorUnwrapManyAsyncAsyncError;

async function errorsIsAsyncError(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderAsyncErrorIs>,
  isUnwrap: AsyncErrorGuard<ProviderAsyncErrorUnwrap>,
  isUnwrapMany: AsyncErrorGuard<ProviderAsyncErrorUnwrapMany>,
): Promise<bool> {
  if (failure === undefined || target === undefined) {
    return failure === target;
  }
  let current: CanonicalErrorAsync | undefined = failure;
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
    if (isUnwrapMany(current)) {
      for (const cause of sliceValues(await current.Unwrap())) {
        if (await errorsIsAsyncError(
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
    return false;
  }
  return false;
}

export async function ErrorsIsCanonicalAsyncErrorAsyncIs(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderErrorIsAsyncAsyncError>,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapSyncAsyncError>,
  isUnwrapMany: AsyncErrorGuard<ProviderErrorUnwrapManySyncAsyncError>,
): Promise<bool> {
  return errorsIsAsyncError(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncErrorAsyncUnwrap(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderErrorIsSyncAsyncError>,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapAsyncAsyncError>,
  isUnwrapMany: AsyncErrorGuard<ProviderErrorUnwrapManySyncAsyncError>,
): Promise<bool> {
  return errorsIsAsyncError(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncErrorAsyncUnwrapMany(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderErrorIsSyncAsyncError>,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapSyncAsyncError>,
  isUnwrapMany: AsyncErrorGuard<ProviderErrorUnwrapManyAsyncAsyncError>,
): Promise<bool> {
  return errorsIsAsyncError(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncErrorAsyncIsUnwrap(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderErrorIsAsyncAsyncError>,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapAsyncAsyncError>,
  isUnwrapMany: AsyncErrorGuard<ProviderErrorUnwrapManySyncAsyncError>,
): Promise<bool> {
  return errorsIsAsyncError(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncErrorAsyncIsUnwrapMany(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderErrorIsAsyncAsyncError>,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapSyncAsyncError>,
  isUnwrapMany: AsyncErrorGuard<ProviderErrorUnwrapManyAsyncAsyncError>,
): Promise<bool> {
  return errorsIsAsyncError(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncErrorAsyncUnwrapBoth(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderErrorIsSyncAsyncError>,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapAsyncAsyncError>,
  isUnwrapMany: AsyncErrorGuard<ProviderErrorUnwrapManyAsyncAsyncError>,
): Promise<bool> {
  return errorsIsAsyncError(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export async function ErrorsIsCanonicalAsyncErrorAsyncAll(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isCustom: AsyncErrorGuard<ProviderErrorIsAsyncAsyncError>,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapAsyncAsyncError>,
  isUnwrapMany: AsyncErrorGuard<ProviderErrorUnwrapManyAsyncAsyncError>,
): Promise<bool> {
  return errorsIsAsyncError(failure, target, isCustom, isUnwrap, isUnwrapMany);
}

export function ErrorsUnwrapCanonicalSync(
  failure: GoError | undefined,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapSync>,
): GoError | undefined {
  return failure !== undefined && isUnwrap(failure)
    ? failure.Unwrap()
    : undefined;
}

export async function ErrorsUnwrapCanonicalAsync(
  failure: GoError | undefined,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapAsync>,
): Promise<GoError | undefined> {
  return failure !== undefined && isUnwrap(failure)
    ? await failure.Unwrap()
    : undefined;
}

export function ErrorsUnwrapCanonicalAsyncErrorSync(
  failure: CanonicalErrorAsync | undefined,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapSyncAsyncError>,
): CanonicalErrorAsync | undefined {
  return failure !== undefined && isUnwrap(failure)
    ? failure.Unwrap()
    : undefined;
}

export async function ErrorsUnwrapCanonicalAsyncErrorAsync(
  failure: CanonicalErrorAsync | undefined,
  isUnwrap: AsyncErrorGuard<ProviderErrorUnwrapAsyncAsyncError>,
): Promise<CanonicalErrorAsync | undefined> {
  return failure !== undefined && isUnwrap(failure)
    ? await failure.Unwrap()
    : undefined;
}
