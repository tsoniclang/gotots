import type { int32 } from "@gotots/runtime/scalars.js";
export function Compute(input: int32): int32 {
    let base = input;
    {
        let base__shadow_1 = base + 1;
        let left = base__shadow_1, right = base__shadow_1 + 1;
        let __u3c0_ = left + right;
        const __gotots_assign_0 = right;
        const __gotots_assign_1 = left;
        left = __gotots_assign_0;
        right = __gotots_assign_1;
        return __u3c0_;
    }
}
export function LateOuter(input: int32): int32 {
    {
        let value__shadow_1 = input + 1;
        input = value__shadow_1;
    }
    let value = input + 2;
    let __go_class = value + 3;
    let __go_arguments = __go_class + 4;
    return __go_arguments;
}
