import { readFile, readdir } from "node:fs/promises";

const sourceRoot = new URL("../src/", import.meta.url);
const packageFile = new URL("../package.json", import.meta.url);
const forbidden = [
  /\$argument/u,
  /__from_/u,
  /\$cooperative_/u,
  /\$contract/u,
  /\$state/u,
  /\b[A-Za-z_][A-Za-z0-9_]*_[A-Z][A-Za-z0-9_]*\s*\(/u,
];

const files = await sourceFiles(sourceRoot);
for (const file of files) {
  if (
    file.pathname.includes("/internal/") &&
    !file.pathname.endsWith("/internal/abi.ts")
  ) {
    continue;
  }
  const source = await readFile(file, "utf8");
  for (const pattern of forbidden) {
    if (pattern.test(source)) {
      throw new Error(
        `public provider source ${file.pathname} contains ${pattern}`,
      );
    }
  }
}

const packageManifest = JSON.parse(await readFile(packageFile, "utf8"));
const expectedExports = files
  .map((file) => file.pathname.slice(sourceRoot.pathname.length))
  .filter((path) => !path.startsWith("internal/") || path === "internal/abi.ts")
  .map((path) => `./${path.slice(0, -3)}.js`)
  .sort();
const actualExports = Object.keys(packageManifest.exports)
  .filter((path) => path !== "./package.json")
  .sort();
if (JSON.stringify(actualExports) !== JSON.stringify(expectedExports)) {
  throw new Error(
    `public export map differs from source modules:\n` +
      `actual=${JSON.stringify(actualExports)}\n` +
      `expected=${JSON.stringify(expectedExports)}`,
  );
}
for (const modulePath of expectedExports) {
  const sourcePath = modulePath.slice(2, -3);
  const target = packageManifest.exports[modulePath];
  const expectedTarget = {
    types: `./dist/src/${sourcePath}.d.ts`,
    default: `./dist/src/${sourcePath}.js`,
  };
  if (JSON.stringify(target) !== JSON.stringify(expectedTarget)) {
    throw new Error(
      `public export ${modulePath} has target ${JSON.stringify(target)}, ` +
        `want ${JSON.stringify(expectedTarget)}`,
    );
  }
}

async function sourceFiles(directory) {
  const result = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const child = new URL(entry.name + (entry.isDirectory() ? "/" : ""), directory);
    if (entry.isDirectory()) {
      result.push(...await sourceFiles(child));
    } else if (entry.name.endsWith(".ts")) {
      result.push(child);
    }
  }
  return result;
}
