import { expect, it } from "vitest";

import { runContract } from "@reputecast/adapter-sdk/testing";

import contract from "../reputecast.contract.ts";

it("passes ReputeCast's contract suite", async () => {
  const lines: string[] = [];
  const report = await runContract(contract, { log: (line) => lines.push(line) });
  expect(report.failed, lines.join("\n")).toBe(0);
  expect(report.skipped).toBe(0);
});
