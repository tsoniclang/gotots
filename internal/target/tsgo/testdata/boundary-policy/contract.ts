export interface InterfaceValue {
  $equal(other: InterfaceValue): boolean;
}

export type Awaitable<Value> = Value | Promise<Value>;

export interface ProviderError extends InterfaceValue {
  Error(): string;
}

export interface ProviderDirEntry extends InterfaceValue {
  Info(): [BoundaryInfo | undefined, ProviderError | undefined];
  IsDir(): boolean;
  Name(): string;
}

export interface ReadDirFSIdentity extends InterfaceValue {}
export interface ReadDirFileIdentity extends InterfaceValue {}
export interface StatFSIdentity extends InterfaceValue {}

export interface BoundaryFailure extends InterfaceValue {
  Error(): Awaitable<string>;
}

export interface BoundaryInfo {
  IsDir(): Awaitable<boolean>;
  Name(): Awaitable<string>;
}

export interface BoundaryDirEntry<Failure extends BoundaryFailure>
  extends InterfaceValue {
  Info(): Awaitable<[BoundaryInfo | undefined, Failure | undefined]>;
  IsDir(): Awaitable<boolean>;
  Name(): Awaitable<string>;
}

export interface BoundaryFile<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
> extends InterfaceValue {
  Close(): Awaitable<Failure | undefined>;
  Stat(): Awaitable<[BoundaryInfo | undefined, Failure | undefined]>;
}

export interface BoundaryReadDirFile<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
> extends BoundaryFile<Entry, Failure> {
  ReadDir(): Awaitable<[Entry[], Failure | undefined]>;
}

export interface BoundaryFS<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
> extends InterfaceValue {
  Open(path: string): Awaitable<[
    BoundaryFile<Entry, Failure> | undefined,
    Failure | undefined,
  ]>;
}

export interface BoundaryReadDirFS<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
> extends BoundaryFS<Entry, Failure> {
  ReadDir(path: string): Awaitable<[Entry[], Failure | undefined]>;
}

export interface BoundaryStatFS<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
> extends BoundaryFS<Entry, Failure> {
  Stat(path: string): Awaitable<[
    BoundaryInfo | undefined,
    Failure | undefined,
  ]>;
}

export type BoundaryVisit<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
> = (
  path: string,
  entry: Entry | undefined,
  failure: Failure | undefined,
) => Awaitable<Failure | undefined>;

export function SourceWalkDir(
  _fileSystem: InterfaceValue | undefined,
  _root: string,
  _visit: (...values: InterfaceValue[]) => ProviderError | undefined,
): ProviderError | undefined {
  return undefined;
}

export interface CanonicalBoundaryPolicy<Source> {
  readonly $go$canonicalBoundarySource?: Source;
}

export interface FromProviderRequest<
  Source extends InterfaceValue,
  Target extends InterfaceValue,
> {
  readonly $go$fromProviderSource?: Source;
  $from(value: Source | undefined): Target | undefined;
}

export interface InterfaceGuardRequest<
  Source extends InterfaceValue,
  Target extends InterfaceValue,
> {
  readonly $go$guardSource?: Source;
  (value: InterfaceValue | undefined): value is Target;
}

export interface WalkPolicy<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
> extends CanonicalBoundaryPolicy<typeof SourceWalkDir> {
  readonly entryBridge: FromProviderRequest<ProviderDirEntry, Entry>;
  readonly errorBridge: FromProviderRequest<ProviderError, Failure>;
  readonly isReadDirFS: InterfaceGuardRequest<
    ReadDirFSIdentity,
    BoundaryReadDirFS<Entry, Failure>
  >;
  readonly isReadDirFile: InterfaceGuardRequest<
    ReadDirFileIdentity,
    BoundaryReadDirFile<Entry, Failure>
  >;
  readonly isStatFS: InterfaceGuardRequest<
    StatFSIdentity,
    BoundaryStatFS<Entry, Failure>
  >;
}

class ProviderErrorValue implements ProviderError {
  constructor(readonly message: string) {}

  $equal(other: InterfaceValue): boolean {
    return this === other;
  }

  Error(): string {
    return this.message;
  }
}

class ProviderEntryValue implements ProviderDirEntry {
  constructor(
    readonly name: string,
    readonly directory: boolean,
  ) {}

  $equal(other: InterfaceValue): boolean {
    return this === other;
  }

  Info(): [BoundaryInfo, undefined] {
    return [new InfoValue(this.name, this.directory), undefined];
  }

  IsDir(): boolean {
    return this.directory;
  }

  Name(): string {
    return this.name;
  }
}

class InfoValue implements BoundaryInfo {
  constructor(
    readonly name: string,
    readonly directory: boolean,
    readonly asynchronous = false,
  ) {}

  IsDir(): Awaitable<boolean> {
    return this.asynchronous ? Promise.resolve(this.directory) : this.directory;
  }

  Name(): Awaitable<string> {
    return this.asynchronous ? Promise.resolve(this.name) : this.name;
  }
}

