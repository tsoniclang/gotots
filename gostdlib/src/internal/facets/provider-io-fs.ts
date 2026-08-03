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
import type { CanonicalError } from "./provider-io-contract.js";
import {
  byteSlice,
  sliceValues,
} from "../runtime/slice.js";
import { goInterfaceEqual } from "../runtime/interface.js";
import { ProviderInterfaceValue } from "../portable/io/value.js";
import type {
  FromProviderBridge,
  InterfaceContract,
  InterfaceGuard,
} from "./provider-support.js";

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

export interface CanonicalFile extends GoInterfaceValue {
  Close(recovery?: GoRecovery): Awaitable<CanonicalError | undefined>;
  Read(
    buffer: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Awaitable<[int64, CanonicalError | undefined]>;
  Stat(
    recovery?: GoRecovery,
  ): Awaitable<[CanonicalFileInfo | undefined, CanonicalError | undefined]>;
}

export interface CanonicalFS extends GoInterfaceValue {
  Open(
    name: gostring,
    recovery?: GoRecovery,
  ): Awaitable<[
    CanonicalFile | undefined,
    CanonicalError | undefined,
  ]>;
}

export interface CanonicalReadFileFS extends CanonicalFS {
  ReadFile(
    name: gostring,
    recovery?: GoRecovery,
  ): Awaitable<[RuntimeSlice<uint8>, CanonicalError | undefined]>;
}

export interface CanonicalDirEntry extends GoInterfaceValue {
  Info(
    recovery?: GoRecovery,
  ): Awaitable<[CanonicalFileInfo | undefined, CanonicalError | undefined]>;
  IsDir(recovery?: GoRecovery): Awaitable<boolean>;
  Name(recovery?: GoRecovery): Awaitable<gostring>;
  Type(recovery?: GoRecovery): Awaitable<FileMode>;
}

export interface CanonicalReadDirFS extends CanonicalFS {
  ReadDir(
    name: gostring,
    recovery?: GoRecovery,
  ): Awaitable<[
    RuntimeSlice<CanonicalDirEntry | undefined>,
    CanonicalError | undefined,
  ]>;
}

export interface CanonicalReadDirFile extends CanonicalFile {
  ReadDir(
    count: int64,
    recovery?: GoRecovery,
  ): Awaitable<[
    RuntimeSlice<CanonicalDirEntry | undefined>,
    CanonicalError | undefined,
  ]>;
}

type CanonicalWalkDirFunc = ((
  path: gostring,
  entry: CanonicalDirEntry | undefined,
  failure: CanonicalError | undefined,
  recovery?: GoRecovery,
) => Awaitable<CanonicalError | undefined>) | undefined;

export type { CanonicalError } from "./provider-io-contract.js";

export interface CanonicalStatFS extends CanonicalFS {
  Stat(
    name: gostring,
    recovery?: GoRecovery,
  ): Awaitable<[CanonicalFileInfo | undefined, CanonicalError | undefined]>;
}

export function IoFsFileInfoToDirEntryCanonical(
  information: CanonicalFileInfo | undefined,
  dirEntryContract: InterfaceContract,
): CanonicalDirEntry | undefined {
  return information === undefined
    ? undefined
    : new CanonicalInfoDirEntry<CanonicalError>(information, dirEntryContract);
}

export async function IoFsReadFileCanonical(
  fileSystem: CanonicalFS | undefined,
  name: gostring,
  eof: CanonicalError | undefined,
  isReadFileFS: InterfaceGuard<CanonicalReadFileFS>,
): Promise<[RuntimeSlice<uint8>, CanonicalError | undefined]> {
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

export async function IoFsStatCanonical(
  fileSystem: CanonicalFS | undefined,
  name: gostring,
  isStatFS: InterfaceGuard<CanonicalStatFS>,
): Promise<[CanonicalFileInfo | undefined, CanonicalError | undefined]> {
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

export async function IoFsReadDirCanonical(
  fileSystem: CanonicalFS | undefined,
  name: gostring,
  isReadDirFS: InterfaceGuard<CanonicalReadDirFS>,
  isReadDirFile: InterfaceGuard<CanonicalReadDirFile>,
  errorBridge: FromProviderBridge<GoError, CanonicalError>,
): Promise<[
  RuntimeSlice<CanonicalDirEntry | undefined>,
  CanonicalError | undefined,
]> {
  if (fileSystem === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  if (isReadDirFS(fileSystem)) {
    return fileSystem.ReadDir(name);
  }
  const [file, openFailure] = await fileSystem.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<CanonicalDirEntry | undefined>(), openFailure];
  }
  try {
    if (!isReadDirFile(file)) {
      return [
        RuntimeSlice.nil<CanonicalDirEntry | undefined>(),
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

export async function IoFsWalkDirCanonical(
  fileSystem: CanonicalFS | undefined,
  root: gostring,
  visit: CanonicalWalkDirFunc,
  skipAll: CanonicalError | undefined,
  skipDir: CanonicalError | undefined,
  isReadDirFS: InterfaceGuard<CanonicalReadDirFS>,
  isReadDirFile: InterfaceGuard<CanonicalReadDirFile>,
  isStatFS: InterfaceGuard<CanonicalStatFS>,
  dirEntryContract: InterfaceContract,
  errorBridge: FromProviderBridge<GoError, CanonicalError>,
): Promise<CanonicalError | undefined> {
  const [information, statFailure] = await IoFsStatCanonical(
    fileSystem,
    root,
    isStatFS,
  );
  let failure: CanonicalError | undefined;
  if (statFailure !== undefined) {
    failure = await invokeWalkDir(visit, root, undefined, statFailure);
  } else {
    const entry = information === undefined
      ? undefined
      : new CanonicalInfoDirEntry<CanonicalError>(
        information,
        dirEntryContract,
      );
    failure = await walk(
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

async function walk(
  fileSystem: CanonicalFS | undefined,
  path: gostring,
  entry: CanonicalDirEntry | undefined,
  visit: CanonicalWalkDirFunc,
  skipDir: CanonicalError | undefined,
  isReadDirFS: InterfaceGuard<CanonicalReadDirFS>,
  isReadDirFile: InterfaceGuard<CanonicalReadDirFile>,
  errorBridge: FromProviderBridge<GoError, CanonicalError>,
): Promise<CanonicalError | undefined> {
  let failure = await invokeWalkDir(visit, path, entry, undefined);
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
  const [entries, readFailure] = await IoFsReadDirCanonical(
    fileSystem,
    path,
    isReadDirFS,
    isReadDirFile,
    errorBridge,
  );
  if (readFailure !== undefined) {
    failure = await invokeWalkDir(visit, path, entry, readFailure);
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
    const childFailure = await walk(
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

async function invokeWalkDir(
  visit: CanonicalWalkDirFunc,
  path: gostring,
  entry: CanonicalDirEntry | undefined,
  failure: CanonicalError | undefined,
): Promise<CanonicalError | undefined> {
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
  entries: (CanonicalDirEntry | undefined)[],
): Promise<(CanonicalDirEntry | undefined)[]> {
  const named = await Promise.all(entries.map(async (entry) => ({
    entry,
    name: entry === undefined ? "" : await entry.Name(),
  })));
  named.sort((left, right) =>
    left.name < right.name ? -1 : left.name > right.name ? 1 : 0);
  return named.map(({ entry }) => entry);
}
