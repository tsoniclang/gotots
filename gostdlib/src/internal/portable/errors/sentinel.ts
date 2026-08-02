import type { GoError } from "@gotots/runtime/interface-value.js";

import { ProviderError } from "../../runtime/error.js";

export const closed: GoError = new ProviderError("file already closed");
export const exists: GoError = new ProviderError("file already exists");
export const invalid: GoError = new ProviderError("invalid argument");
export const notExists: GoError = new ProviderError("file does not exist");
export const permission: GoError = new ProviderError("permission denied");
export const unsupported: GoError = new ProviderError("unsupported operation");
