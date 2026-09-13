import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { uint64 } from "@gotots/gostdlib/internal/scalars.js";

import { ProviderError } from "../../runtime/error.js";

export const closed: GoError = ProviderError.fromText("file already closed");
export const exists: GoError = ProviderError.fromText("file already exists");
export const invalid: GoError = ProviderError.fromText("invalid argument");
export const notExists: GoError = ProviderError.fromText("file does not exist");
export const permission: GoError = ProviderError.fromText("permission denied");
export const unsupported: GoError = ProviderError.fromText("unsupported operation");

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
