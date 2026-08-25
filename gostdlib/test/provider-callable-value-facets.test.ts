import assert from "node:assert/strict";
import test from "node:test";
import type { int64 } from "../src/internal/scalars.js";
import {
  hostInteger,
  integerFromHost,
} from "../src/internal/host-integer.js";

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

test("provider named callables keep source arity", () => {
  const cancel: NonNullable<CancelCauseFunc> = () => {};
  const walk: NonNullable<WalkDirFunc> = (
    _path,
    _entry,
    failure,
  ) => failure;

  cancel(undefined);
  assert.equal(walk(".", undefined, undefined), undefined);
});

class CooperativeSortable extends ProviderInterfaceValue implements SortInterfaceCanonical {
  constructor(readonly values: number[]) {
    super(Object.freeze({ comparable: true }));
  }

  async Len(): Promise<int64> {
    return integerFromHost(this.values.length);
  }

  async Less(left: int64, right: int64): Promise<boolean> {
    return (this.values[hostInteger(left)] ?? 0)
      < (this.values[hostInteger(right)] ?? 0);
  }

  async Swap(left: int64, right: int64): Promise<void> {
    const leftIndex = hostInteger(left);
    const rightIndex = hostInteger(right);
    const saved = this.values[leftIndex];
    const replacement = this.values[rightIndex];
    if (saved === undefined || replacement === undefined) {
      throw new Error("sort index is outside the test value");
    }
    this.values[leftIndex] = replacement;
    this.values[rightIndex] = saved;
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

  async Size(): Promise<int64> {
    return 3n;
  }

  async Sys(): Promise<undefined> {
    return undefined;
  }
}

test("provider callable profiles transport cooperative callbacks", async () => {
  assert.equal(
    await SortSearchCanonical(8n, async (index): Promise<boolean> => index >= 5n),
    5n,
  );
  const sortable = new CooperativeSortable([3, 1, 2]);
  await SortCanonical(sortable);
  assert.deepEqual(sortable.values, [1, 2, 3]);

  const isLetterA = async (rune: number): Promise<boolean> => rune === 97;
  assert.equal(await StringsContainsFuncCanonical("ba", isLetterA), true);
  assert.equal(await StringsIndexFuncCanonical("ba", isLetterA), 1n);
  assert.equal(await StringsLastIndexFuncCanonical("aba", isLetterA), 2n);
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
