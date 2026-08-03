import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  Awaitable,
  gostring,
  int64,
  uint8,
} from "@gotots/runtime/scalars.js";

import { New } from "../../errors.js";
import type { FileMode } from "../../io/fs.js";
import {
  PathError,
} from "../../io/fs.js";
import type { Time } from "../../time.js";
import { Join as JoinPath } from "../../path.js";
import type { CanonicalErrorAsync } from "./provider-io-contract.js";
import {
  byteSlice,
  sliceValues,
} from "../runtime/slice.js";
import { goInterfaceEqual } from "../runtime/interface.js";
import { ProviderInterfaceValue } from "../portable/io/value.js";

type InterfaceGuard<Value extends GoInterfaceValue> = (
  value: GoInterfaceValue | undefined,
) => value is Value;

export interface CanonicalFileInfo extends GoInterfaceValue {
  IsDir(recovery?: GoRecovery): Awaitable<boolean>;
  ModTime(recovery?: GoRecovery): Awaitable<Time>;
  Mode(recovery?: GoRecovery): Awaitable<FileMode>;
  Name(recovery?: GoRecovery): Awaitable<gostring>;
  Size(recovery?: GoRecovery): Awaitable<int64>;
  Sys(recovery?: GoRecovery): Awaitable<GoInterfaceValue | undefined>;
}

const canonicalInfoDirEntryType = Object.freeze({ comparable: true });

class CanonicalInfoDirEntry<
  Failure extends GoInterfaceValue,
> extends ProviderInterfaceValue {
  override readonly $go$methods: ReadonlySet<object>;

  constructor(
    private readonly information: CanonicalFileInfo,
    contract: readonly object[],
  ) {
    super(canonicalInfoDirEntryType);
    this.$go$methods = new Set(contract);
  }

  Info(): Awaitable<[CanonicalFileInfo, Failure | undefined]> {
    return [this.information, undefined];
  }

  IsDir(): Awaitable<boolean> {
    return this.information.IsDir();
  }

  Name(): Awaitable<gostring> {
    return this.information.Name();
  }

  async Type(): Promise<FileMode> {
    return (await this.information.Mode()).Type();
  }
}

interface FromProviderBridge<
  Provider extends GoInterfaceValue,
  Canonical extends GoInterfaceValue,
> {
  $from(value: Provider | undefined): Canonical | undefined;
}

export interface CanonicalFileAsyncError extends GoInterfaceValue {
  Close(recovery?: GoRecovery): Awaitable<CanonicalErrorAsync | undefined>;
  Read(
    buffer: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Awaitable<[int64, CanonicalErrorAsync | undefined]>;
  Stat(
    recovery?: GoRecovery,
  ): Awaitable<[CanonicalFileInfo | undefined, CanonicalErrorAsync | undefined]>;
}

export interface CanonicalFSAsyncError extends GoInterfaceValue {
  Open(
    name: gostring,
    recovery?: GoRecovery,
  ): Awaitable<[
    CanonicalFileAsyncError | undefined,
    CanonicalErrorAsync | undefined,
  ]>;
}

export interface CanonicalReadFileFSAsyncError extends CanonicalFSAsyncError {
  ReadFile(
    name: gostring,
    recovery?: GoRecovery,
  ): Awaitable<[RuntimeSlice<uint8>, CanonicalErrorAsync | undefined]>;
}

export interface CanonicalDirEntryAsyncError extends GoInterfaceValue {
  Info(
    recovery?: GoRecovery,
  ): Awaitable<[CanonicalFileInfo | undefined, CanonicalErrorAsync | undefined]>;
  IsDir(recovery?: GoRecovery): Awaitable<boolean>;
  Name(recovery?: GoRecovery): Awaitable<gostring>;
  Type(recovery?: GoRecovery): Awaitable<FileMode>;
}

export interface CanonicalReadDirFSAsyncError extends CanonicalFSAsyncError {
  ReadDir(
    name: gostring,
    recovery?: GoRecovery,
  ): Awaitable<[
    RuntimeSlice<CanonicalDirEntryAsyncError | undefined>,
    CanonicalErrorAsync | undefined,
  ]>;
}

export interface CanonicalReadDirFileAsyncError extends CanonicalFileAsyncError {
  ReadDir(
    count: int64,
    recovery?: GoRecovery,
  ): Awaitable<[
    RuntimeSlice<CanonicalDirEntryAsyncError | undefined>,
    CanonicalErrorAsync | undefined,
  ]>;
}

type CanonicalWalkDirFuncAsyncError = ((
  path: gostring,
  entry: CanonicalDirEntryAsyncError | undefined,
  failure: CanonicalErrorAsync | undefined,
  recovery?: GoRecovery,
) => Awaitable<CanonicalErrorAsync | undefined>) | undefined;

export type { CanonicalErrorAsync } from "./provider-io-contract.js";

export interface CanonicalStatFSAsyncError extends CanonicalFSAsyncError {
  Stat(
    name: gostring,
    recovery?: GoRecovery,
  ): Awaitable<[CanonicalFileInfo | undefined, CanonicalErrorAsync | undefined]>;
}

export async function IoFsReadFileCanonicalAsyncError(
  fileSystem: CanonicalFSAsyncError | undefined,
  name: gostring,
  eof: CanonicalErrorAsync | undefined,
  isReadFileFS: InterfaceGuard<CanonicalReadFileFSAsyncError>,
): Promise<[RuntimeSlice<uint8>, CanonicalErrorAsync | undefined]> {
  if (fileSystem === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  if (isReadFileFS(fileSystem)) {
    return fileSystem.ReadFile(name);
  }
  const [file, openFailure] = await fileSystem.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<uint8>(), openFailure];
  }
  try {
    const values: number[] = [];
    for (;;) {
      const buffer = RuntimeSlice.make<uint8>(32 * 1024, 32 * 1024, 0);
      const [count, readFailure] = await file.Read(buffer);
      for (let index = 0; index < count; index += 1) {
        values.push(buffer.get(index));
      }
      if (readFailure !== undefined) {
        return [
          byteSlice(values),
          goInterfaceEqual(readFailure, eof) ? undefined : readFailure,
        ];
      }
    }
  } finally {
    await file.Close();
  }
}

