import type { int64 } from "@tsonic/core/types.js";
export function Small(): int64 {
    return 42 as int64;
}
export function BeyondSafe(): int64 {
    return (2097152 as int64) * (4294967296 as int64) + (1 as int64);
}
export function Maximum(): int64 {
    return (2147483647 as int64) * (4294967296 as int64) + (4294967295 as int64);
}
export function Minimum(): int64 {
    return ((0 as int64) - (2147483647 as int64) - (1 as int64)) * (4294967296 as int64);
}
