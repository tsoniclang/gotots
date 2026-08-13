import { Read as Read__from_sink } from "../sink/package.js";
import { $state } from "./state.js";
export function $initialize(): void {
    $state.Observed = 0;
    {
        $state.Observed = Read__from_sink();
    }
}
export { Run } from "../../../../modules/example.com/package-initialization/api/api.js";
export { $state };