export async function IoFsStatCanonicalAsyncError(
  fileSystem: CanonicalFSAsyncError | undefined,
  name: gostring,
  isStatFS: InterfaceGuard<CanonicalStatFSAsyncError>,
): Promise<[CanonicalFileInfo | undefined, CanonicalErrorAsync | undefined]> {
  if (fileSystem === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  if (isStatFS(fileSystem)) {
    return fileSystem.Stat(name);
  }
  const [file, openFailure] = await fileSystem.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [undefined, openFailure];
  }
  try {
    return await file.Stat();
  } finally {
    await file.Close();
  }
}

export async function IoFsReadDirCanonicalAsyncError(
  fileSystem: CanonicalFSAsyncError | undefined,
  name: gostring,
  isReadDirFS: InterfaceGuard<CanonicalReadDirFSAsyncError>,
  isReadDirFile: InterfaceGuard<CanonicalReadDirFileAsyncError>,
  errorBridge: FromProviderBridge<GoError, CanonicalErrorAsync>,
): Promise<[
  RuntimeSlice<CanonicalDirEntryAsyncError | undefined>,
  CanonicalErrorAsync | undefined,
]> {
  if (fileSystem === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  if (isReadDirFS(fileSystem)) {
    return fileSystem.ReadDir(name);
  }
  const [file, openFailure] = await fileSystem.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<CanonicalDirEntryAsyncError | undefined>(), openFailure];
  }
  try {
    if (!isReadDirFile(file)) {
      return [
        RuntimeSlice.nil<CanonicalDirEntryAsyncError | undefined>(),
        requireBridgedValue(
          errorBridge.$from(
            new PathError("readdir", name, New("not implemented")),
          ),
          "io/fs.PathError",
        ),
      ];
    }
    const [entries, readFailure] = await file.ReadDir(-1);
    const values = await sortDirectoryEntries(sliceValues(entries));
    return [RuntimeSlice.literal(values), readFailure];
  } finally {
    await file.Close();
  }
}

