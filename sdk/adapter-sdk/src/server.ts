import { createServer } from "node:http";
import type { IncomingMessage, Server, ServerResponse } from "node:http";
import type { AddressInfo } from "node:net";

import type { Adapter, AdapterContext } from "./define.ts";
import { isAdapter } from "./define.ts";
import { dispatch } from "./dispatch.ts";
import { isAdapterError, statusOf, toWireError } from "./errors.ts";
import { allowHostsFromEnv, createHttpClient, hostAllowlist } from "./http.ts";
import { IdempotencyCache } from "./idempotency.ts";
import type { Answer } from "./idempotency.ts";
import type { Logger } from "./log.ts";
import { consoleLogger, silentLogger } from "./log.ts";
import type { Manifest } from "./manifest.ts";
import { ManifestError, capabilityDifferences, operationsFor, validateManifest } from "./manifest.ts";
import type { Connection, Envelope, Operation, ResponseEnvelope, WireError } from "./protocol.ts";
import { HEADERS, PROTOCOL_VERSION, TIMEOUTS, isOperation } from "./protocol.ts";
import { MAX_SKEW_SECONDS, secretsFromEnv, sign, verifySignature } from "./signing.ts";

export interface ServeOptions {
  /** PORT, default 4600 (0 picks a free port). */
  port?: number;
  /** HOST, default 127.0.0.1; containers use 0.0.0.0. */
  host?: string;
  /** The signing secret(s) ReputeCast issued; default REPUTECAST_ADAPTER_SECRET (comma-separated during a rotation). */
  secret?: string | readonly string[];
  /** manifest.json: checked against the code at start, and the source of the network permissions. */
  manifest?: unknown;
  /** Hosts allowed on top of the manifest's (REPUTECAST_ADAPTER_ALLOW_HOSTS). */
  allowHosts?: readonly string[];
  /** Largest request body (32 MiB). */
  maxBodyBytes?: number;
  /** Where logs go; false for none (default: JSON lines on stderr). */
  logger?: Logger | false;
  /** How long successful answers are remembered per idempotency key (15 minutes). */
  idempotencyTtlMs?: number;
  /** The adapter's version in health answers (default: the manifest's). */
  version?: string;
  now?: () => number;
}

export interface AdapterServer {
  /** http://host:port */
  readonly url: string;
  readonly port: number;
  readonly server: Server;
  close(): Promise<void>;
}

type Handler = (req: IncomingMessage, res: ServerResponse) => void;

interface Setup {
  adapter: Adapter;
  secrets: string[];
  manifest: Manifest | undefined;
  network: readonly string[] | undefined;
  allowHosts: readonly string[];
  maxBody: number;
  log: Logger;
  cache: IdempotencyCache;
  version: string;
  operations: Set<Operation>;
  now: () => number;
}

function setup(adapter: Adapter, o: ServeOptions): Setup {
  if (!isAdapter(adapter)) throw new TypeError("serve() takes an adapter made with defineAdapter()");
  const secrets = o.secret === undefined ? secretsFromEnv() : (typeof o.secret === "string" ? [o.secret] : [...o.secret]).filter((s) => s !== "");
  if (secrets.length === 0) {
    throw new Error(
      `Set REPUTECAST_ADAPTER_SECRET to the signing secret ReputeCast issued for ${adapter.id} (reputecast adapter install prints it; reputecast adapter secret ${adapter.id} issues a new one).`,
    );
  }
  let manifest: Manifest | undefined;
  if (o.manifest !== undefined) {
    manifest = validateManifest(o.manifest);
    const problems: string[] = [];
    if (manifest.id !== adapter.id) problems.push(`manifest.json has the id ${manifest.id}, the code ${adapter.id}`);
    problems.push(...capabilityDifferences(adapter.capabilities, manifest.capabilities));
    if (adapter.terms && adapter.terms.bulkMarketing !== manifest.terms.bulkMarketing) {
      problems.push(`terms.bulkMarketing: ${adapter.terms.bulkMarketing} in the code, ${manifest.terms.bulkMarketing} in manifest.json`);
    }
    if (problems.length > 0) throw new ManifestError(problems.map((p) => "the code and manifest.json disagree: " + p));
  }
  return {
    adapter,
    secrets,
    manifest,
    network: manifest?.permissions?.network,
    allowHosts: [...(o.allowHosts ?? []), ...allowHostsFromEnv()],
    maxBody: o.maxBodyBytes ?? 32 << 20,
    log: o.logger === false ? silentLogger : (o.logger ?? consoleLogger()),
    cache: new IdempotencyCache({ ttlMs: o.idempotencyTtlMs ?? 15 * 60_000, now: o.now ?? Date.now }),
    version: o.version ?? manifest?.version ?? adapter.version ?? "0.0.0",
    operations: new Set(operationsFor(adapter.capabilities)),
    now: o.now ?? Date.now,
  };
}

class TooLarge extends Error {}

