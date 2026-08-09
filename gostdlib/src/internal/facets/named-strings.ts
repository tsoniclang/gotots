import { Builder, Reader, Replacer } from "../../strings.js";
import {
  builderRepresentationAssign,
  builderRepresentationCopy,
} from "../portable/strings/builder.js";
import {
  readerRepresentationAssign,
  readerRepresentationCopy,
  readerRepresentationEqual,
  readerRepresentationHash,
} from "../portable/strings/reader.js";
import {
  replacerRepresentationAssign,
  replacerRepresentationCopy,
} from "../portable/strings/replacer.js";

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

export class StringsReaderOperations {
  static $copy(source: Reader): Reader {
    return readerRepresentationCopy(source);
  }

  static $assign(target: Reader, source: Reader): void {
    readerRepresentationAssign(target, source);
  }

  static $equal(left: Reader, right: Reader): boolean {
    return readerRepresentationEqual(left, right);
  }

  static $hash(source: Reader): number {
    return readerRepresentationHash(source);
  }
}

export class StringsReplacerOperations {
  static $copy(source: Replacer): Replacer {
    return replacerRepresentationCopy(source);
  }

  static $assign(target: Replacer, source: Replacer): void {
    replacerRepresentationAssign(target, source);
  }
}
