import { init } from "../../../../modules/example.com/package-initialization/sideeffect/a_init.js";
import { init__shadow_1 } from "../../../../modules/example.com/package-initialization/sideeffect/z_init.js";
import { Mark as Mark__from_sink, Pair as Pair__from_sink } from "../sink/package.js";
import { $state } from "./state.js";
export function $initialize(): void {
    $state.first = 0;
    $state.second = 0;
    {
        Mark__from_sink(3);
    }
    {
        const results = Pair__from_sink();
        $state.first = results[0];
        $state.second = results[1];
    }
    init();
    init__shadow_1();
}
