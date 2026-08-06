import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { uint64 } from "@gotots/gostdlib/internal/scalars.js";

import { ProviderError } from "../../runtime/error.js";

export const closed: GoError = new ProviderError("file already closed");
export const exists: GoError = new ProviderError("file already exists");
export const invalid: GoError = new ProviderError("invalid argument");
export const notExists: GoError = new ProviderError("file does not exist");
export const permission: GoError = new ProviderError("permission denied");
export const unsupported: GoError = new ProviderError("unsupported operation");

export function errnoMatchesSentinel(
  value: uint64,
  target: GoInterfaceValue | undefined,
): boolean {
  if (target === permission) {
    return value === 13n || value === 1n;
  }
  if (target === exists) {
    return value === 17n || value === 39n;
  }
  if (target === notExists) {
    return value === 2n;
  }
  if (target === unsupported) {
    return value === 38n || value === 95n;
  }
  return false;
}
