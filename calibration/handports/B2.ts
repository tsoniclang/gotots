// Hand port B2 — defer + named results + external contract: the named
// result stays visible to the deferred call; the semaphore is the
// external emulation's contract. Ordinary shape otherwise.
ReadFile(path: string): readonly [string, boolean] {
  const release = readSema.Acquire();
  try {
    return this.common.ReadFile(path);
  } finally {
    release();
  }
}
