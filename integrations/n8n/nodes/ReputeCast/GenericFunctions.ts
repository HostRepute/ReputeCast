import { createHmac, timingSafeEqual } from 'crypto';
import type {
	IDataObject,
	IExecuteFunctions,
	IHookFunctions,
	IHttpRequestMethods,
	IHttpRequestOptions,
	ILoadOptionsFunctions,
	IWebhookFunctions,
	JsonObject,
} from 'n8n-workflow';
import { NodeApiError, NodeOperationError } from 'n8n-workflow';

import { version } from '../../package.json';

/** How ReputeCast's call log names this package: "n8n, ReputeCast node 1.1.0". */
export const USER_AGENT = `ReputeCast-n8n/${version}`;

export const CREDENTIAL = 'reputeCastApi';

export type ReputeCastContext =
	| IExecuteFunctions
	| ILoadOptionsFunctions
	| IHookFunctions
	| IWebhookFunctions;

/**
 * The instance's address as typed in the credential: a scheme is added
 * when missing, and a pasted API path or trailing slash is dropped.
 */
export function instanceBase(raw: unknown): string {
	let url = String(raw ?? '').trim();
	if (!url) {
		return '';
	}
	if (!/^https?:\/\//i.test(url)) {
		url = `https://${url}`;
	}
	return url
		.replace(/\/+$/, '')
		.replace(/\/api\/v1$/i, '')
		.replace(/\/+$/, '');
}

/**
 * X-ReputeCast-Workflow: "<name>; id=<id>". Header values are ASCII, so
 * the name's characters outside printable ASCII, and the four the header
 * uses itself, are percent-encoded (UTF-8); the rest stay as typed.
 */
export function workflowHeader(workflow: { name?: string; id?: string | number }): string {
	const parts: string[] = [];
	const name = Array.from(
		String(workflow.name ?? '')
			.replace(/\s+/g, ' ')
			.trim(),
	)
		.slice(0, 200)
		.map((ch) => {
			const c = ch.codePointAt(0) ?? 0;
			if (c >= 0x20 && c <= 0x7e && !'%;="'.includes(ch)) {
				return ch;
			}
			try {
				return encodeURIComponent(ch);
			} catch {
				return '';
			}
		})
		.join('');
	if (name) {
		parts.push(name);
	}
	const id = String(workflow.id ?? '')
		.replace(/[^A-Za-z0-9._:-]/g, '')
		.slice(0, 120);
	if (id) {
		parts.push(`id=${id}`);
	}
	return parts.join('; ');
}

interface Problem {
	status?: number;
	title?: string;
	detail?: string;
	code?: string;
	errors?: Array<{ location?: string; message?: string }>;
	missing?: string[];
	meta?: IDataObject;
}

function sentence(s: unknown): string {
	const t = String(s ?? '').trim();
	if (!t) {
		return '';
	}
	return /[.!?…]$/.test(t) ? t : `${t}.`;
}

function asProblem(body: unknown): Problem {
	if (body && typeof body === 'object') {
		return body as Problem;
	}
	if (typeof body === 'string' && body.trim().startsWith('{')) {
		try {
			return JSON.parse(body) as Problem;
		} catch {
			return {};
		}
	}
	return {};
}

/**
 * The message for an RFC 9457 problem: its detail, then field errors and
 * missing template variables the detail doesn't name.
 */
export function messageOf(status: number, body: unknown): string {
	const p = asProblem(body);
	const detail = sentence(p.detail) || sentence(p.title) || `ReputeCast answered HTTP ${status}.`;
	const parts = [detail];
	const fields = (Array.isArray(p.errors) ? p.errors : [])
		.map((e) => {
			const msg = String(e?.message ?? '').trim();
			if (!msg || detail.includes(msg)) {
				return '';
			}
			const where = String(e?.location ?? '').replace(/^body\./, '');
			return where && where !== 'body' ? `${where}: ${msg}` : msg;
		})
		.filter(Boolean);
	if (fields.length) {
		parts.push(sentence(fields.join('; ')));
	}
	const missing = (Array.isArray(p.missing) ? p.missing : []).filter((m) => !detail.includes(m));
	if (missing.length) {
		parts.push(`Missing variables: ${missing.join(', ')}.`);
	}
	return parts.join(' ');
}

// Answers that skip one record without breaking the workflow: an action
// never adds someone back to a list they left or adds a suppressed address
// (409), and transactional email isn't sent to a suppressed recipient (422).
const SKIPS = new Set(['409 unsubscribed', '409 suppressed', '422 recipient_suppressed']);

/** A record ReputeCast skipped to keep consent: the item says so instead of failing. */
export class Skipped {
	constructor(
		readonly code: string,
		readonly reason: string,
		readonly meta: IDataObject,
	) {}

	toJSON(): IDataObject {
		return { skipped: true, code: this.code, reason: this.reason, ...this.meta };
	}
}

interface RequestOptions {
	body?: IDataObject;
	headers?: IDataObject;
	/** Answers that aren't errors for this call (a webhook already gone: 404). */
	allow?: number[];
}

