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
  const flags = NewFlagSet("provider", ContinueOnError);
  assert.ok(flags !== undefined);
  const verbose = FlagSet.Bool(flags, "verbose", false, "enable output");
  const output = FlagSet.String(flags, "output", "default", "output path");
  assert.ok(verbose !== undefined);
  assert.ok(output !== undefined);

  const error = FlagSet.Parse(
    flags,
    RuntimeSlice.literal(["-verbose=T", "--output=result.txt"]),
  );
  assert.equal(error, undefined);
  assert.equal(verbose.value, true);
  assert.equal(output.value, "result.txt");
});

test("FlagSet rejects undefined and invalid flag values", () => {
  const flags = NewFlagSet("provider", ContinueOnError);
  assert.ok(flags !== undefined);
  FlagSet.Bool(flags, "verbose", false, "enable output");
  assert.equal(
    FlagSet.Parse(flags, RuntimeSlice.literal(["-missing"]))?.Error(),
    "provider: flag provided but not defined: -missing",
  );
  assert.equal(
    FlagSet.Parse(flags, RuntimeSlice.literal(["-verbose=maybe"]))?.Error(),
    "provider: invalid value maybe for flag -verbose",
  );
});

test("FlagSet duplicate definitions preserve Go panic behavior", () => {
  const flags = NewFlagSet("provider", ContinueOnError);
  assert.ok(flags !== undefined);
  FlagSet.String(flags, "output", "", "output path");
  assert.throws(() => FlagSet.String(flags, "output", "", "again"));
});

test("FlagSet value operations preserve shallow Go struct assignment", () => {
  const source = NewFlagSet("source", ContinueOnError);
  const target = NewFlagSet("target", ContinueOnError);
  assert.ok(source !== undefined);
  assert.ok(target !== undefined);
  const output = FlagSet.String(source, "output", "default", "output path");
  assert.ok(output !== undefined);
  const usage = (): void => {};
  source.Usage = usage;

  FlagSetValueOperations.$assign(target, source);
  assert.equal(target.Usage, usage);
  assert.equal(
    FlagSet.Parse(target, RuntimeSlice.literal(["--output=assigned"])),
    undefined,
  );
  assert.equal(output.value, "assigned");

  const copied = FlagSetValueOperations.$copy(source);
  assert.notEqual(copied, source);
  assert.equal(copied.Usage, usage);
  assert.equal(
    FlagSet.Parse(copied, RuntimeSlice.literal(["--output=copied"])),
    undefined,
  );
  assert.equal(output.value, "copied");
});
