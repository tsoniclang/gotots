import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";

import { ProviderError } from "./error.js";

// providerPlaceholder is the one canonical typed throwing boundary for a
// selected provider behavior that is not implemented. Certification
// classifies a checked implementation body as a placeholder when its call
// evidence reaches this symbol; the message is runtime diagnostics only and
// never selects behavior.
export function providerPlaceholder(message: string): never {
  return GoPanic.raise(new ProviderError(message));
}

// providerPlaceholderError is the canonical typed error form of the same
// boundary for Go contracts that report an unimplemented provider behavior
// through an error result instead of a panic.
export function providerPlaceholderError(message: string): GoError {
  return new ProviderError(message);
}

// providerPlaceholderMessage is the canonical placeholder form for profile
// facades whose contract requires wrapping the boundary in their own typed
// error carrier. Certification classifies by this symbol's identity, never
// by the message content.
export function providerPlaceholderMessage(message: string): string {
  return message;
}