class CanonicalFailure implements BoundaryFailure {
  constructor(
    readonly provider: ProviderError,
    readonly asynchronous = false,
  ) {}

  $equal(other: InterfaceValue): boolean {
    return other instanceof CanonicalFailure
      ? this.provider.$equal(other.provider)
      : this.provider.$equal(other);
  }

  Error(): Awaitable<string> {
    return this.asynchronous
      ? Promise.resolve(this.provider.Error())
      : this.provider.Error();
  }
}

class CanonicalEntry implements BoundaryDirEntry<CanonicalFailure> {
  constructor(
    readonly provider: ProviderDirEntry,
    readonly asynchronous = false,
  ) {}

  $equal(other: InterfaceValue): boolean {
    return other instanceof CanonicalEntry
      ? this.provider.$equal(other.provider)
      : this.provider.$equal(other);
  }

  Info(): Awaitable<[BoundaryInfo | undefined, CanonicalFailure | undefined]> {
    const [information, failure] = this.provider.Info();
    const result: [BoundaryInfo | undefined, CanonicalFailure | undefined] = [
      information,
      failure === undefined ? undefined : new CanonicalFailure(failure),
    ];
    return this.asynchronous ? Promise.resolve(result) : result;
  }

  IsDir(): Awaitable<boolean> {
    return this.asynchronous
      ? Promise.resolve(this.provider.IsDir())
      : this.provider.IsDir();
  }

  Name(): Awaitable<string> {
    return this.asynchronous
      ? Promise.resolve(this.provider.Name())
      : this.provider.Name();
  }
}

const providerSkipDir = new ProviderErrorValue("skip-dir");
const providerSkipAll = new ProviderErrorValue("skip-all");

export const errorBridge: FromProviderRequest<
  ProviderError,
  CanonicalFailure
> = {
  $from(value): CanonicalFailure | undefined {
    return value === undefined ? undefined : new CanonicalFailure(value, true);
  },
};

export const entryBridge: FromProviderRequest<
  ProviderDirEntry,
  CanonicalEntry
> = {
  $from(value): CanonicalEntry | undefined {
    return value === undefined ? undefined : new CanonicalEntry(value, true);
  },
};

class SyncFS implements BoundaryReadDirFS<CanonicalEntry, CanonicalFailure>,
  BoundaryStatFS<CanonicalEntry, CanonicalFailure> {
  $equal(other: InterfaceValue): boolean {
    return this === other;
  }

  Open(): [undefined, CanonicalFailure] {
    return [undefined, new CanonicalFailure(new ProviderErrorValue("unused"))];
  }

  Stat(path: string): [BoundaryInfo, undefined] {
    return [new InfoValue(path, true), undefined];
  }

  ReadDir(path: string): [CanonicalEntry[], undefined] {
    if (path !== ".") {
      return [[], undefined];
    }
    return [[
      requireValue(entryBridge.$from(new ProviderEntryValue("a", false))),
      requireValue(entryBridge.$from(new ProviderEntryValue("skip", true))),
      requireValue(entryBridge.$from(new ProviderEntryValue("z", false))),
    ], undefined];
  }
}

class AsyncFile implements BoundaryReadDirFile<CanonicalEntry, CanonicalFailure> {
  constructor(readonly path: string) {}

  $equal(other: InterfaceValue): boolean {
    return this === other;
  }

  async Close(): Promise<undefined> {
    return undefined;
  }

  async Stat(): Promise<[BoundaryInfo, undefined]> {
    return [new InfoValue(this.path, true, true), undefined];
  }

  async ReadDir(): Promise<[CanonicalEntry[], undefined]> {
    return this.path === "."
      ? [[requireValue(entryBridge.$from(new ProviderEntryValue("b", false)))], undefined]
      : [[], undefined];
  }
}

class AsyncFS implements BoundaryFS<CanonicalEntry, CanonicalFailure> {
  $equal(other: InterfaceValue): boolean {
    return this === other;
  }

  async Open(path: string): Promise<[AsyncFile, undefined]> {
    return [new AsyncFile(path), undefined];
  }
}

export const isReadDirFS: InterfaceGuardRequest<
  ReadDirFSIdentity,
  BoundaryReadDirFS<CanonicalEntry, CanonicalFailure>
> = (value): value is SyncFS => value instanceof SyncFS;

export const isReadDirFile: InterfaceGuardRequest<
  ReadDirFileIdentity,
  BoundaryReadDirFile<CanonicalEntry, CanonicalFailure>
> = (value): value is AsyncFile => value instanceof AsyncFile;

export const isStatFS: InterfaceGuardRequest<
  StatFSIdentity,
  BoundaryStatFS<CanonicalEntry, CanonicalFailure>
> = (value): value is SyncFS => value instanceof SyncFS;

