import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { bool } from "@gotots/gostdlib/internal/scalars.js";

import {
  ErrorsIsDirect,
  type ProviderErrorInterface,
  type ProviderErrorUnwrapDirect,
  type ProviderErrorUnwrapManyDirect,
} from "./provider-error.js";
import type { InterfaceGuard } from "./provider-support.js";

export type {
  ProviderErrorUnwrapDirect,
  ProviderErrorUnwrapManyDirect,
} from "./provider-error.js";

export interface ProviderOsErrorIsDirect extends GoInterfaceValue {
  Is(target: ProviderErrorInterface | undefined): bool;
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
