import type { bool } from "@tsonic/core/types.js";
import { identity } from "./entry.js";
export function flip(input: bool): bool {
    return !identity(input);
}
