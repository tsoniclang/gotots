import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { Awaitable, bool } from "@gotots/gostdlib/internal/scalars.js";

import {
  ErrorsIsDirect,
  ErrorsIsCanonical,
  type ProviderErrorInterface,
  type ProviderErrorUnwrap,
  type ProviderErrorUnwrapDirect,
  type ProviderErrorUnwrapMany,
  type ProviderErrorUnwrapManyDirect,
} from "./provider-error.js";
import type { CanonicalError } from "./provider-io-contract.js";
import type { InterfaceGuard } from "./provider-support.js";

export type {
  ProviderErrorUnwrapDirect,
  ProviderErrorUnwrapManyDirect,
  ProviderErrorUnwrap,
  ProviderErrorUnwrapMany,
} from "./provider-error.js";
export type { CanonicalError } from "./provider-io-contract.js";

export interface ProviderOsErrorIsDirect extends GoInterfaceValue {
  Is(target: ProviderErrorInterface | undefined): bool;
}

export interface ProviderOsErrorIs extends GoInterfaceValue {
  Is(target: CanonicalError | undefined): Awaitable<bool>;
}

export function OsIsNotExistDirect(
  failure: ProviderErrorInterface | undefined,
  target: ProviderErrorInterface | undefined,
  isCustom: InterfaceGuard<ProviderOsErrorIsDirect>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrapDirect>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapManyDirect>,
): bool {
  return ErrorsIsDirect(
    failure,
    target,
    isCustom,
    isUnwrap,
    isUnwrapMany,
  );
}

export function OsIsNotExistCanonical(
  failure: CanonicalError | undefined,
  target: CanonicalError | undefined,
  isCustom: InterfaceGuard<ProviderOsErrorIs>,
  isUnwrap: InterfaceGuard<ProviderErrorUnwrap>,
  isUnwrapMany: InterfaceGuard<ProviderErrorUnwrapMany>,
): Promise<bool> {
  return ErrorsIsCanonical(
    failure,
    target,
    isCustom,
    isUnwrap,
    isUnwrapMany,
  );
}
