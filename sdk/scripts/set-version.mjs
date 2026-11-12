#!/usr/bin/env node
// Stamps the client SDKs with a release's version, from its tag:
//
//   node sdk/scripts/set-version.mjs v1.7.0
//
// The SDKs' versions follow the server's: the TypeScript package
// (package.json and the User-Agent constant), the Python package
// (__version__, which pyproject.toml reads) and the Go module's User-Agent
// constant. A pre-release tag (v1.7.0-rc.1) becomes 1.7.0-rc.1 for npm and
// Go and 1.7.0rc1 for Python (PEP 440). In GitHub Actions the versions are
// also written to $GITHUB_OUTPUT as `version` and `python_version`.
import { appendFileSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const sdk = join(dirname(fileURLToPath(import.meta.url)), "..");
const tag = process.argv[2] ?? "";
const m = /^v?(\d+\.\d+\.\d+)(?:-(alpha|beta|rc)\.(\d+))?$/.exec(tag);
if (!m) {
  console.error(`set-version: ${JSON.stringify(tag)} isn't a release tag such as v1.7.0 or v1.7.0-rc.1`);
  process.exit(1);
}
const [, release, stage, n] = m;
const version = stage ? `${release}-${stage}.${n}` : release;
const python = stage ? `${release}${{ alpha: "a", beta: "b", rc: "rc" }[stage]}${n}` : release;

/** Replaces the one match of pattern in file with the line it makes of the version. */
function stamp(file, pattern, line) {
  const path = join(sdk, file);
  const text = readFileSync(path, "utf8");
  const matches = text.match(new RegExp(pattern.source, "gm")) ?? [];
  if (matches.length !== 1) {
    console.error(`set-version: ${file} has ${matches.length} lines matching ${pattern}, not one`);
    process.exit(1);
  }
  writeFileSync(path, text.replace(new RegExp(pattern.source, "m"), line));
}

const pkgPath = join(sdk, "client-ts", "package.json");
const pkg = JSON.parse(readFileSync(pkgPath, "utf8"));
pkg.version = version;
writeFileSync(pkgPath, `${JSON.stringify(pkg, null, 2)}\n`);
stamp("client-ts/src/version.ts", /^export const VERSION = ".*";$/, `export const VERSION = "${version}";`);
stamp("client-python/src/reputecast/_version.py", /^__version__ = ".*"$/, `__version__ = "${python}"`);
stamp("client-go/version.go", /^const Version = ".*"$/, `const Version = "${version}"`);

if (process.env.GITHUB_OUTPUT) {
  appendFileSync(process.env.GITHUB_OUTPUT, `version=${version}\npython_version=${python}\n`);
}
console.log(`client SDKs: ${version} (Python ${python})`);