export async function IoFsWalkDirCanonicalAsyncError(
  fileSystem: CanonicalFSAsyncError | undefined,
  root: gostring,
  visit: CanonicalWalkDirFuncAsyncError,
  skipAll: CanonicalErrorAsync | undefined,
  skipDir: CanonicalErrorAsync | undefined,
  isReadDirFS: InterfaceGuard<CanonicalReadDirFSAsyncError>,
  isReadDirFile: InterfaceGuard<CanonicalReadDirFileAsyncError>,
  isStatFS: InterfaceGuard<CanonicalStatFSAsyncError>,
  errorBridge: FromProviderBridge<GoError, CanonicalErrorAsync>,
  dirEntryContract: readonly object[],
): Promise<CanonicalErrorAsync | undefined> {
  const [information, statFailure] = await IoFsStatCanonicalAsyncError(
    fileSystem,
    root,
    isStatFS,
  );
  let failure: CanonicalErrorAsync | undefined;
  if (statFailure !== undefined) {
    failure = await invokeWalkDirAsyncError(visit, root, undefined, statFailure);
  } else {
    const entry = information === undefined
      ? undefined
      : new CanonicalInfoDirEntry<CanonicalErrorAsync>(
        information,
        dirEntryContract,
      );
    failure = await walkAsyncError(
      fileSystem,
      root,
      entry,
      visit,
      skipDir,
      isReadDirFS,
      isReadDirFile,
      errorBridge,
    );
  }
  return goInterfaceEqual(failure, skipDir) || goInterfaceEqual(failure, skipAll)
    ? undefined
    : failure;
}

async function walkAsyncError(
  fileSystem: CanonicalFSAsyncError | undefined,
  path: gostring,
  entry: CanonicalDirEntryAsyncError | undefined,
  visit: CanonicalWalkDirFuncAsyncError,
  skipDir: CanonicalErrorAsync | undefined,
  isReadDirFS: InterfaceGuard<CanonicalReadDirFSAsyncError>,
  isReadDirFile: InterfaceGuard<CanonicalReadDirFileAsyncError>,
  errorBridge: FromProviderBridge<GoError, CanonicalErrorAsync>,
): Promise<CanonicalErrorAsync | undefined> {
  let failure = await invokeWalkDirAsyncError(visit, path, entry, undefined);
  if (entry === undefined) {
    if (failure !== undefined) {
      return failure;
    }
    return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const isDirectory = await entry.IsDir();
  if (failure !== undefined || !isDirectory) {
    if (goInterfaceEqual(failure, skipDir) && isDirectory) {
      failure = undefined;
    }
    return failure;
  }
  const [entries, readFailure] = await IoFsReadDirCanonicalAsyncError(
    fileSystem,
    path,
    isReadDirFS,
    isReadDirFile,
    errorBridge,
  );
  if (readFailure !== undefined) {
    failure = await invokeWalkDirAsyncError(visit, path, entry, readFailure);
    if (failure !== undefined) {
      return goInterfaceEqual(failure, skipDir) && isDirectory
        ? undefined
        : failure;
    }
  }
  for (const child of sliceValues(entries)) {
    if (child === undefined) {
      return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    const childFailure = await walkAsyncError(
      fileSystem,
      JoinPath(RuntimeSlice.literal([path, await child.Name()])),
      child,
      visit,
      skipDir,
      isReadDirFS,
      isReadDirFile,
      errorBridge,
    );
    if (goInterfaceEqual(childFailure, skipDir)) {
      break;
    }
    if (childFailure !== undefined) {
      return childFailure;
    }
  }
  return undefined;
}

async function invokeWalkDirAsyncError(
  visit: CanonicalWalkDirFuncAsyncError,
  path: gostring,
  entry: CanonicalDirEntryAsyncError | undefined,
  failure: CanonicalErrorAsync | undefined,
): Promise<CanonicalErrorAsync | undefined> {
  if (visit === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return await visit(path, entry, failure);
}

function requireBridgedValue<Value extends GoInterfaceValue>(
  value: Value | undefined,
  source: string,
): Value {
  if (value === undefined) {
    GoPanic.raiseRuntime(`provider bridge discarded non-nil ${source}`);
  }
  return value;
}

async function sortDirectoryEntries(
  entries: (CanonicalDirEntryAsyncError | undefined)[],
): Promise<(CanonicalDirEntryAsyncError | undefined)[]> {
  const named = await Promise.all(entries.map(async (entry) => ({
    entry,
    name: entry === undefined ? "" : await entry.Name(),
  })));
  named.sort((left, right) =>
    left.name < right.name ? -1 : left.name > right.name ? 1 : 0);
  return named.map(({ entry }) => entry);
}
