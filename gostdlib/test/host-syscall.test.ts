import assert from "node:assert/strict";
import test from "node:test";
import {
  Credential,
  EINTR,
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
  assert.equal(new Errno(2).Error(), "no such file or directory");
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
