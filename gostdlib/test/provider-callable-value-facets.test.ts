import assert from "node:assert/strict";
import test from "node:test";

import type { CancelCauseFunc } from "../src/context.js";
import type { WalkDirFunc } from "../src/io/fs.js";
import { RegexpReplaceAllStringFuncCanonical } from "../src/internal/facets/provider-regexp.js";
import {
  IoFsFileInfoToDirEntryCanonical,
  type CanonicalFileInfo,
} from "../src/internal/facets/provider-io-fs.js";
import {
  SortCanonical,
  type SortInterfaceCanonical,
  SortSearchCanonical,
} from "../src/internal/facets/provider-sort.js";
import {
  StringsContainsFuncCanonical,
  StringsIndexFuncCanonical,
  StringsLastIndexFuncCanonical,
  StringsMapCanonical,
  StringsTrimFuncCanonical,
  StringsTrimLeftFuncCanonical,
  StringsTrimRightFuncCanonical,
} from "../src/internal/facets/provider-strings.js";
import { SyscallErrnoIsCanonical } from "../src/internal/facets/provider-syscall.js";
import { permission } from "../src/internal/portable/errors/sentinel.js";
import { MustCompile } from "../src/regexp.js";
import { EPERM } from "../src/syscall.js";
import { ProviderInterfaceValue } from "../src/internal/portable/io/value.js";

interface CanonicalFailure {
  Error(): string | Promise<string>;
}

interface CanonicalEntry {
  Name(): string | Promise<string>;
}

test("provider named callables keep source arity", async () => {
  const cancel: NonNullable<CancelCauseFunc> = async () => {};
  const walk: NonNullable<WalkDirFunc> = async (
    _path,
    _entry,
    failure,
  ) => failure;

  await cancel(undefined);
  assert.equal(await walk(".", undefined, undefined), undefined);
});

class CooperativeSortable extends ProviderInterfaceValue implements SortInterfaceCanonical {
  constructor(readonly values: number[]) {
    super(Object.freeze({ comparable: true }));
  }

  async Len(): Promise<number> {
    return this.values.length;
  }

  async Less(left: number, right: number): Promise<boolean> {
    return (this.values[left] ?? 0) < (this.values[right] ?? 0);
  }

  async Swap(left: number, right: number): Promise<void> {
    const saved = this.values[left];
    const replacement = this.values[right];
    if (saved === undefined || replacement === undefined) {
      throw new Error("sort index is outside the test value");
    }
    this.values[left] = replacement;
    this.values[right] = saved;
  }
}

class CooperativeFileInfo extends ProviderInterfaceValue implements CanonicalFileInfo {
  constructor() {
    super(Object.freeze({ comparable: true }));
  }

  async IsDir(): Promise<boolean> {
    return false;
  }

  async ModTime(): Promise<never> {
    throw new Error("unused test method");
  }

  async Mode(): Promise<never> {
    throw new Error("unused test method");
  }

  async Name(): Promise<string> {
    return "entry";
  }

  async Size(): Promise<number> {
    return 3;
  }

  async Sys(): Promise<undefined> {
    return undefined;
  }
}

test("provider callable profiles transport cooperative callbacks", async () => {
  assert.equal(
    await SortSearchCanonical(8, async (index): Promise<boolean> => index >= 5),
    5,
  );
  const sortable = new CooperativeSortable([3, 1, 2]);
  await SortCanonical(sortable);
  assert.deepEqual(sortable.values, [1, 2, 3]);

  const isLetterA = async (rune: number): Promise<boolean> => rune === 97;
  assert.equal(await StringsContainsFuncCanonical("ba", isLetterA), true);
  assert.equal(await StringsIndexFuncCanonical("ba", isLetterA), 1);
  assert.equal(await StringsLastIndexFuncCanonical("aba", isLetterA), 2);
  assert.equal(
    await StringsMapCanonical(async (rune) => rune === 97 ? 65 : rune, "ab"),
    "Ab",
  );
  assert.equal(await StringsTrimFuncCanonical("aabaa", isLetterA), "b");
  assert.equal(await StringsTrimLeftFuncCanonical("aab", isLetterA), "b");
  assert.equal(await StringsTrimRightFuncCanonical("baa", isLetterA), "b");
  assert.equal(
    await RegexpReplaceAllStringFuncCanonical(
      MustCompile("[0-9]+"),
      "a1b22",
      async (match) => `[${match}]`,
    ),
    "a[1]b[22]",
  );
  const entry = IoFsFileInfoToDirEntryCanonical(new CooperativeFileInfo(), []);
  assert.equal(await entry?.Name(), "entry");
  assert.equal(SyscallErrnoIsCanonical(EPERM, permission), true);
});
