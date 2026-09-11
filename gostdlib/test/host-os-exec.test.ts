import { GoString } from "@gotots/runtime/string-value.js";
import { fromHostString } from "../src/internal/portable/utf8/codec.js";
import assert from "node:assert/strict";
import test from "node:test";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import {
  Cmd,
  Command,
} from "../src/os/exec.js";
import { OsExecCmdOperations } from "../src/internal/facets/named-os-exec.js";
import { sliceValues } from "../src/internal/runtime/slice.js";
import { Duration } from "../src/time.js";

test("Cmd.Output captures stdout and preserves exit errors", () => {
  const success = Command(
    fromHostString(process.execPath),
    RuntimeSlice.literal([
      GoString.fromText("-e"),
      GoString.fromText("process.stdout.write('provider-output')"),
    ]),
  );
  assert.ok(success !== undefined);
  const [output, outputError] = Cmd.Output(success);
  assert.equal(outputError, undefined);
  assert.equal(Buffer.from(sliceValues(output)).toString("utf8"), "provider-output");
  assert.ok(success.Process !== undefined);
  assert.ok(success.ProcessState !== undefined);
  assert.ok(success.Stdout !== undefined);
  assert.equal((Cmd.Output(success)[1]?.Error())?.text(), "exec: Stdout already set");

  const failure = Command(
    fromHostString(process.execPath),
    RuntimeSlice.literal([GoString.fromText("-e"), GoString.fromText("process.exit(7)")]),
  );
  assert.ok(failure !== undefined);
  const [failedOutput, failureError] = Cmd.Output(failure);
  assert.equal(failedOutput.length, 0);
  assert.equal((failureError?.Error())?.text(), "exit status 7");
});

test("Cmd.Output supplies selected environment entries", () => {
  const command = Command(
    fromHostString(process.execPath),
    RuntimeSlice.literal([
      GoString.fromText("-e"),
      GoString.fromText("process.stdout.write(process.env.GOTOTS_CHILD ?? '')"),
    ]),
  );
  assert.ok(command !== undefined);
  command.Env = RuntimeSlice.literal([GoString.fromText("GOTOTS_CHILD=present")]);
  const [output, error] = Cmd.Output(command);
  assert.equal(error, undefined);
  assert.equal(Buffer.from(sliceValues(output)).toString("utf8"), "present");
});

test("Cmd value operations preserve shallow Go struct assignment", () => {
  const source = Command(GoString.fromText("node"), RuntimeSlice.literal([GoString.fromText("source")]));
  const target = Command(GoString.fromText("node"), RuntimeSlice.literal([GoString.fromText("target")]));
  assert.ok(source !== undefined);
  assert.ok(target !== undefined);
  source.Env = RuntimeSlice.literal([GoString.fromText("VALUE=source")]);
  source.Dir = GoString.fromText("/source");
  source.WaitDelay = new Duration(7n);

  OsExecCmdOperations.$assign(target, source);
  assert.equal(target.Path, source.Path);
  assert.equal(target.Args, source.Args);
  assert.equal(target.Env, source.Env);
  assert.equal(target.Dir, source.Dir);
  assert.equal(target.WaitDelay, source.WaitDelay);

  const copied = OsExecCmdOperations.$copy(source);
  assert.notEqual(copied, source);
  assert.equal(copied.Args, source.Args);
  assert.equal(copied.Env, source.Env);
  assert.equal(copied.Dir, source.Dir);
  assert.equal(copied.WaitDelay, source.WaitDelay);
});
