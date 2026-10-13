import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

import { runContract } from "./testing/index.ts";
import type { Contract } from "./testing/index.ts";
import { SDK_VERSION } from "./version.ts";

const USAGE = `reputecast-adapter ${SDK_VERSION}

Usage:
  reputecast-adapter test [--config reputecast.contract.ts]
      Runs the contract suite against the adapter over the protocol: the
      scenarios ReputeCast's Go suite runs, plus signatures, idempotency
      and declared operations. Exits 1 when a scenario fails.

  reputecast-adapter help

Docs: https://github.com/hostrepute/reputecast/blob/main/docs/adapters.md#community-adapters
`;

const DEFAULT_CONFIGS = ["reputecast.contract.ts", "reputecast.contract.js", "reputecast.contract.mjs"];

export async function main(argv: string[]): Promise<number> {
  const [cmd, ...rest] = argv;
  if (cmd === undefined || cmd === "help" || cmd === "--help" || cmd === "-h") {
    process.stdout.write(USAGE);
    return cmd === undefined ? 2 : 0;
  }
  if (cmd === "--version" || cmd === "version") {
    process.stdout.write(SDK_VERSION + "\n");
    return 0;
  }
  if (cmd !== "test") {
    process.stderr.write(`Unknown command ${JSON.stringify(cmd)}.\n\n${USAGE}`);
    return 2;
  }
  let config: string | undefined;
  for (let i = 0; i < rest.length; i++) {
    const a = rest[i] as string;
    if (a === "--config" || a === "-c") config = rest[++i];
    else if (a.startsWith("--config=")) config = a.slice("--config=".length);
    else {
      process.stderr.write(`Unknown option ${a}.\n`);
      return 2;
    }
  }
  const path = config ? resolve(config) : DEFAULT_CONFIGS.map((c) => resolve(c)).find((p) => existsSync(p));
  if (!path || !existsSync(path)) {
    process.stderr.write(`No contract found: create reputecast.contract.ts (see the template) or pass --config.\n`);
    return 2;
  }
  const mod = (await import(pathToFileURL(path).href)) as { default?: Contract; contract?: Contract };
  const contract = mod.default ?? mod.contract;
  if (!contract || typeof contract.api !== "function") {
    process.stderr.write(`${path} must export a contract (export default defineContract({...})).\n`);
    return 2;
  }
  process.stdout.write(`Contract suite for ${contract.adapter?.id ?? "the adapter"} (adapter protocol 1)\n`);
  const report = await runContract(contract, { log: (line) => process.stdout.write(line + "\n") });
  process.stdout.write(`\n${report.passed} passed, ${report.failed} failed, ${report.skipped} skipped\n`);
  return report.failed > 0 ? 1 : 0;
}
