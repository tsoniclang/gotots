import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { bool } from "@gotots/runtime/scalars.js";

import {
  ErrorsIsDirect,
  ErrorsIsCanonical,
  type ProviderErrorInterface,
  type ProviderErrorIs,
  type ProviderErrorIsDirect,
  type ProviderErrorUnwrap,
  type ProviderErrorUnwrapDirect,
  type ProviderErrorUnwrapMany,
  type ProviderErrorUnwrapManyDirect,
} from "./provider-error.js";
import type { CanonicalError } from "./provider-io-contract.js";

export type {
  ProviderErrorUnwrapDirect,
  ProviderErrorUnwrapManyDirect,
  ProviderErrorUnwrap,
  ProviderErrorUnwrapMany,
} from "./provider-error.js";
export type { CanonicalError } from "./provider-io-contract.js";

type ErrorGuard<Value extends GoInterfaceValue> = (
  value: GoInterfaceValue | undefined,
) => value is Value;

export function OsIsNotExistDirect(
  failure: ProviderErrorInterface | undefined,
  target: ProviderErrorInterface | undefined,
  isUnwrap: ErrorGuard<ProviderErrorUnwrapDirect>,
  isUnwrapMany: ErrorGuard<ProviderErrorUnwrapManyDirect>,
): bool {
  return ErrorsIsDirect(
    failure,
    target,
    isNeverCustomDirect,
    isUnwrap,
    isUnwrapMany,
  );
}

export function OsIsNotExistCanonical(
  failure: CanonicalError | undefined,
  target: CanonicalError | undefined,
  isUnwrap: ErrorGuard<ProviderErrorUnwrap>,
  isUnwrapMany: ErrorGuard<ProviderErrorUnwrapMany>,
): Promise<bool> {
  return ErrorsIsCanonical(
    failure,
    target,
    isNeverCustom,
    isUnwrap,
    isUnwrapMany,
  );
}

function isNeverCustom(
  _value: GoInterfaceValue | undefined,
): _value is ProviderErrorIs {
  return false;
}

function isNeverCustomDirect(
  _value: GoInterfaceValue | undefined,
): _value is ProviderErrorIsDirect {
  return false;
}
