import type { bool } from "../../../support/scalars.js";
import { identity } from "./entry.js";
export function flip(input: bool): bool {
    return !identity(input);
}
