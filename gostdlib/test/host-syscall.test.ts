import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  mkdtempSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { state as errorState } from "../src/errors.js";
import { state as fsState } from "../src/io/fs.js";
import {
  Credential,
  EAGAIN,
  EINTR,
  EINVAL,
  ENOENT,
  ENOTDIR,
  EPERM,
  Errno,
  SIGINT,
  SIGTERM,
  Signal,
  SysProcAttr,
  SysProcIDMap,
} from "../src/syscall.js";

test("syscall constants and value types preserve selected Linux identities", () => {
  assert.equal(EPERM.value, 1n);
  assert.equal((EPERM.Error())?.text(), "operation not permitted");
  assert.equal(EINTR.value, 4n);
  assert.equal((EINTR.Error())?.text(), "interrupted system call");
  assert.equal((EAGAIN.Error())?.text(), "resource temporarily unavailable");
  assert.equal((EINVAL.Error())?.text(), "invalid argument");
  assert.equal((ENOENT.Error())?.text(), "no such file or directory");
  assert.equal(EPERM.Is(fsState.ErrPermission), true);
  assert.equal(ENOENT.Is(fsState.ErrNotExist), true);
  assert.equal(EINVAL.Is(errorState.ErrUnsupported), false);
  assert.equal(EAGAIN.Timeout(), true);
  assert.equal(EINTR.Temporary(), true);
  assert.equal(EINVAL.Temporary(), false);
  assert.equal((SIGINT.String())?.text(), "interrupt");
  assert.equal((SIGTERM.String())?.text(), "terminated");
  assert.equal((new Signal(99n).String())?.text(), "signal 99");
  assert.equal((new Signal(10n).String())?.text(), "user defined signal 1");

  const credential = new Credential(1000, 1001);
  assert.equal(credential.Uid, 1000);
  assert.equal(credential.Gid, 1001);
  assert.equal(credential.Groups.isNil(), true);

  const mapping = new SysProcIDMap(1n, 2n, 3n);
  assert.deepEqual(
    [mapping.ContainerID, mapping.HostID, mapping.Size],
    [1n, 2n, 3n],
  );
  const attributes = new SysProcAttr();
  assert.equal(attributes.Pdeathsig.value, 0n);
  assert.equal(attributes.PidFD, undefined);
});

test("syscall selected errno constants agree with Go", (): void => {
  const directory = mkdtempSync(join(tmpdir(), "gotots-syscall-"));
  const sourcePath = join(directory, "main.go");
  const source = `
package main

import (
  "errors"
  "fmt"
  "io/fs"
  "syscall"
)

func main() {
  for _, value := range []syscall.Errno{
    syscall.EPERM,
    syscall.ENOENT,
    syscall.EINTR,
    syscall.EAGAIN,
    syscall.EINVAL,
    syscall.ENOTDIR,
  } {
    fmt.Printf("%d:%s|", value, value.Error())
  }
  fmt.Printf("%t:%t:%t:%t:%t:%t",
    syscall.EPERM.Is(fs.ErrPermission),
    syscall.ENOENT.Is(fs.ErrNotExist),
    syscall.EINVAL.Is(errors.ErrUnsupported),
    syscall.EAGAIN.Timeout(),
    syscall.EINTR.Temporary(),
    syscall.EINVAL.Temporary(),
  )
}
`;
  try {
    writeFileSync(sourcePath, source);
    const result = spawnSync("go", ["run", sourcePath], { encoding: "utf8" });
    assert.equal(result.status, 0, result.stderr);
    const provider = [EPERM, ENOENT, EINTR, EAGAIN, EINVAL, ENOTDIR]
      .map((value): string => `${value.value}:${value.Error().text()}|`)
      .join("") + [
        EPERM.Is(fsState.ErrPermission),
        ENOENT.Is(fsState.ErrNotExist),
        EINVAL.Is(errorState.ErrUnsupported),
        EAGAIN.Timeout(),
        EINTR.Temporary(),
        EINVAL.Temporary(),
      ].join(":");
    assert.equal(provider, result.stdout);
  } finally {
    rmSync(directory, { force: true, recursive: true });
  }
});
