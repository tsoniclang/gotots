import type { int32 } from "../../../support/scalars.js";
import { $state } from "../../../packages/f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af/api/state.js";
import { Read as Read__from_sink } from "../../../packages/f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af/sink/package.js";
export function Run(): int32 {
    return $state.Observed + Read__from_sink();
}