// Past the limit the rest of the body is read and dropped, so the client
// gets the 413 answer rather than a reset connection.
function readBody(req: IncomingMessage, max: number): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    let size = 0;
    let over = false;
    req.on("data", (c: Buffer) => {
      size += c.length;
      if (over) return;
      if (size > max) {
        over = true;
        chunks.length = 0;
        return;
      }
      chunks.push(c);
    });
    req.on("end", () => (over ? reject(new TooLarge()) : resolve(Buffer.concat(chunks))));
    req.on("error", reject);
  });
}

function errorAnswer(key: string, e: WireError): Answer {
  const body: ResponseEnvelope = { protocol: PROTOCOL_VERSION, idempotencyKey: key, error: e };
  return { status: statusOf(e.code), body: JSON.stringify(body) };
}

function header(req: IncomingMessage, name: string): string | undefined {
  const v = req.headers[name];
  return Array.isArray(v) ? v[0] : v;
}

const KEY = /^[A-Za-z0-9._:-]{1,128}$/;

function isConnection(c: unknown): c is Connection {
  if (typeof c !== "object" || c === null) return false;
  const r = c as Record<string, unknown>;
  const flat = (v: unknown) =>
    typeof v === "object" && v !== null && !Array.isArray(v) && Object.values(v).every((x) => ["string", "number", "boolean"].includes(typeof x));
  return typeof r["providerId"] === "string" && typeof r["workspaceId"] === "string" && flat(r["settings"]) && flat(r["credentials"]);
}

/** A node:http request handler speaking the protocol (serve() uses it; embed it in your own server if you prefer). */
export function createHandler(adapter: Adapter, opts: ServeOptions = {}): Handler {
  const s = setup(adapter, opts);
  const reply = (res: ServerResponse, a: Answer) => {
    const ts = Math.floor(s.now() / 1000);
    res.writeHead(a.status, {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store",
      [HEADERS.timestamp]: String(ts),
      [HEADERS.signature]: "v1=" + sign(s.secrets[0]!, ts, a.body),
    });
    res.end(a.body);
  };

  const handle = async (req: IncomingMessage, res: ServerResponse): Promise<void> => {
    const started = s.now();
    const path = (req.url ?? "/").split("?")[0] ?? "/";
    if (req.method === "GET" && path === "/healthz") {
      res.writeHead(200, { "content-type": "application/json", "cache-control": "no-store" });
      res.end('{"ok":true}');
      return;
    }
    const op = path.startsWith("/v1/") ? path.slice(4) : "";
    // Refuse requests that can't be signed in time before reading their body.
    const ts = Number(header(req, HEADERS.timestamp) ?? NaN);
    if (!header(req, HEADERS.signature) || !Number.isFinite(ts) || Math.abs(s.now() / 1000 - ts) > MAX_SKEW_SECONDS) {
      s.log.warn("refused an unauthenticated request", { op, reason: "missing or stale signature" });
      res.setHeader("connection", "close");
      reply(res, errorAnswer("", { code: "unauthenticated", message: "Refused: the request is not signed, or its timestamp is more than 5 minutes off." }));
      return;
    }
    let body: Buffer;
    try {
      body = await readBody(req, s.maxBody);
    } catch (e) {
      if (e instanceof TooLarge) reply(res, errorAnswer("", { code: "too_large", message: `The request is larger than ${s.maxBody} bytes.` }));
      return;
    }
    if (req.method !== "POST" || !isOperation(op)) {
      reply(res, errorAnswer("", { code: "not_supported", message: `No operation at ${req.method} ${path}.` }));
      return;
    }
    const check = verifySignature(s.secrets, header(req, HEADERS.timestamp), header(req, HEADERS.signature), body, s.now());
    if (!check.ok) {
      const why = { missing: "the request is not signed", stale: "the signature's timestamp is more than 5 minutes off; check both clocks", mismatch: "the signature doesn't match this adapter's secret" }[check.reason];
      s.log.warn("refused an unauthenticated request", { op, reason: check.reason });
      reply(res, errorAnswer("", { code: "unauthenticated", message: `Refused: ${why}.` }));
      return;
    }
    let env: Envelope;
    try {
      env = JSON.parse(body.toString("utf8")) as Envelope;
    } catch {
      reply(res, errorAnswer("", { code: "invalid_request", message: "The body is not JSON." }));
      return;
    }
    const key = typeof env?.idempotencyKey === "string" && KEY.test(env.idempotencyKey) ? env.idempotencyKey : "";
    const deadline = Date.parse(env?.deadline ?? "");
    const problem =
      env?.protocol !== PROTOCOL_VERSION
        ? `protocol ${String(env?.protocol)} is not ${PROTOCOL_VERSION}`
        : env.operation !== op
          ? `the body's operation ${JSON.stringify(env.operation)} doesn't match the path ${op}`
          : env.adapter !== s.adapter.id
            ? `this is the ${s.adapter.id} adapter, not ${JSON.stringify(env.adapter)}`
            : key === ""
              ? "the idempotencyKey is missing or malformed"
              : Number.isNaN(deadline)
                ? "the deadline is not a time"
                : op !== "health" && !isConnection(env.connection)
                  ? "the connection is missing or malformed"
                  : "";
    if (problem !== "") {
      reply(res, errorAnswer(key, { code: "invalid_request", message: `Invalid request: ${problem}.` }));
      return;
    }
    if (!s.operations.has(op)) {
      reply(res, errorAnswer(key, { code: "not_supported", message: `The ${s.adapter.id} adapter doesn't declare ${op}.` }));
      return;
    }
    const abort = new AbortController();
    res.on("close", () => {
      if (!res.writableFinished) abort.abort(new Error("ReputeCast closed the connection"));
    });
    const answer = await s.cache.run(`${op}:${env.connection?.providerId ?? ""}:${key}`, () => execute(s, op, env, key, deadline, abort.signal));
    reply(res, answer);
    const fields: Record<string, unknown> = { op, status: answer.status, ms: s.now() - started, key };
    if (answer.status === 200) s.log.info("handled", fields);
    else {
      try {
        const e = (JSON.parse(answer.body) as ResponseEnvelope).error;
        s.log.warn("failed", { ...fields, code: e?.code, error: e?.message });
      } catch {
        s.log.warn("failed", fields);
      }
    }
  };

  return (req, res) => {
    handle(req, res).catch((e: unknown) => {
      s.log.error("request failed", { error: e instanceof Error ? e.message : String(e) });
      if (!res.headersSent) reply(res, errorAnswer("", { code: "internal", message: "The adapter failed." }));
      else res.destroy();
    });
  };
}

