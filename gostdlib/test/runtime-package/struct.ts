export class GoEmptyStruct {
    declare private readonly $go$emptyStruct: void;
    public constructor() {
    }
    static $zero(): GoEmptyStruct {
        return new GoEmptyStruct;
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
    static $storageOf($source: GoEmptyStruct): GoEmptyStruct {
        return $source;
    }
    static $fromStorage($source: GoEmptyStruct): GoEmptyStruct {
        return $source;
    }
    declare private readonly then?: never;
}
