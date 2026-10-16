// Fakes of the n8n contexts the nodes run in, and a mock ReputeCast whose
// every call is checked against web/openapi.json.

import type { IDataObject, IHttpRequestOptions, INode } from 'n8n-workflow';
import { expect } from 'vitest';

import { checkRequest, checkResponse } from '../../contract/openapi-contract.mjs';

export const BASE = 'https://reputecast.acme.io';
export const WS = `${BASE}/api/v1/workspaces/acme-newsletter`;
export const KEY = `rc_live_9a40${'0'.repeat(56)}e1c3`;

// Fields another workstream adds to the API in this release (X05 adds
// `language` to the transactional send API).
const pending = { 'send-transactional-message': ['language'] };

export interface Call {
	method: string;
	url: string;
	body?: unknown;
	headers: IDataObject;
	credential: string;
}

interface Route {
	method: string;
	url: string;
	status?: number;
	body?: unknown;
	operation?: string;
}

/** A mock ReputeCast: answers queued routes in order, and records the calls. */
export function mockServer(routes: Route[]) {
	const queue = [...routes];
	const calls: Call[] = [];
	const problems: string[] = [];
	for (const r of routes) {
		if (r.operation) {
			const res = checkResponse(r.operation, r.status ?? 200, r.body);
			if (!res.ok) problems.push(`mocked answer of ${r.operation}: ${res.errors.join('; ')}`);
		}
	}
	const httpRequestWithAuthentication = async function (
		credential: string,
		options: IHttpRequestOptions,
	) {
		const method = String(options.method ?? 'GET');
		const call: Call = {
			method,
			url: options.url,
			body: options.body,
			headers: (options.headers ?? {}) as IDataObject,
			credential,
		};
		calls.push(call);
		const i = queue.findIndex((r) => r.method === method && r.url === options.url);
		if (i < 0) throw new Error(`unexpected ${method} ${options.url}`);
		const [route] = queue.splice(i, 1);
		return { statusCode: route.status ?? 200, body: route.body ?? '', headers: {} };
	};
	const done = () => {
		expect(queue.map((r) => `${r.method} ${r.url}`)).toEqual([]);
		for (const c of calls) {
			const res = checkRequest(
				{
					method: c.method,
					url: c.url,
					body: c.body,
					headers: c.headers as Record<string, string>,
				},
				{ pending },
			);
			if (!res.ok) problems.push(`${c.method} ${c.url}: ${res.errors.join('; ')}`);
		}
		expect(problems).toEqual([]);
	};
	return { calls, httpRequestWithAuthentication, done };
}

export const node: INode = {
	id: 'node-1',
	name: 'ReputeCast',
	type: 'n8n-nodes-reputecast.reputeCast',
	typeVersion: 1,
	position: [0, 0],
	parameters: {},
};

export const workflow = { id: 'wf_2Lm8', name: 'Acme Cloud: workspace created', active: true };

interface ContextOptions {
	params?: IDataObject | IDataObject[];
	items?: IDataObject[];
	credentials?: IDataObject;
	server: ReturnType<typeof mockServer>;
	continueOnFail?: boolean;
	staticData?: IDataObject;
	webhookUrl?: string;
	headers?: IDataObject;
	body?: IDataObject;
	rawBody?: Buffer;
	workflow?: { id?: string; name?: string };
}

/** One fake context for every function type the nodes use. */
export function context(opts: ContextOptions) {
	const params = opts.params ?? {};
	const paramsFor = (i: number): IDataObject =>
		Array.isArray(params) ? (params[i] ?? params[0]) : params;
	const response = { statusCode: 200, sent: undefined as unknown };
	const warnings: string[] = [];
	const staticData = opts.staticData ?? {};
	const ctx = {
		getNode: () => node,
		getWorkflow: () => opts.workflow ?? workflow,
		getCredentials: async () => ({
			instanceUrl: BASE,
			apiKey: KEY,
			workspace: 'acme-newsletter',
			...(opts.credentials ?? {}),
		}),
		getInputData: () => (opts.items ?? [{}]).map((json) => ({ json })),
		getNodeParameter: (name: string, a?: unknown, b?: unknown) => {
			// Execute functions: (name, itemIndex, fallback); the others: (name, fallback).
			const index = typeof a === 'number' ? a : 0;
			const fallback = typeof a === 'number' ? b : a;
			const p = paramsFor(index);
			return name in p ? p[name] : fallback;
		},
		getCurrentNodeParameter: (name: string) => paramsFor(0)[name],
		continueOnFail: () => opts.continueOnFail ?? false,
		getWorkflowStaticData: () => staticData,
		getNodeWebhookUrl: () => opts.webhookUrl,
		getHeaderData: () => opts.headers ?? {},
		getBodyData: () => opts.body ?? {},
		getRequestObject: () => ({ rawBody: opts.rawBody }),
		getResponseObject: () => ({
			status(code: number) {
				response.statusCode = code;
				return {
					json(v: unknown) {
						response.sent = v;
					},
				};
			},
		}),
		logger: {
			warn: (m: string) => warnings.push(m),
			info: () => {},
			debug: () => {},
			error: () => {},
		},
		helpers: {
			httpRequestWithAuthentication: opts.server.httpRequestWithAuthentication,
			returnJsonArray: (items: IDataObject[]) => items.map((json) => ({ json })),
		},
	};
	return { ctx, response, warnings, staticData };
}

export const problem = (status: number, code: string, detail: string, extra: IDataObject = {}) => ({
	title:
		(
			{
				401: 'Unauthorized',
				403: 'Forbidden',
				404: 'Not Found',
				409: 'Conflict',
				422: 'Unprocessable Entity',
			} as Record<number, string>
		)[status] ?? 'Error',
	status,
	detail,
	code,
	...extra,
});
