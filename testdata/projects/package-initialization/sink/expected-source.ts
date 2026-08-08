import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../packages/f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af/sink/state.js";
export function Mark(value: int32): int32 {
    $state.Count = $state.Count * 10 + value;
    return $state.Count;
}
export function Pair(): [
    int32,
    int32
] {
    return [Mark(4), Mark(5)];
}
export function Read(): int32 {
    return $state.Count;
}
