import { Builder } from "../../strings.js";

export type StringsBuilderStorage = Builder;

export class StringsBuilderOperations {
  static $zero(): Builder {
    return new Builder();
  }

  static $copy(_source: Builder): Builder {
    return new Builder();
  }

  static $storageOf(source: Builder): StringsBuilderStorage {
    return source;
  }

  static $fromStorage(source: StringsBuilderStorage): Builder {
    return source;
  }
}
