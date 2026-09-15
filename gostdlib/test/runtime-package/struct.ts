import type { $goContainerStorageType, $goStorageType } from "./storage.js";
export class GoEmptyStruct {
    declare readonly [$goStorageType]: {};
    declare readonly [$goContainerStorageType]: {};
    declare private readonly $go$emptyStruct: void;
    public constructor() {
    }
    static $zero(): GoEmptyStruct {
        return new GoEmptyStruct;
    }
    static $assign($target: GoEmptyStruct, $source: GoEmptyStruct): void {
    }
    static $copy($source: GoEmptyStruct): GoEmptyStruct {
        return $source;
    }
    static $equal($left: GoEmptyStruct, $right: GoEmptyStruct): boolean {
        return true;
    }
    static $hash($source: GoEmptyStruct): number {
        return 2166136261;
    }
    static $convert($source: object): GoEmptyStruct {
        return new GoEmptyStruct;
    }
    static $storageOf($source: GoEmptyStruct): {} {
        return {};
    }
    static $fromStorage($source: {}): GoEmptyStruct {
        return new GoEmptyStruct;
    }
    declare private readonly then?: never;
}
