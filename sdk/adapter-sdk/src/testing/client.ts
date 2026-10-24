import { randomUUID } from "node:crypto";

import type { Connection, Envelope, Operation, ResponseEnvelope } from "../protocol.ts";
import { HEADERS, PROTOCOL_VERSION, TIMEOUTS } from "../protocol.ts";
import { signatureHeader, verifySignature } from "../signing.ts";
import { trimTrailingSlashes } from "../url.ts";

export interface CallOptions {
  connection?: Connection;
  /** Default: a fresh UUID. */
  key?: string;
  /** How long the adapter has; default: ReputeCast's timeout for the operation. */
  deadlineMs?: number;
  /** Break the request on purpose (authentication scenarios). */
  unsigned?: boolean;
  secret?: string;
  timestampOffsetSeconds?: number;
  operation?: string;
  adapter?: string;
  path?: string;
}

export interface CallResult<R> {
  status: number;
  body: ResponseEnvelope<R>;
  /** Whether the answer carries a valid signature. */
  signed: boolean;
  raw: string;
}

/** Plays ReputeCast's side of the protocol: signs requests and checks answers. */
export class ProtocolClient {
  readonly url: string;
  readonly secrets: string[];
  readonly adapter: string;

  constructor(o: { url: string; secret: string | string[]; adapter: string }) {
    this.url = trimTrailingSlashes(o.url);
    this.secrets = typeof o.secret === "string" ? [o.secret] : o.secret;
    this.adapter = o.adapter;
  }

  async call<R = unknown>(op: Operation, input: unknown, o: CallOptions = {}): Promise<CallResult<R>> {
    const key = o.key ?? randomUUID();
    const env: Envelope = {
      protocol: PROTOCOL_VERSION,
      operation: (o.operation ?? op) as Operation,
      adapter: o.adapter ?? this.adapter,
      idempotencyKey: key,
      deadline: new Date(Date.now() + (o.deadlineMs ?? TIMEOUTS[op])).toISOString(),
      input,
    };
    if (o.connection) env.connection = o.connection;
    const body = JSON.stringify(env);
    const ts = Math.floor(Date.now() / 1000) + (o.timestampOffsetSeconds ?? 0);
    const headers: Record<string, string> = {
      "content-type": "application/json",
      [HEADERS.protocol]: String(PROTOCOL_VERSION),
      [HEADERS.operation]: op,
      [HEADERS.idempotency]: key,
      [HEADERS.timestamp]: String(ts),
    };
    if (!o.unsigned) headers[HEADERS.signature] = signatureHeader(o.secret ? [o.secret] : this.secrets, ts, body);
    const res = await fetch(this.url + (o.path ?? "/v1/" + op), { method: "POST", headers, body, signal: AbortSignal.timeout(TIMEOUTS[op] + 5_000) });
    const raw = await res.text();
    const signed = verifySignature(this.secrets, res.headers.get(HEADERS.timestamp), res.headers.get(HEADERS.signature), raw).ok;
    let parsed: ResponseEnvelope<R>;
    try {
      parsed = JSON.parse(raw) as ResponseEnvelope<R>;
    } catch {
      throw new Error(`${op} answered ${res.status} with a body that is not JSON: ${raw.slice(0, 200)}`);
    }
    return { status: res.status, body: parsed, signed, raw };
  }
}
