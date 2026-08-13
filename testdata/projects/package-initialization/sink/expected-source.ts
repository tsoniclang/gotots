import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../../packages/example.com/package-initialization/sink/state.js";
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
