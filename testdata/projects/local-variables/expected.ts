import type { int64 } from "@tsonic/core/types.js";
export function Compute(input: int64): int64 {
    let base: int64 = input;
    {
        let base__shadow_1: int64 = base + (1 as int64);
        let left: int64 = base__shadow_1, right: int64 = base__shadow_1 + (1 as int64);
        let __u3c0_: int64 = left + right;
        const __gotots_assign_0: int64 = right;
        const __gotots_assign_1: int64 = left;
        left = __gotots_assign_0;
        right = __gotots_assign_1;
        return __u3c0_;
    }
}
export function LateOuter(input: int64): int64 {
    {
        let value__shadow_1: int64 = input + (1 as int64);
        input = value__shadow_1;
    }
    let value: int64 = input + (2 as int64);
    let __go_class: int64 = value + (3 as int64);
    let __go_arguments: int64 = __go_class + (4 as int64);
    return __go_arguments;
}
