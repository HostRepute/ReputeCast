import { describe, expect, it } from "vitest";

import { defineAdapter } from "../src/define.ts";
import { ManifestError, capabilityDifferences, operationsFor, validateManifest } from "../src/manifest.ts";
import { manifest } from "./fixtures.ts";

function problems(change: (m: Record<string, any>) => void): string {
  const m = manifest();
  change(m);
  try {
    validateManifest(m);
  } catch (e) {
    expect(e).toBeInstanceOf(ManifestError);
    return (e as ManifestError).problems.join(" | ");
  }
  return "";
}

describe("manifest", () => {
  it("accepts a valid manifest", () => {
    expect(validateManifest(manifest()).id).toBe("fake-esp");
  });

  // The same rules as ReputeCast's install check (internal/providers/remote/manifest.go).
  it.each([
    ["kind", (m: any) => (m.kind = "http"), 'kind must be "remote"'],
    ["protocol", (m: any) => (m.protocol = 2), '"protocol": 1'],
    ["id", (m: any) => (m.id = "Fake ESP"), "id must be"],
    ["pricing", (m: any) => delete m.pricing, "pricing"],
    ["unknown capability", (m: any) => (m.capabilities.supressionSync = "pull"), "supressionSync is not a capability"],
    ["batch", (m: any) => (m.capabilities.batch = 5000), "capabilities.batch"],
    ["suppression value", (m: any) => (m.capabilities.suppressionSync = true), "capabilities.suppressionSync"],
    ["register without webhooks", (m: any) => (m.capabilities.webhooks = false), "registerWebhooks needs"],
    ["bulk", (m: any) => (m.terms = { bulkMarketing: "broadcast_streams_only" }), "bulkMarketing"],
    ["secret field", (m: any) => (m.config = [{ key: "server_token", label: "Server token" }]), "looks like a secret"],
    ["duplicate field", (m: any) => (m.config = [{ key: "region", label: "Region" }, { key: "region", label: "Region" }]), "declared twice"],
    ["network host", (m: any) => (m.permissions.network = ["https://api.example.com/v1"]), "not a host name"],
    ["network setting", (m: any) => (m.permissions.network = ["settings:server_url"]), "doesn't declare"],
  ])("refuses %s", (_name, change, want) => {
    expect(problems(change)).toContain(want);
  });

  it("lists every problem at once", () => {
    const p = problems((m) => {
      m.id = "X";
      delete m.pricing;
    });
    expect(p).toContain("id must be");
    expect(p).toContain("pricing");
  });

  it("derives the operations ReputeCast may call", () => {
    expect(operationsFor({})).toEqual(["health", "verify", "send"]);
    expect(operationsFor({ webhooks: true, suppressionSync: "pull", quotaDiscovery: true })).toEqual([
      "health",
      "verify",
      "send",
      "webhooks/verify",
      "webhooks/parse",
      "quota",
      "suppressions/list",
    ]);
    expect(operationsFor(manifest().capabilities)).toHaveLength(11);
  });

  it("reports where the code and manifest.json disagree", () => {
    expect(capabilityDifferences({ batch: 1 }, {})).toEqual([]);
    expect(capabilityDifferences({ batch: 100, webhooks: true }, { batch: 50, webhooks: true })).toEqual(["batch: 100 in the code, 50 in manifest.json"]);
  });
});

describe("defineAdapter", () => {
  const send = async () => ({ providerMessageId: "x" });

  it("needs a hook for every declared capability", () => {
    expect(() => defineAdapter({ id: "fake-esp", capabilities: { webhooks: true }, send })).toThrow(/verifyWebhook\(\).*normalizeEvent\(\)/s);
    expect(() => defineAdapter({ id: "fake-esp", capabilities: { suppressionSync: "two-way" }, send, syncSuppressions: async () => ({ suppressions: [] }) })).toThrow(
      /pushSuppressions/,
    );
    expect(() => defineAdapter({ id: "fake-esp", capabilities: { quotaDiscovery: true }, send })).toThrow(/discoverQuota/);
    expect(() => defineAdapter({ id: "fake-esp", capabilities: {} })).toThrow(/send\(\) or sendBatch\(\)/);
    expect(() => defineAdapter({ id: "Fake", capabilities: {}, send })).toThrow(/id must be/);
    expect(() => defineAdapter({ id: "fake-esp", capabilities: { batch: 0 }, send })).toThrow(/batch/);
  });

  it("spells out the capabilities", () => {
    const a = defineAdapter({ id: "fake-esp", capabilities: { batch: 10 }, send });
    expect(a.normalized).toMatchObject({ batch: 10, webhooks: false, suppressionSync: false, quotaDiscovery: false });
    expect(Object.isFrozen(a)).toBe(true);
  });
});
