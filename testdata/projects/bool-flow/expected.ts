import type { bool } from "@tsonic/core/types.js";
export function Run(input: bool): bool {
    let current: bool = false as bool;
    if (!input) {
        current = Flip(true as bool);
    }
    else {
        current = Same(input, true as bool);
    }
    return current;
}
export function Flip(value: bool): bool {
    return !value;
}
export function Same(left: bool, right: bool): bool {
    return left === right;
}
