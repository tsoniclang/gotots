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
  FileInfo,
  WalkDirFunc,
} from "../../io/fs.js";
import {
  FileInfoToDirEntry,
  PathError,
  state,
} from "../../io/fs.js";
import { state as ioState } from "../../io.js";
import {
  byteSlice,
  sliceValues,
} from "../runtime/slice.js";

type InterfaceGuard<Value extends GoInterfaceValue> = (
  value: GoInterfaceValue | undefined,
) => value is Value;

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

export interface CanonicalStatFS extends CanonicalFS {
  Stat(
    name: gostring,
    recovery?: GoRecovery,
  ): [FileInfo | undefined, GoError | undefined];
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
          readFailure === ioState.EOF ? undefined : readFailure,
        ];
      }
      if (count === 0) {
        return [
          byteSlice(values),
          New("multiple Read calls return no data or error"),
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
  return failure === state.SkipDir || failure === state.SkipAll
    ? undefined
    : failure;
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
    if (failure === state.SkipDir && entry.IsDir()) {
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
      return failure === state.SkipDir && entry.IsDir()
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
    if (childFailure === state.SkipDir) {
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
