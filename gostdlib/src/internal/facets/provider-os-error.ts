import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { bool } from "@gotots/runtime/scalars.js";

import {
  ErrorsIsCanonicalAsyncErrorSync,
  type ProviderErrorIsSyncAsyncError,
  type ProviderErrorUnwrapManySyncAsyncError,
  type ProviderErrorUnwrapSyncAsyncError,
} from "./provider-error.js";
import type { CanonicalErrorAsync } from "./provider-io-contract.js";

export type {
  ProviderErrorUnwrapManySyncAsyncError,
  ProviderErrorUnwrapSyncAsyncError,
} from "./provider-error.js";
export type { CanonicalErrorAsync } from "./provider-io-contract.js";

type ErrorGuard<Value extends GoInterfaceValue> = (
  value: GoInterfaceValue | undefined,
) => value is Value;

export function OsIsNotExistCanonicalAsyncErrorSync(
  failure: CanonicalErrorAsync | undefined,
  target: CanonicalErrorAsync | undefined,
  isUnwrap: ErrorGuard<ProviderErrorUnwrapSyncAsyncError>,
  isUnwrapMany: ErrorGuard<ProviderErrorUnwrapManySyncAsyncError>,
): bool {
  return ErrorsIsCanonicalAsyncErrorSync(
    failure,
    target,
    isNeverCustom,
    isUnwrap,
    isUnwrapMany,
  );
}

function isNeverCustom(
  _value: GoInterfaceValue | undefined,
): _value is ProviderErrorIsSyncAsyncError {
  return false;
}
