import type { bool } from "../../../runtime/scalars.js";
import { identity } from "./entry.js";
export function flip(input: bool): bool {
    return !identity(input);
}
