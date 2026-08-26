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
  IoFsFileInfoToDirEntryDirect,
  type ProviderFileInfo,
} from "../src/internal/facets/provider-io-fs-direct.js";
import {
  SortDirect,
  type SortInterfaceDirect,
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

class Sortable extends ProviderInterfaceValue implements SortInterfaceDirect {
  constructor(readonly values: number[]) {
    super(Object.freeze({ comparable: true }));
  }

  Len(): int64 {
    return integerFromHost(this.values.length);
  }

  Less(left: int64, right: int64): boolean {
    return (this.values[hostInteger(left)] ?? 0)
      < (this.values[hostInteger(right)] ?? 0);
  }

  Swap(left: int64, right: int64): void {
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

class FileInfoValue extends ProviderInterfaceValue implements ProviderFileInfo {
  constructor() {
    super(Object.freeze({ comparable: true }));
  }

  IsDir(): boolean {
    return false;
  }

  ModTime(): never {
    throw new Error("unused test method");
  }

  Mode(): never {
    throw new Error("unused test method");
  }

  Name(): string {
    return "entry";
  }

  Size(): int64 {
    return 3n;
  }

  Sys(): undefined {
    return undefined;
  }
}

test("provider callable profiles transport callbacks", () => {
  assert.equal(
    SortSearchCanonical(8n, (index): boolean => index >= 5n),
    5n,
  );
  const sortable = new Sortable([3, 1, 2]);
  SortDirect(sortable);
  assert.deepEqual(sortable.values, [1, 2, 3]);

  const isLetterA = (rune: number): boolean => rune === 97;
  assert.equal(StringsContainsFuncCanonical("ba", isLetterA), true);
  assert.equal(StringsIndexFuncCanonical("ba", isLetterA), 1n);
  assert.equal(StringsLastIndexFuncCanonical("aba", isLetterA), 2n);
  assert.equal(
    StringsMapCanonical((rune) => rune === 97 ? 65 : rune, "ab"),
    "Ab",
  );
  assert.equal(StringsTrimFuncCanonical("aabaa", isLetterA), "b");
  assert.equal(StringsTrimLeftFuncCanonical("aab", isLetterA), "b");
  assert.equal(StringsTrimRightFuncCanonical("baa", isLetterA), "b");
  assert.equal(
    RegexpReplaceAllStringFuncCanonical(
      MustCompile("[0-9]+"),
      "a1b22",
      (match) => `[${match}]`,
    ),
    "a[1]b[22]",
  );
  const entry = IoFsFileInfoToDirEntryDirect(new FileInfoValue(), []);
  assert.equal(entry?.Name(), "entry");
  assert.equal(SyscallErrnoIsCanonical(EPERM, permission), true);
});
