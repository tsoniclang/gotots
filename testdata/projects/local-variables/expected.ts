import type { int32 } from "../../../support/scalars.js";
export function Compute(input: int32): int32 {
    let base: int32 = input;
    {
        let base__shadow_1: int32 = (base + (1 as int32)) | 0;
        let left: int32 = base__shadow_1, right: int32 = (base__shadow_1 + (1 as int32)) | 0;
        let __u3c0_: int32 = (left + right) | 0;
        const __gotots_assign_0: int32 = right;
        const __gotots_assign_1: int32 = left;
        left = __gotots_assign_0;
        right = __gotots_assign_1;
        return __u3c0_;
    }
}
export function LateOuter(input: int32): int32 {
    {
        let value__shadow_1: int32 = (input + (1 as int32)) | 0;
        input = value__shadow_1;
    }
    let value: int32 = (input + (2 as int32)) | 0;
    let __go_class: int32 = (value + (3 as int32)) | 0;
    let __go_arguments: int32 = (__go_class + (4 as int32)) | 0;
    return __go_arguments;
}