async function execute(s: Setup, op: Operation, env: Envelope, key: string, deadline: number, closed: AbortSignal): Promise<Answer> {
  const ms = Math.max(1, Math.min(deadline - s.now(), TIMEOUTS[op]));
  const signal = AbortSignal.any([AbortSignal.timeout(ms), closed]);
  const connection: Connection = env.connection ?? { providerId: "", workspaceId: "", settings: {}, credentials: {} };
  const base: Omit<AdapterContext, "http"> = {
    adapter: s.adapter.id,
    operation: op,
    idempotencyKey: key,
    connection,
    settings: connection.settings,
    credentials: connection.credentials,
    webhookSecret: connection.webhookSecret,
    signal,
    deadline: new Date(s.now() + ms),
    log: s.log,
  };
  try {
    const h = s.adapter.http;
    const baseUrl = h ? (typeof h.baseUrl === "function" ? h.baseUrl(base) : h.baseUrl) : "http://adapter.invalid";
    let entries = s.network;
    if (!entries || entries.length === 0) {
      try {
        entries = [new URL(baseUrl).host];
      } catch {
        entries = [];
      }
    }
    const http = createHttpClient({
      baseUrl,
      headers: h?.headers ? h.headers(base) : {},
      signal,
      allow: hostAllowlist(entries, connection.settings, s.allowHosts),
      ...(h?.errorCode ? { errorCode: h.errorCode } : {}),
      ...(h?.errorMessage ? { errorMessage: h.errorMessage } : {}),
      name: h?.name ?? s.manifest?.name ?? s.adapter.id,
    });
    const result = await dispatch(s.adapter, op, env.input, { ...base, http }, s.version);
    const body: ResponseEnvelope = { protocol: PROTOCOL_VERSION, idempotencyKey: key, result };
    return { status: 200, body: JSON.stringify(body) };
  } catch (e) {
    if (signal.aborted && !isAdapterError(e)) {
      return errorAnswer(key, { code: "provider_unavailable", message: `The ${op} call ran past its deadline.` });
    }
    return errorAnswer(key, toWireError(e));
  }
}

/** Serves the adapter protocol until close(). */
export async function serve(adapter: Adapter, opts: ServeOptions = {}): Promise<AdapterServer> {
  const handler = createHandler(adapter, opts);
  const log = opts.logger === false ? silentLogger : (opts.logger ?? consoleLogger());
  const server = createServer(handler);
  // Longer than ReputeCast's idle connections (90 s), so it closes them first.
  server.keepAliveTimeout = 95_000;
  server.headersTimeout = 100_000;
  server.requestTimeout = 120_000;
  const port = opts.port ?? Number(process.env["PORT"] ?? 4600);
  const host = opts.host ?? process.env["HOST"] ?? "127.0.0.1";
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(port, host, () => {
      server.off("error", reject);
      resolve();
    });
  });
  const addr = server.address() as AddressInfo;
  const url = `http://${addr.family === "IPv6" ? `[${addr.address}]` : addr.address}:${addr.port}`;
  log.info("adapter listening", { adapter: adapter.id, url, protocol: PROTOCOL_VERSION });
  return {
    url,
    port: addr.port,
    server,
    close: () =>
      new Promise<void>((resolve) => {
        server.close(() => resolve());
        server.closeIdleConnections();
      }),
  };
}
