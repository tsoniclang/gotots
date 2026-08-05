import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  Awaitable,
  bool,
  gostring,
  int,
  int64,
  uint32,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";

import { New } from "../errors.js";
import {
  closed,
  exists,
  invalid,
  notExists,
  permission,
} from "../internal/portable/errors/sentinel.js";
import { WrappedProviderError } from "../internal/portable/errors/tree.js";
import { DirectoryFile } from "../internal/portable/io/filesystem.js";
import { ProviderInterfaceValue } from "../internal/portable/io/value.js";
import {
  byteSlice,
  sliceValues,
} from "../internal/runtime/slice.js";
import { state as ioState } from "../io.js";
import type { Time } from "../time.js";

const modeTypeMask = 0x8f28_0000;

export class FileMode {
  constructor(readonly value: uint32) {}

  IsDir(): bool {
    return (this.value & ModeDir.value) !== 0;
  }

  IsRegular(): bool {
    return (this.value & modeTypeMask) === 0;
  }

  Type(): FileMode {
    return new FileMode((this.value & modeTypeMask) >>> 0);
  }
}

export const ModeDir = new FileMode(0x8000_0000);
export const ModeSymlink = new FileMode(0x0800_0000);
export const ModeIrregular = new FileMode(0x0008_0000);

export interface DirEntry extends GoInterfaceValue {
  Info(): [FileInfo | undefined, GoError | undefined];
  IsDir(): bool;
  Name(): gostring;
  Type(): FileMode;
}

export interface FS extends GoInterfaceValue {
  Open(name: gostring): [File | undefined, GoError | undefined];
}

export interface File extends GoInterfaceValue {
  Close(): GoError | undefined;
  Read(buffer: RuntimeSlice<uint8>): [int, GoError | undefined];
  Stat(): [FileInfo | undefined, GoError | undefined];
}

export interface FileInfo extends GoInterfaceValue {
  IsDir(): bool;
  ModTime(): Time;
  Mode(): FileMode;
  Name(): gostring;
  Size(): int64;
  Sys(): GoInterfaceValue | undefined;
}

const pathErrorType = Object.freeze({ comparable: true });

export class PathError extends WrappedProviderError {
  constructor(
    public Op: gostring,
    public Path: gostring,
    public Err: GoError | undefined,
  ) {
    super(pathErrorType);
  }

  static Error(receiver: PathError | undefined): gostring {
    if (receiver === undefined) {
      return "<nil>";
    }
    return receiver.Error();
  }

  static $make(
    operation: gostring,
    path: gostring,
    failure: GoError | undefined,
  ): PathError {
    return new PathError(operation, path, failure);
  }

  static $storageOf(source: PathError): PathError {
    return source;
  }

  static $fromStorage(source: PathError): PathError {
    return source;
  }

  Error(): gostring {
    const detail = this.Err?.Error() ?? "<nil>";
    if (this.Op === "") {
      return `${this.Path}: ${detail}`;
    }
    if (this.Path === "") {
      return `${this.Op}: ${detail}`;
    }
    return `${this.Op} ${this.Path}: ${detail}`;
  }

  Unwrap(): GoError | undefined {
    return this.Err;
  }

  static Unwrap(receiver: PathError | undefined): GoError | undefined {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    return receiver.Unwrap();
  }
}

interface WalkDirError extends GoInterfaceValue {
  Error(): Awaitable<gostring>;
}

interface WalkDirFileInfo extends GoInterfaceValue {
  Name(): Awaitable<gostring>;
  Size(): Awaitable<int64>;
  Mode(): Awaitable<FileMode>;
  ModTime(): Awaitable<Time>;
  IsDir(): Awaitable<bool>;
  Sys(): Awaitable<GoInterfaceValue | undefined>;
}

interface WalkDirEntry extends GoInterfaceValue {
  Name(): Awaitable<gostring>;
  IsDir(): Awaitable<bool>;
  Type(): Awaitable<FileMode>;
  Info(): Awaitable<[
    WalkDirFileInfo | undefined,
    WalkDirError | undefined,
  ]>;
}

export type WalkDirFunc = ((
  path: gostring,
  entry: WalkDirEntry | undefined,
  failure: WalkDirError | undefined,
) => Awaitable<WalkDirError | undefined>) | undefined;

export const state: {
  ErrClosed: GoError;
  ErrExist: GoError;
  ErrInvalid: GoError;
  ErrNotExist: GoError;
  ErrPermission: GoError;
  SkipAll: GoError;
  SkipDir: GoError;
} = {
  ErrClosed: closed,
  ErrExist: exists,
  ErrInvalid: invalid,
  ErrNotExist: notExists,
  ErrPermission: permission,
  SkipAll: New("skip everything and stop the walk"),
  SkipDir: New("skip this directory"),
};

const dirEntryType = Object.freeze({ comparable: true });

class InfoDirEntry extends ProviderInterfaceValue implements DirEntry {
  constructor(private readonly information: FileInfo) {
    super(dirEntryType);
  }

