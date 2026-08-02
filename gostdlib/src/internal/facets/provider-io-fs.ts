import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  gostring,
  int64,
  uint8,
} from "@gotots/runtime/scalars.js";

import { New } from "../../errors.js";
import type {
  DirEntry,
  FileMode,
  FileInfo,
  WalkDirFunc,
} from "../../io/fs.js";
import {
  FileInfoToDirEntry,
  PathError,
  state,
} from "../../io/fs.js";
import { state as ioState } from "../../io.js";
import { Join as JoinPath } from "../../path.js";
import type { CanonicalErrorAsync } from "./provider-io-contract.js";
import {
  byteSlice,
  sliceValues,
} from "../runtime/slice.js";
import { goInterfaceEqual } from "../runtime/interface.js";

type InterfaceGuard<Value extends GoInterfaceValue> = (
  value: GoInterfaceValue | undefined,
) => value is Value;

interface FromProviderBridge<
  Provider extends GoInterfaceValue,
  Canonical extends GoInterfaceValue,
> {
  $from(value: Provider | undefined): Canonical | undefined;
}

export interface CanonicalFile extends GoInterfaceValue {
  Close(recovery?: GoRecovery): Promise<GoError | undefined>;
  Read(
    buffer: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, GoError | undefined];
  Stat(recovery?: GoRecovery): [FileInfo | undefined, GoError | undefined];
}

export interface CanonicalFS extends GoInterfaceValue {
  Open(
    name: gostring,
    recovery?: GoRecovery,
  ): [CanonicalFile | undefined, GoError | undefined];
}

export interface CanonicalReadDirFS extends CanonicalFS {
  ReadDir(
    name: gostring,
    recovery?: GoRecovery,
  ): [RuntimeSlice<DirEntry | undefined>, GoError | undefined];
}

export interface CanonicalReadDirFile extends CanonicalFile {
  ReadDir(
    count: int64,
    recovery?: GoRecovery,
  ): [RuntimeSlice<DirEntry | undefined>, GoError | undefined];
}

export interface CanonicalReadFileFS extends CanonicalFS {
  ReadFile(
    name: gostring,
    recovery?: GoRecovery,
  ): [RuntimeSlice<uint8>, GoError | undefined];
}

export interface CanonicalFileAsyncError extends GoInterfaceValue {
  Close(recovery?: GoRecovery): Promise<CanonicalErrorAsync | undefined>;
  Read(
    buffer: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, CanonicalErrorAsync | undefined];
  Stat(
    recovery?: GoRecovery,
  ): [FileInfo | undefined, CanonicalErrorAsync | undefined];
}

export interface CanonicalFSAsyncError extends GoInterfaceValue {
  Open(
    name: gostring,
    recovery?: GoRecovery,
  ): [CanonicalFileAsyncError | undefined, CanonicalErrorAsync | undefined];
}

export interface CanonicalReadFileFSAsyncError extends CanonicalFSAsyncError {
  ReadFile(
    name: gostring,
    recovery?: GoRecovery,
  ): [RuntimeSlice<uint8>, CanonicalErrorAsync | undefined];
}

export interface CanonicalDirEntryAsyncError extends GoInterfaceValue {
  Info(
    recovery?: GoRecovery,
  ): [FileInfo | undefined, CanonicalErrorAsync | undefined];
  IsDir(recovery?: GoRecovery): boolean;
  Name(recovery?: GoRecovery): gostring;
  Type(recovery?: GoRecovery): FileMode;
}

export interface CanonicalReadDirFSAsyncError extends CanonicalFSAsyncError {
  ReadDir(
    name: gostring,
    recovery?: GoRecovery,
  ): [
    RuntimeSlice<CanonicalDirEntryAsyncError | undefined>,
    CanonicalErrorAsync | undefined,
  ];
}

export interface CanonicalReadDirFileAsyncError extends CanonicalFileAsyncError {
  ReadDir(
    count: int64,
    recovery?: GoRecovery,
  ): [
    RuntimeSlice<CanonicalDirEntryAsyncError | undefined>,
    CanonicalErrorAsync | undefined,
  ];
}

type CanonicalWalkDirFuncAsyncError = ((
  path: gostring,
  entry: CanonicalDirEntryAsyncError | undefined,
  failure: CanonicalErrorAsync | undefined,
  recovery?: GoRecovery,
) => Promise<CanonicalErrorAsync | undefined>) | undefined;

