import type { Operation } from "./protocol.ts";
import { PROTOCOL_VERSION } from "./protocol.ts";

/** What an adapter can do; ReputeCast calls the operations they enable and nothing else. */
export interface Capabilities {
  /** Messages per send call, 1 to 1000 (default 1). */
  batch?: number;
  /** Verifies and normalizes the provider's webhooks. */
  webhooks?: boolean;
  /** Registers ReputeCast's webhook URL at the provider (needs webhooks). */
  registerWebhooks?: boolean;
  domainSetup?: "full" | "dkim_only" | false;
  suppressionSync?: "pull" | "push" | "two-way" | false;
  /** true for quota, "rate" when the API also reports the send rate. */
  quotaDiscovery?: boolean | "rate";
  sandbox?: "detection" | "test token" | "mode" | false;
  /** The dedicated-IP option ("optional, $25 per IP per month") or false. */
  dedicatedIps?: string | false;
  rewritesLinks?: boolean;
  addsOpenPixel?: boolean;
}

/** Capabilities with every value spelled out. */
export interface NormalizedCapabilities {
  batch: number;
  webhooks: boolean;
  registerWebhooks: boolean;
  domainSetup: "full" | "dkim_only" | false;
  suppressionSync: "pull" | "push" | "two-way" | false;
  quotaDiscovery: boolean | "rate";
  sandbox: "detection" | "test token" | "mode" | false;
  dedicatedIps: string | false;
  rewritesLinks: boolean;
  addsOpenPixel: boolean;
}

export interface Terms {
  bulkMarketing: "allowed" | "forbidden" | "your_policy";
  note?: string;
  url?: string;
}

export interface ConfigField {
  key: string;
  label: string;
  secret?: boolean;
  required?: boolean;
  options?: string[];
  help?: string;
}

/** manifest.json of a community adapter (kind "remote"). */
export interface Manifest {
  protocol: 1;
  kind: "remote";
  id: string;
  name: string;
  version: string;
  maintainer: string;
  description?: string;
  homepage?: string;
  source_url?: string;
  last_release?: string;
  latest_version?: string;
  capabilities: Capabilities;
  terms: Terms;
  pricing: string;
  config?: ConfigField[];
  permissions?: { network?: string[] };
}

export const CAPABILITY_KEYS = [
  "batch",
  "webhooks",
  "registerWebhooks",
  "domainSetup",
  "suppressionSync",
  "quotaDiscovery",
  "sandbox",
  "dedicatedIps",
  "rewritesLinks",
  "addsOpenPixel",
] as const;