  Info(): [FileInfo, undefined] {
    return [this.information, undefined];
  }

  IsDir(): bool {
    return this.information.IsDir();
  }

  Name(): gostring {
    return this.information.Name();
  }

  Type(): FileMode {
    return this.information.Mode().Type();
  }
}

export function FileInfoToDirEntry(information: FileInfo | undefined): DirEntry | undefined {
  return information === undefined ? undefined : new InfoDirEntry(information);
}

export function ReadDir(
  fileSystem: FS | undefined,
  name: gostring,
): [RuntimeSlice<DirEntry | undefined>, GoError | undefined] {
  const [file, openFailure] = open(fileSystem, name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<DirEntry | undefined>(), openFailure];
  }
  if (!(file instanceof DirectoryFile)) {
    file.Close();
    return [
      RuntimeSlice.nil<DirEntry | undefined>(),
      new PathError("readdir", name, state.ErrInvalid),
    ];
  }
  const [entries, readFailure] = file.ReadDir(-1);
  const closeFailure = file.Close();
  const values = sliceValues(entries);
  values.sort((left, right): number => {
    const leftName = left?.Name() ?? "";
    const rightName = right?.Name() ?? "";
    return leftName < rightName ? -1 : leftName > rightName ? 1 : 0;
  });
  return [
    RuntimeSlice.literal(values),
    readFailure ?? closeFailure,
  ];
}

export function ReadFile(
  fileSystem: FS | undefined,
  name: gostring,
): [RuntimeSlice<uint8>, GoError | undefined] {
  const [file, openFailure] = open(fileSystem, name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<uint8>(), openFailure];
  }

  const values: number[] = [];
  let failure: GoError | undefined;
  for (;;) {
    const buffer = RuntimeSlice.make<uint8>(32 * 1024, 32 * 1024, 0);
    const [count, readFailure] = file.Read(buffer);
    for (let index = 0; index < count; index += 1) {
      values.push(buffer.get(index));
    }
    if (readFailure !== undefined) {
      if (readFailure !== ioState.EOF) {
        failure = readFailure;
      }
      break;
    }
    if (count === 0n) {
      failure = New("multiple Read calls return no data or error");
      break;
    }
  }
  const closeFailure = file.Close();
  return [byteSlice(values), failure ?? closeFailure];
}

export function Stat(
  fileSystem: FS | undefined,
  name: gostring,
): [FileInfo | undefined, GoError | undefined] {
  const [file, openFailure] = open(fileSystem, name);
  if (openFailure !== undefined || file === undefined) {
    return [undefined, openFailure];
  }
  const [information, statFailure] = file.Stat();
  const closeFailure = file.Close();
  return [information, statFailure ?? closeFailure];
}

export async function WalkDir(
  fileSystem: FS | undefined,
  root: gostring,
  visit: WalkDirFunc,
): Promise<WalkDirError | undefined> {
  const [information, statFailure] = Stat(fileSystem, root);
  const rootEntry = FileInfoToDirEntry(information);
  if (statFailure !== undefined || rootEntry === undefined) {
    return invokeWalkDir(visit, root, undefined, statFailure);
  }

  const failure = await walk(fileSystem, root, rootEntry, visit);
  if (failure === state.SkipAll) {
    return undefined;
  }
  if (failure === state.SkipDir && rootEntry.IsDir()) {
    return undefined;
  }
  return failure;
}

async function walk(
  fileSystem: FS | undefined,
  path: string,
  entry: DirEntry,
  visit: WalkDirFunc,
): Promise<WalkDirError | undefined> {
  const visitFailure = await invokeWalkDir(visit, path, entry, undefined);
  if (visitFailure !== undefined) {
    return visitFailure;
  }
  if (!entry.IsDir()) {
    return undefined;
  }

  const [entries, readFailure] = ReadDir(fileSystem, path);
  if (readFailure !== undefined) {
    return invokeWalkDir(visit, path, entry, readFailure);
  }
  for (const child of sliceValues(entries)) {
    if (child === undefined) {
      continue;
    }
    const childPath = path === "." ? child.Name() : `${path}/${child.Name()}`;
    const childFailure = await walk(fileSystem, childPath, child, visit);
    if (childFailure === state.SkipDir && child.IsDir()) {
      continue;
    }
    if (childFailure !== undefined) {
      return childFailure;
    }
  }
  return undefined;
}

async function invokeWalkDir(
  visit: WalkDirFunc,
  path: gostring,
  entry: WalkDirEntry | undefined,
  failure: WalkDirError | undefined,
): Promise<WalkDirError | undefined> {
  if (visit === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return await visit(path, entry, failure);
}

function open(
  fileSystem: FS | undefined,
  name: string,
): [File | undefined, GoError | undefined] {
  if (fileSystem === undefined) {
    return [undefined, new PathError("open", name, state.ErrInvalid)];
  }
  return fileSystem.Open(name);
}
