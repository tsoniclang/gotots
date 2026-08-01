import type { GoError } from "@gotots/runtime/interface-value.js";

import { ProviderError } from "../../../runtime/error.js";

export function unsupportedReflectionOperation(
  operation: "Read" | "Write",
): GoError {
  return new ProviderError(
    `encoding/binary.${operation} requires generated reflection metadata`,
  );
}
