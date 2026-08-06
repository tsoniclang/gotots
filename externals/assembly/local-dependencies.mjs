import { lstat, mkdir, readFile, readlink, symlink } from "node:fs/promises";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const providerRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repositoryRoot = resolve(providerRoot, "..");
const dependencyRoot = join(repositoryRoot, "gostdlib");
const providerPackage = JSON.parse(
  await readFile(join(providerRoot, "package.json"), "utf8"),
);
const dependencyPackage = JSON.parse(
  await readFile(join(dependencyRoot, "package.json"), "utf8"),
);

if (
  dependencyPackage.name !== "@gotots/gostdlib" ||
  providerPackage.peerDependencies?.[dependencyPackage.name] !==
    dependencyPackage.version
) {
  throw new Error("local gostdlib package does not match the peer contract");
}

const link = join(
  providerRoot,
  "node_modules",
  "@gotots",
  "gostdlib",
);
const target = relative(dirname(link), dependencyRoot);
await mkdir(dirname(link), { recursive: true });

try {
  const existing = await lstat(link);
  if (!existing.isSymbolicLink() || (await readlink(link)) !== target) {
    throw new Error("local gostdlib assembly path is occupied or stale");
  }
} catch (error) {
  if (error?.code !== "ENOENT") {
    throw error;
  }
  await symlink(target, link, "dir");
}
