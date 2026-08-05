import { Builder } from "../../strings.js";
import {
  builderRepresentationAssign,
  builderRepresentationCopy,
} from "../portable/strings/builder.js";

export type StringsBuilderStorage = Builder;

export class StringsBuilderOperations {
  static $zero(): Builder {
    return new Builder();
  }

  static $copy(source: Builder): Builder {
    return builderRepresentationCopy(source);
  }

  static $assign(target: Builder, source: Builder): void {
    builderRepresentationAssign(target, source);
  }

  static $storageOf(source: Builder): StringsBuilderStorage {
    return source;
  }

  static $fromStorage(source: StringsBuilderStorage): Builder {
    return source;
  }
}
