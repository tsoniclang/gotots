import { Read as Read__from_sink } from "../sink/package.js";
import { $state } from "./state.js";
export function $initialize(): void {
    $state.Observed = 0;
    {
        $state.Observed = Read__from_sink();
    }
}
export { Run } from "../../../modules/f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af/api/api.js";
export { $state };
