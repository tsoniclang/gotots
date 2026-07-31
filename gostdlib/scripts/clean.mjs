import {
  readdir,
  rm,
  unlink,
} from "node:fs/promises";

await rm(new URL("../dist", import.meta.url), {
  force: true,
  recursive: true,
});

const runtimeDirectory = new URL("../test/runtime-package/", import.meta.url);
for (const entry of await readdir(runtimeDirectory, { withFileTypes: true })) {
  if (
    entry.isFile()
    && (entry.name.endsWith(".js") || entry.name.endsWith(".d.ts"))
  ) {
    await unlink(new URL(entry.name, runtimeDirectory));
  }
}
