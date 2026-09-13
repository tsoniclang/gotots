import { GoString } from "@gotots/runtime/string-value.js";
import assert from "node:assert/strict";
import test from "node:test";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import {
  ContinueOnError,
  FlagSet,
  NewFlagSet,
} from "../src/flag.js";
import { FlagSetValueOperations } from "../src/internal/facets/named-flag.js";

test("FlagSet parses selected boolean and string flags through Go pointers", () => {
  const flags = NewFlagSet(GoString.fromText("provider"), ContinueOnError);
  assert.ok(flags !== undefined);
  const verbose = FlagSet.Bool(flags, GoString.fromText("verbose"), false, GoString.fromText("enable output"));
  const output = FlagSet.String(flags, GoString.fromText("output"), GoString.fromText("default"), GoString.fromText("output path"));
  assert.ok(verbose !== undefined);
  assert.ok(output !== undefined);

  const error = FlagSet.Parse(
    flags,
    RuntimeSlice.literal([GoString.fromText("-verbose=T"), GoString.fromText("--output=result.txt")]),
  );
  assert.equal(error, undefined);
  assert.equal(verbose.value, true);
  assert.equal(output.value.text(), "result.txt");
});

test("FlagSet rejects undefined and invalid flag values", () => {
  const flags = NewFlagSet(GoString.fromText("provider"), ContinueOnError);
  assert.ok(flags !== undefined);
  FlagSet.Bool(flags, GoString.fromText("verbose"), false, GoString.fromText("enable output"));
  assert.equal(
    (FlagSet.Parse(flags, RuntimeSlice.literal([GoString.fromText("-missing")]))?.Error())?.text(),
    "provider: flag provided but not defined: -missing",
  );
  assert.equal(
    (FlagSet.Parse(flags, RuntimeSlice.literal([GoString.fromText("-verbose=maybe")]))?.Error())?.text(),
    "provider: invalid value maybe for flag -verbose",
  );
});

test("FlagSet duplicate definitions preserve Go panic behavior", () => {
  const flags = NewFlagSet(GoString.fromText("provider"), ContinueOnError);
  assert.ok(flags !== undefined);
  FlagSet.String(flags, GoString.fromText("output"), GoString.fromText(""), GoString.fromText("output path"));
  assert.throws(() => FlagSet.String(flags, GoString.fromText("output"), GoString.fromText(""), GoString.fromText("again")));
});

test("FlagSet value operations preserve shallow Go struct assignment", () => {
  const source = NewFlagSet(GoString.fromText("source"), ContinueOnError);
  const target = NewFlagSet(GoString.fromText("target"), ContinueOnError);
  assert.ok(source !== undefined);
  assert.ok(target !== undefined);
  const output = FlagSet.String(source, GoString.fromText("output"), GoString.fromText("default"), GoString.fromText("output path"));
  assert.ok(output !== undefined);
  const usage = (): void => {};
  source.Usage = usage;

  FlagSetValueOperations.$assign(target, source);
  assert.equal(target.Usage, usage);
  assert.equal(
    FlagSet.Parse(target, RuntimeSlice.literal([GoString.fromText("--output=assigned")])),
    undefined,
  );
  assert.equal(output.value.text(), "assigned");

  const copied = FlagSetValueOperations.$copy(source);
  assert.notEqual(copied, source);
  assert.equal(copied.Usage, usage);
  assert.equal(
    FlagSet.Parse(copied, RuntimeSlice.literal([GoString.fromText("--output=copied")])),
    undefined,
  );
  assert.equal(output.value.text(), "copied");
});