export const ID_PATTERN = /^[a-z][a-z0-9-]{1,39}$/;
const FIELD_KEY = /^[a-z][a-z0-9_]{0,39}$/;
const HOST = /^(\*\.)?[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*(:[0-9]{1,5})?$/;

/** A manifest or adapter definition that breaks the rules; every problem is listed. */
export class ManifestError extends Error {
  readonly problems: string[];
  constructor(problems: string[]) {
    super(problems.join("; "));
    this.name = "ManifestError";
    this.problems = problems;
  }
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

function oneOf(v: unknown, allowed: readonly (string | boolean)[]): boolean {
  return allowed.includes(v as string | boolean);
}

/** Checks a capabilities object and returns its problems. */
export function capabilityProblems(c: unknown): string[] {
  if (!isRecord(c)) return ["capabilities must be an object"];
  const out: string[] = [];
  for (const [k, v] of Object.entries(c)) {
    switch (k) {
      case "batch":
        if (typeof v !== "number" || !Number.isInteger(v) || v < 1 || v > 1000) out.push("capabilities.batch must be a whole number from 1 to 1000");
        break;
      case "webhooks":
      case "registerWebhooks":
      case "rewritesLinks":
      case "addsOpenPixel":
        if (typeof v !== "boolean") out.push(`capabilities.${k} must be true or false`);
        break;
      case "domainSetup":
        if (!oneOf(v, [false, "full", "dkim_only"])) out.push('capabilities.domainSetup must be false, "full" or "dkim_only"');
        break;
      case "suppressionSync":
        if (!oneOf(v, [false, "pull", "push", "two-way"])) out.push('capabilities.suppressionSync must be false, "pull", "push" or "two-way"');
        break;
      case "quotaDiscovery":
        if (!oneOf(v, [false, true, "rate"])) out.push('capabilities.quotaDiscovery must be true, "rate" or false');
        break;
      case "sandbox":
        if (!oneOf(v, [false, "detection", "test token", "mode"])) out.push('capabilities.sandbox must be false, "detection", "test token" or "mode"');
        break;
      case "dedicatedIps":
        if (v !== false && (typeof v !== "string" || v.trim() === "" || v.length > 80)) out.push("capabilities.dedicatedIps must be false or a text of at most 80 characters");
        break;
      default:
        out.push(`capabilities.${k} is not a capability (use ${CAPABILITY_KEYS.join(", ")})`);
    }
  }
  if (c["registerWebhooks"] === true && c["webhooks"] !== true) out.push("capabilities.registerWebhooks needs capabilities.webhooks");
  return out;
}

/** Spells out every capability. */
export function normalizeCapabilities(c: Capabilities): NormalizedCapabilities {
  return {
    batch: c.batch ?? 1,
    webhooks: c.webhooks ?? false,
    registerWebhooks: c.registerWebhooks ?? false,
    domainSetup: c.domainSetup ?? false,
    suppressionSync: c.suppressionSync ?? false,
    quotaDiscovery: c.quotaDiscovery ?? false,
    sandbox: c.sandbox ?? false,
    dedicatedIps: c.dedicatedIps ?? false,
    rewritesLinks: c.rewritesLinks ?? false,
    addsOpenPixel: c.addsOpenPixel ?? false,
  };
}

/** The differences between two declarations ("batch: 100 in the code, 50 in manifest.json"). */
export function capabilityDifferences(code: Capabilities, manifest: Capabilities): string[] {
  const a = normalizeCapabilities(code);
  const b = normalizeCapabilities(manifest);
  const out: string[] = [];
  for (const k of CAPABILITY_KEYS) {
    if (a[k] !== b[k]) out.push(`${k}: ${JSON.stringify(a[k])} in the code, ${JSON.stringify(b[k])} in manifest.json`);
  }
  return out;
}

/** The operations ReputeCast may call for a declaration. */
export function operationsFor(c: Capabilities): Operation[] {
  const n = normalizeCapabilities(c);
  const ops: Operation[] = ["health", "verify", "send"];
  if (n.webhooks) ops.push("webhooks/verify", "webhooks/parse");
  if (n.registerWebhooks) ops.push("webhooks/register");
  if (n.domainSetup) ops.push("domains/add", "domains/status");
  if (n.quotaDiscovery) ops.push("quota");
  if (n.suppressionSync === "pull" || n.suppressionSync === "two-way") ops.push("suppressions/list");
  if (n.suppressionSync === "push" || n.suppressionSync === "two-way") ops.push("suppressions/add");
  return ops;
}

function looksSecret(key: string): boolean {
  return key.includes("password") || key.includes("token") || key.includes("secret") || key.endsWith("api_key");
}

function text(m: Record<string, unknown>, key: string, max: number, required: boolean, out: string[], label = key): string {
  const v = m[key];
  if (v === undefined && !required) return "";
  if (typeof v !== "string" || (required && v.trim() === "") || v.length > max) {
    out.push(required ? `the manifest needs a ${label} of at most ${max} characters` : `the manifest's ${label} must be a text of at most ${max} characters`);
    return "";
  }
  return v.trim();
}

function link(v: unknown, label: string, out: string[]): void {
  if (v === undefined) return;
  try {
    const u = new URL(String(v));
    if (u.protocol !== "https:" && u.protocol !== "http:") throw new Error("scheme");
  } catch {
    out.push(`the manifest's ${label} is not an http(s) URL`);
  }
}

/**
 * Validates manifest.json with the rules ReputeCast applies at install time
 * (internal/providers/remote/manifest.go) and returns it typed. Throws a
 * ManifestError listing every problem.
 */
export function validateManifest(input: unknown): Manifest {
  if (!isRecord(input)) throw new ManifestError(["the manifest must be a JSON object"]);
  const m = input;
  const out: string[] = [];
  if (m["kind"] !== "remote") out.push(`the manifest's kind must be "remote" (a process speaking the adapter protocol)`);
  if (m["protocol"] !== PROTOCOL_VERSION) out.push(`the manifest needs "protocol": ${PROTOCOL_VERSION}`);
  const id = m["id"];
  if (typeof id !== "string" || !ID_PATTERN.test(id)) out.push("the manifest's id must be 2 to 40 characters: lowercase letters, digits and dashes, starting with a letter");
  text(m, "name", 80, true, out);
  text(m, "version", 40, true, out);
  text(m, "maintainer", 80, true, out);
  text(m, "description", 500, false, out);
  text(m, "pricing", 80, true, out, 'pricing model ("$0.80 per 1,000")');
  link(m["homepage"], "homepage", out);
  link(m["source_url"], "source_url", out);
  if (m["last_release"] !== undefined && (typeof m["last_release"] !== "string" || Number.isNaN(Date.parse(m["last_release"])))) {
    out.push("the manifest's last_release must be an RFC 3339 time");
  }
  out.push(...capabilityProblems(m["capabilities"]));
  const terms = m["terms"];
  if (!isRecord(terms) || !oneOf(terms["bulkMarketing"], ["allowed", "forbidden", "your_policy"])) {
    out.push("the manifest's terms.bulkMarketing must be allowed, forbidden or your_policy");
  } else {
    if (terms["note"] !== undefined && (typeof terms["note"] !== "string" || terms["note"].length > 500)) out.push("the manifest's terms.note must be a text of at most 500 characters");
    link(terms["url"], "terms.url", out);
  }
  const keys = new Set<string>();
  const config = m["config"] ?? [];
  if (!Array.isArray(config)) {
    out.push("the manifest's config must be a list of fields");
  } else {
    if (config.length > 30) out.push("the manifest's config has more than 30 fields");
    for (const f of config) {
      if (!isRecord(f) || typeof f["key"] !== "string") {
        out.push("every config field needs a key");
        continue;
      }
      const key = f["key"];
      if (!FIELD_KEY.test(key)) out.push(`config field "${key}": keys are lowercase letters, digits and underscores, starting with a letter`);
      if (keys.has(key)) out.push(`config field "${key}" is declared twice`);
      keys.add(key);
      if (typeof f["label"] !== "string" || f["label"].trim() === "" || f["label"].length > 80) out.push(`config field "${key}" needs a label of at most 80 characters`);
      if (f["help"] !== undefined && (typeof f["help"] !== "string" || f["help"].length > 300)) out.push(`config field "${key}": help is longer than 300 characters`);
      if (looksSecret(key) && f["secret"] !== true) out.push(`config field "${key}" looks like a secret: mark it "secret": true so it is sealed and never shown again`);
      const options = f["options"];
      if (options !== undefined) {
        if (!Array.isArray(options) || options.some((o) => typeof o !== "string") || options.length > 50) out.push(`config field "${key}": options must be at most 50 texts`);
        else if (f["secret"] === true) out.push(`config field "${key}" can't be both secret and a list of options`);
      }
    }
  }
  const perms = m["permissions"];
  if (perms !== undefined) {
    const network = isRecord(perms) ? (perms["network"] ?? []) : null;
    if (!Array.isArray(network)) {
      out.push("the manifest's permissions.network must be a list of hosts");
    } else {
      if (network.length > 20) out.push("the manifest's permissions.network lists more than 20 hosts");
      for (const h of network) {
        if (typeof h !== "string") {
          out.push("permissions.network entries are texts");
        } else if (h.startsWith("settings:")) {
          if (!keys.has(h.slice("settings:".length))) out.push(`permissions.network "${h}" names a setting the config doesn't declare`);
        } else if (!HOST.test(h)) {
          out.push(`permissions.network "${h}" is not a host name (api.example.com, *.example.com or host:port)`);
        }
      }
    }
  }
  if (out.length > 0) throw new ManifestError(out);
  return input as unknown as Manifest;
}
