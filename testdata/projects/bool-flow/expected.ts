import type { bool } from "../../../support/scalars.js";
export function Run(input: bool): bool {
    let current = false;
    if (!input) {
        current = Flip(true);
    }
    else {
        current = Same(input, true);
    }
    return current;
}
export function Flip(value: bool): bool {
    return !value;
}
export function Same(left: bool, right: bool): bool {
    return left === right;
}
