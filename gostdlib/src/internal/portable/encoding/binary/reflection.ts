import type { GoError } from "@gotots/runtime/interface-value.js";

import { ProviderError } from "../../../runtime/error.js";

export function unsupportedReflectionOperation(
  operation: "Read" | "Write",
): GoError {
  return new ProviderError(unsupportedReflectionMessage(operation));
}

export function unsupportedReflectionMessage(
  operation: "Read" | "Write",
): string {
  return `encoding/binary.${operation} requires generated reflection metadata`;
}
