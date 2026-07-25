import type { bool } from "@tsonic/core/types.js";
import { flip } from "./logic.js";
export function Run(input: bool): bool {
    return flip(input);
}
export function Again(input: bool): bool {
    return flip(flip(input));
}
export function identity(input: bool): bool {
    return input;
}
