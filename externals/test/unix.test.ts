import assert from "node:assert/strict";
import test from "node:test";

import { RawSyscall, Syscall, Syscall6 } from "../src/golang.org/x/sys/unix.js";

test("unsupported Node syscalls return ENOSYS", (): void => {
  for (const result of [
    Syscall(0, 0, 0, 0),
    Syscall6(0, 0, 0, 0, 0, 0, 0),
    RawSyscall(0, 0, 0, 0),
  ]) {
    assert.equal(result[0], 0);
    assert.equal(result[1], 0);
    assert.equal(result[2].value, 38n);
  }
});
