import { $init_0 } from "../../../modules/f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af/sideeffect/a_init.js";
import { $init_1 } from "../../../modules/f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af/sideeffect/z_init.js";
import { Mark as Mark__from_sink, Pair as Pair__from_sink } from "../sink/package.js";
import { $state } from "./state.js";
export function $initialize(): void {
    $state.first = 0;
    $state.second = 0;
    Mark__from_sink(3);
    const __gotots_results_0 = Pair__from_sink();
    $state.first = __gotots_results_0[0];
    $state.second = __gotots_results_0[1];
    $init_0();
    $init_1();
}
