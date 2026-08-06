import { GoPanic } from "./panic.js";
export class GoDenseIndex {
    private static present<T>(values: readonly T[], index: number, value: T | undefined): value is T {
        return index in values;
    }
    public static get<T>(values: readonly T[], index: number): T {
        const value = values[index];
        if (!GoDenseIndex.present(values, index, value))
            GoPanic.raiseRuntime("dense storage index is absent");
        return value;
    }
}