async function send(
	ctx: ReputeCastContext,
	method: IHttpRequestMethods,
	url: string,
	opts: RequestOptions,
): Promise<{ statusCode: number; body: unknown }> {
	const headers: IDataObject = {
		Accept: 'application/json',
		'User-Agent': USER_AGENT,
		...(opts.headers ?? {}),
	};
	const workflow = ctx.getWorkflow();
	const wf = workflowHeader({ name: workflow.name, id: workflow.id });
	if (wf) {
		headers['X-ReputeCast-Workflow'] = wf;
	}
	const options: IHttpRequestOptions = {
		method,
		url,
		headers,
		json: true,
		returnFullResponse: true,
		ignoreHttpStatusErrors: true,
	};
	if (opts.body !== undefined) {
		options.body = opts.body;
	}
	const response = (await ctx.helpers.httpRequestWithAuthentication.call(
		ctx,
		CREDENTIAL,
		options,
	)) as {
		statusCode: number;
		body: unknown;
	};
	return { statusCode: response.statusCode, body: response.body };
}

/**
 * Calls ReputeCast's API with the credential's key. path starts with "/"
 * and may hold {workspace}: the key's workspace, which the credential
 * learns once (GET /api/v1/integration) and keeps.
 */
export async function reputeCastApiRequest(
	this: ReputeCastContext,
	method: IHttpRequestMethods,
	path: string,
	opts: RequestOptions = {},
): Promise<unknown> {
	const credentials = await this.getCredentials(CREDENTIAL);
	const base = instanceBase(credentials.instanceUrl);
	let workspace = String(credentials.workspace ?? '');
	const urlFor = (ws: string) =>
		`${base}/api/v1${path.replace('{workspace}', encodeURIComponent(ws))}`;
	const learnWorkspace = async (): Promise<string> => {
		const me = await send(this, 'GET', `${base}/api/v1/integration`, {});
		fail(this, me.statusCode, me.body);
		return String((me.body as IDataObject).workspace ?? '');
	};
	if (path.includes('{workspace}') && !workspace) {
		workspace = await learnWorkspace();
	}
	let res = await send(this, method, urlFor(workspace), opts);
	// The workspace's address changed since the credential learned it.
	if (
		res.statusCode === 404 &&
		asProblem(res.body).code === 'workspace_not_found' &&
		path.includes('{workspace}')
	) {
		workspace = await learnWorkspace();
		res = await send(this, method, urlFor(workspace), opts);
	}
	if (opts.allow?.includes(res.statusCode)) {
		return res.body;
	}
	const p = asProblem(res.body);
	if (SKIPS.has(`${res.statusCode} ${p.code ?? ''}`)) {
		return new Skipped(String(p.code), messageOf(res.statusCode, p), p.meta ?? {});
	}
	fail(this, res.statusCode, res.body);
	return res.body;
}

/** Throws ReputeCast's problem as n8n's API error, in ReputeCast's words. */
function fail(ctx: ReputeCastContext, statusCode: number, body: unknown): void {
	if (statusCode < 400) {
		return;
	}
	const p = asProblem(body);
	throw new NodeApiError(
		ctx.getNode(),
		(Object.keys(p).length ? p : { status: statusCode }) as JsonObject,
		{
			message: messageOf(statusCode, p),
			description: p.code
				? `ReputeCast answered HTTP ${statusCode} ${p.code}.`
				: `ReputeCast answered HTTP ${statusCode}.`,
			httpCode: String(statusCode),
		},
	);
}

/** An error as n8n shows it: ReputeCast's own stay as they are, the rest are wrapped. */
export function asNodeError(
	ctx: ReputeCastContext,
	error: unknown,
	itemIndex: number,
): NodeApiError | NodeOperationError {
	if (error instanceof NodeApiError || error instanceof NodeOperationError) {
		return error;
	}
	return new NodeApiError(ctx.getNode(), error as JsonObject, { itemIndex });
}

/** A delivery as a trigger's output: data's fields, then the event's own. */
export function flatten(env: IDataObject): IDataObject {
	const data =
		env.data && typeof env.data === 'object' && !Array.isArray(env.data)
			? (env.data as IDataObject)
			: {};
	return {
		...data,
		id: env.id,
		event: env.type,
		created_at: env.created_at,
		workspace: env.workspace,
	};
}

export const MAX_SIGNATURE_AGE_SECONDS = 5 * 60;

/**
 * Checks a delivery: X-ReputeCast-Signature is the hex HMAC-SHA256 of
 * "{timestamp}.{raw body}" with the webhook's secret, X-ReputeCast-Timestamp
 * the Unix time, at most 5 minutes old.
 */
export function verifySignature(input: {
	secret: string;
	body: string | Buffer;
	signature?: string;
	timestamp?: string;
	now?: number;
}): { ok: boolean; why?: string } {
	const { secret, body, signature, timestamp } = input;
	const now = input.now ?? Math.floor(Date.now() / 1000);
	if (!signature || !timestamp) {
		return { ok: false, why: 'the delivery isn’t signed' };
	}
	const ts = Number.parseInt(timestamp, 10);
	if (!Number.isFinite(ts) || Math.abs(now - ts) > MAX_SIGNATURE_AGE_SECONDS) {
		return { ok: false, why: 'the delivery is more than 5 minutes old' };
	}
	const expected = createHmac('sha256', secret).update(`${ts}.`).update(body).digest('hex');
	const ok = signature
		.split(',')
		.map((s) => s.trim())
		.some(
			(s) => s.length === expected.length && timingSafeEqual(Buffer.from(s), Buffer.from(expected)),
		);
	return ok
		? { ok: true }
		: { ok: false, why: 'the signature doesn’t match this trigger’s webhook secret' };
}