export async function WalkDir<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
>(
  fileSystem: BoundaryFS<Entry, Failure> | undefined,
  root: string,
  visit: BoundaryVisit<Entry, Failure>,
  policy: WalkPolicy<Entry, Failure>,
): Promise<Failure | undefined> {
  const skipDir = requireValue(policy.errorBridge.$from(providerSkipDir));
  const skipAll = requireValue(policy.errorBridge.$from(providerSkipAll));
  const [information, statFailure] = await stat(fileSystem, root, policy);
  const rootEntry = information === undefined
    ? undefined
    : policy.entryBridge.$from(new ProviderEntryValue(
      await information.Name(),
      await information.IsDir(),
    ));
  let failure = statFailure !== undefined || rootEntry === undefined
    ? await visit(root, undefined, statFailure)
    : await walk(fileSystem, root, rootEntry, visit, policy, skipDir);
  if (sameFailure(failure, skipDir) || sameFailure(failure, skipAll)) {
    failure = undefined;
  }
  return failure;
}

async function stat<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
>(
  fileSystem: BoundaryFS<Entry, Failure> | undefined,
  path: string,
  policy: WalkPolicy<Entry, Failure>,
): Promise<[BoundaryInfo | undefined, Failure | undefined]> {
  if (policy.isStatFS(fileSystem)) {
    return await fileSystem.Stat(path);
  }
  if (fileSystem === undefined) {
    return [undefined, requireValue(policy.errorBridge.$from(
      new ProviderErrorValue("nil filesystem"),
    ))];
  }
  const [file, openFailure] = await fileSystem.Open(path);
  if (file === undefined || openFailure !== undefined) {
    return [undefined, openFailure];
  }
  const result = await file.Stat();
  await file.Close();
  return result;
}

async function readDir<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
>(
  fileSystem: BoundaryFS<Entry, Failure> | undefined,
  path: string,
  policy: WalkPolicy<Entry, Failure>,
): Promise<[Entry[], Failure | undefined]> {
  if (policy.isReadDirFS(fileSystem)) {
    return await fileSystem.ReadDir(path);
  }
  if (fileSystem === undefined) {
    return [[], requireValue(policy.errorBridge.$from(
      new ProviderErrorValue("nil filesystem"),
    ))];
  }
  const [file, openFailure] = await fileSystem.Open(path);
  if (file === undefined || openFailure !== undefined) {
    return [[], openFailure];
  }
  if (!policy.isReadDirFile(file)) {
    await file.Close();
    return [[], requireValue(policy.errorBridge.$from(
      new ProviderErrorValue("missing ReadDir"),
    ))];
  }
  const result = await file.ReadDir();
  await file.Close();
  return result;
}

async function walk<
  Entry extends BoundaryDirEntry<Failure>,
  Failure extends BoundaryFailure,
>(
  fileSystem: BoundaryFS<Entry, Failure> | undefined,
  path: string,
  entry: Entry,
  visit: BoundaryVisit<Entry, Failure>,
  policy: WalkPolicy<Entry, Failure>,
  skipDir: Failure,
): Promise<Failure | undefined> {
  let failure = await visit(path, entry, undefined);
  if (failure !== undefined || !await entry.IsDir()) {
    return sameFailure(failure, skipDir) ? undefined : failure;
  }
  const [entries, readFailure] = await readDir(fileSystem, path, policy);
  if (readFailure !== undefined) {
    failure = await visit(path, entry, readFailure);
    if (failure !== undefined) {
      return sameFailure(failure, skipDir) ? undefined : failure;
    }
  }
  for (const child of entries) {
    const name = await child.Name();
    const childPath = path === "." ? name : `${path}/${name}`;
    const childFailure = await walk(
      fileSystem,
      childPath,
      child,
      visit,
      policy,
      skipDir,
    );
    if (sameFailure(childFailure, skipDir)) {
      break;
    }
    if (childFailure !== undefined) {
      return childFailure;
    }
  }
  return undefined;
}

function sameFailure<Failure extends BoundaryFailure>(
  left: Failure | undefined,
  right: Failure,
): boolean {
  return left !== undefined && left.$equal(right);
}

function requireValue<Value extends InterfaceValue>(
  value: Value | undefined,
): Value {
  if (value === undefined) {
    throw new Error("bridge discarded a non-nil provider value");
  }
  return value;
}

export async function runProof(
  policy: WalkPolicy<CanonicalEntry, CanonicalFailure>,
): Promise<string> {
  const syncPaths: string[] = [];
  const syncFailure = await WalkDir(
    new SyncFS(),
    ".",
    async (path): Promise<CanonicalFailure | undefined> => {
      syncPaths.push(path);
      return path === "skip"
        ? policy.errorBridge.$from(providerSkipDir)
        : undefined;
    },
    policy,
  );
  const asyncPaths: string[] = [];
  const asyncFailure = await WalkDir(
    new AsyncFS(),
    ".",
    async (path): Promise<CanonicalFailure | undefined> => {
      await Promise.resolve();
      asyncPaths.push(path);
      return path === "b"
        ? policy.errorBridge.$from(new ProviderErrorValue("blocked"))
        : undefined;
    },
    policy,
  );
  return [
    `sync:${syncPaths.join(",")}:${syncFailure === undefined ? "ok" : "failed"}`,
    `async:${asyncPaths.join(",")}:${asyncFailure === undefined ? "ok" : await asyncFailure.Error()}`,
  ].join("\n");
}
