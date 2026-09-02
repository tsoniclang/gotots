import type { int32 } from "@gotots/runtime/scalars.js";
export function Compute(input: int32): int32 {
    let base = input;
    {
        let base__shadow_1 = base + 1 | 0;
        let left = base__shadow_1, right = base__shadow_1 + 1 | 0;
        let __u3c0_ = left + right | 0;
        const assignmentValue = right;
        const assignmentValue2 = left;
        left = assignmentValue;
        right = assignmentValue2;
        return __u3c0_;
    }
}
export function LateOuter(input: int32): int32 {
    {
        let value__shadow_1 = input + 1 | 0;
        input = value__shadow_1;
    }
    let value = input + 2 | 0;
    let __go_class = value + 3 | 0;
    let __go_arguments = __go_class + 4 | 0;
    return __go_arguments;
}
