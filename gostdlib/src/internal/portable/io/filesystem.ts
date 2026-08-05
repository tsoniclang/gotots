import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import type { DirEntry, File, FileInfo } from "../../../io/fs.js";
import { ProviderInterfaceValue } from "./value.js";

const directoryFileType = Object.freeze({ comparable: true });

export abstract class DirectoryFile extends ProviderInterfaceValue implements File {
  protected constructor() {
    super(directoryFileType);
  }

  abstract Close(): GoError | undefined;

  abstract Read(buffer: RuntimeSlice<uint8>): [int, GoError | undefined];

  abstract Stat(): [FileInfo | undefined, GoError | undefined];

  abstract ReadDir(count: number): [
    RuntimeSlice<DirEntry | undefined>,
    GoError | undefined,
  ];
}