export type { CanonicalErrorAsync } from "./provider-io-contract.js";

export interface CanonicalStatFS extends CanonicalFS {
  Stat(
    name: gostring,
    recovery?: GoRecovery,
  ): [FileInfo | undefined, GoError | undefined];
}

export interface CanonicalStatFSAsyncError extends CanonicalFSAsyncError {
  Stat(
    name: gostring,
    recovery?: GoRecovery,
  ): [FileInfo | undefined, CanonicalErrorAsync | undefined];
}

export async function IoFsReadFileCanonical(
  fileSystem: CanonicalFS | undefined,
  name: gostring,
  isReadFileFS: InterfaceGuard<CanonicalReadFileFS>,
): Promise<[RuntimeSlice<uint8>, GoError | undefined]> {
  requireFileSystem(fileSystem);
  if (isReadFileFS(fileSystem)) {
    return fileSystem.ReadFile(name);
  }
  const [file, openFailure] = fileSystem.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<uint8>(), openFailure];
  }
  try {
    const values: number[] = [];
    for (;;) {
      const buffer = RuntimeSlice.make<uint8>(32 * 1024, 32 * 1024, 0);
      const [count, readFailure] = file.Read(buffer);
      for (let index = 0; index < count; index += 1) {
        values.push(buffer.get(index));
      }
      if (readFailure !== undefined) {
        return [
          byteSlice(values),
          goInterfaceEqual(readFailure, ioState.EOF) ? undefined : readFailure,
        ];
      }
    }
  } finally {
    await file.Close();
  }
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
  const [file, openFailure] = fileSystem.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<uint8>(), openFailure];
  }
  try {
    const values: number[] = [];
    for (;;) {
      const buffer = RuntimeSlice.make<uint8>(32 * 1024, 32 * 1024, 0);
      const [count, readFailure] = file.Read(buffer);
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
): Promise<[FileInfo | undefined, GoError | undefined]> {
  requireFileSystem(fileSystem);
  if (isStatFS(fileSystem)) {
    return fileSystem.Stat(name);
  }
  const [file, openFailure] = fileSystem.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [undefined, openFailure];
  }
  try {
    return file.Stat();
  } finally {
    await file.Close();
  }
}

export async function IoFsStatCanonicalAsyncError(
  fileSystem: CanonicalFSAsyncError | undefined,
  name: gostring,
  isStatFS: InterfaceGuard<CanonicalStatFSAsyncError>,
): Promise<[FileInfo | undefined, CanonicalErrorAsync | undefined]> {
  if (fileSystem === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  if (isStatFS(fileSystem)) {
    return fileSystem.Stat(name);
  }
  const [file, openFailure] = fileSystem.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [undefined, openFailure];
  }
  try {
    return file.Stat();
  } finally {
    await file.Close();
  }
}

export async function IoFsReadDirCanonical(
  fileSystem: CanonicalFS | undefined,
  name: gostring,
  isReadDirFS: InterfaceGuard<CanonicalReadDirFS>,
  isReadDirFile: InterfaceGuard<CanonicalReadDirFile>,
): Promise<[RuntimeSlice<DirEntry | undefined>, GoError | undefined]> {
  requireFileSystem(fileSystem);
  if (isReadDirFS(fileSystem)) {
    return fileSystem.ReadDir(name);
  }
  const [file, openFailure] = fileSystem.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<DirEntry | undefined>(), openFailure];
  }
  try {
    if (!isReadDirFile(file)) {
      return [
        RuntimeSlice.nil<DirEntry | undefined>(),
        new PathError("readdir", name, New("not implemented")),
      ];
    }
    const [entries, readFailure] = file.ReadDir(-1);
    const values = sliceValues(entries);
    values.sort((left, right): number => {
      const leftName = left?.Name() ?? "";
      const rightName = right?.Name() ?? "";
      return leftName < rightName ? -1 : leftName > rightName ? 1 : 0;
    });
    return [RuntimeSlice.literal(values), readFailure];
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
  const [file, openFailure] = fileSystem.Open(name);
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
    const [entries, readFailure] = file.ReadDir(-1);
    const values = sliceValues(entries);
    values.sort(compareDirectoryEntries);
    return [RuntimeSlice.literal(values), readFailure];
  } finally {
    await file.Close();
  }
}

