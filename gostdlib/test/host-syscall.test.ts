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
  assert.equal(EPERM.value, 1);
  assert.equal(EPERM.Error(), "operation not permitted");
  assert.equal(EINTR.value, 4);
  assert.equal(EINTR.Error(), "interrupted system call");
  assert.equal(EAGAIN.Error(), "resource temporarily unavailable");
  assert.equal(EINVAL.Error(), "invalid argument");
  assert.equal(ENOENT.Error(), "no such file or directory");
  assert.equal(SIGINT.String(), "interrupt");
  assert.equal(SIGTERM.String(), "terminated");
  assert.equal(new Signal(99).String(), "signal 99");
  assert.equal(new Signal(10).String(), "user defined signal 1");

  const credential = new Credential(1000, 1001);
  assert.equal(credential.Uid, 1000);
  assert.equal(credential.Gid, 1001);
  assert.equal(credential.Groups.isNil(), true);

  const mapping = new SysProcIDMap(1, 2, 3);
  assert.deepEqual(
    [mapping.ContainerID, mapping.HostID, mapping.Size],
    [1, 2, 3],
  );
  const attributes = new SysProcAttr();
  assert.equal(attributes.Pdeathsig.value, 0);
  assert.equal(attributes.PidFD, undefined);
});

test("syscall selected errno constants agree with Go", (): void => {
  const directory = mkdtempSync(join(tmpdir(), "gotots-syscall-"));
  const sourcePath = join(directory, "main.go");
  const source = `
package main

import (
  "fmt"
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
}
`;
  try {
    writeFileSync(sourcePath, source);
    const result = spawnSync("go", ["run", sourcePath], { encoding: "utf8" });
    assert.equal(result.status, 0, result.stderr);
    const provider = [EPERM, ENOENT, EINTR, EAGAIN, EINVAL, ENOTDIR]
      .map((value): string => `${value.value}:${value.Error()}|`)
      .join("");
    assert.equal(provider, result.stdout);
  } finally {
    rmSync(directory, { force: true, recursive: true });
  }
});