export async function IoFsWalkDirCanonical(
  fileSystem: CanonicalFS | undefined,
  root: gostring,
  visit: WalkDirFunc,
  isReadDirFS: InterfaceGuard<CanonicalReadDirFS>,
  isReadDirFile: InterfaceGuard<CanonicalReadDirFile>,
  isStatFS: InterfaceGuard<CanonicalStatFS>,
): Promise<GoError | undefined> {
  const [information, statFailure] = await IoFsStatCanonical(
    fileSystem,
    root,
    isStatFS,
  );
  let failure: GoError | undefined;
  if (statFailure !== undefined) {
    failure = await invokeWalkDir(visit, root, undefined, statFailure);
  } else {
    const entry = FileInfoToDirEntry(information);
    failure = entry === undefined
      ? await invokeWalkDir(visit, root, undefined, statFailure)
      : await walk(
        fileSystem,
        root,
        entry,
        visit,
        isReadDirFS,
        isReadDirFile,
      );
  }
  return goInterfaceEqual(failure, state.SkipDir) ||
    goInterfaceEqual(failure, state.SkipAll)
    ? undefined
    : failure;
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
  dirEntryBridge: FromProviderBridge<DirEntry, CanonicalDirEntryAsyncError>,
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
    const entry = dirEntryBridge.$from(FileInfoToDirEntry(information));
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
  if (failure !== undefined || !entry.IsDir()) {
    if (goInterfaceEqual(failure, skipDir) && entry.IsDir()) {
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
      return goInterfaceEqual(failure, skipDir) && entry.IsDir()
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
      JoinPath(RuntimeSlice.literal([path, child.Name()])),
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

function invokeWalkDirAsyncError(
  visit: CanonicalWalkDirFuncAsyncError,
  path: gostring,
  entry: CanonicalDirEntryAsyncError | undefined,
  failure: CanonicalErrorAsync | undefined,
): Promise<CanonicalErrorAsync | undefined> {
  if (visit === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return visit(path, entry, failure);
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

function compareDirectoryEntries(
  left: CanonicalDirEntryAsyncError | undefined,
  right: CanonicalDirEntryAsyncError | undefined,
): number {
  const leftName = left?.Name() ?? "";
  const rightName = right?.Name() ?? "";
  return leftName < rightName ? -1 : leftName > rightName ? 1 : 0;
}

async function walk(
  fileSystem: CanonicalFS | undefined,
  path: gostring,
  entry: DirEntry,
  visit: WalkDirFunc,
  isReadDirFS: InterfaceGuard<CanonicalReadDirFS>,
  isReadDirFile: InterfaceGuard<CanonicalReadDirFile>,
): Promise<GoError | undefined> {
  let failure = await invokeWalkDir(visit, path, entry, undefined);
  if (failure !== undefined || !entry.IsDir()) {
    if (goInterfaceEqual(failure, state.SkipDir) && entry.IsDir()) {
      failure = undefined;
    }
    return failure;
  }
  const [entries, readFailure] = await IoFsReadDirCanonical(
    fileSystem,
    path,
    isReadDirFS,
    isReadDirFile,
  );
  if (readFailure !== undefined) {
    failure = await invokeWalkDir(visit, path, entry, readFailure);
    if (failure !== undefined) {
      return goInterfaceEqual(failure, state.SkipDir) && entry.IsDir()
        ? undefined
        : failure;
    }
  }
  for (const child of sliceValues(entries)) {
    if (child === undefined) {
      continue;
    }
    const childPath = path === "." ? child.Name() : `${path}/${child.Name()}`;
    const childFailure = await walk(
      fileSystem,
      childPath,
      child,
      visit,
      isReadDirFS,
      isReadDirFile,
    );
    if (goInterfaceEqual(childFailure, state.SkipDir)) {
      break;
    }
    if (childFailure !== undefined) {
      return childFailure;
    }
  }
  return undefined;
}

function invokeWalkDir(
  visit: WalkDirFunc,
  path: gostring,
  entry: DirEntry | undefined,
  failure: GoError | undefined,
): Promise<GoError | undefined> {
  if (visit === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return visit(path, entry, failure);
}

function requireFileSystem(
  fileSystem: CanonicalFS | undefined,
): asserts fileSystem is CanonicalFS {
  if (fileSystem === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
}
