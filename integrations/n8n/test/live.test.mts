// End to end against a running ReputeCast, skipped unless REPUTECAST_URL
// and REPUTECAST_KEY name one (README › Test). The key needs every trigger
// and action on; for the trigger test the instance must be allowed to call
// this machine (webhooks.allow_private_networks).
//
//   REPUTECAST_URL=http://localhost:8080 REPUTECAST_KEY=rc_live_… npm test

import { createServer } from 'node:http';
import type { AddressInfo } from 'node:net';

import type {
	IDataObject,
	IExecuteFunctions,
	IHookFunctions,
	IHttpRequestOptions,
	ILoadOptionsFunctions,
	IWebhookFunctions,
} from 'n8n-workflow';
import { describe, expect, it } from 'vitest';

import { ReputeCastApi } from '../credentials/ReputeCastApi.credentials';
import { ReputeCast } from '../nodes/ReputeCast/ReputeCast.node';
import { ReputeCastTrigger } from '../nodes/ReputeCastTrigger/ReputeCastTrigger.node';

const url = process.env.REPUTECAST_URL ?? '';
const key = process.env.REPUTECAST_KEY ?? '';

// What n8n does with the credential: its preAuthentication, then the key
// on every request.
const credentials: IDataObject = { instanceUrl: url, apiKey: key, workspace: '' };
const http = async (options: IHttpRequestOptions) => {
	const qs = options.qs ? `?${new URLSearchParams(options.qs as Record<string, string>)}` : '';
	const res = await fetch(`${options.url}${qs}`, {
		method: options.method ?? 'GET',
		headers: { 'Content-Type': 'application/json', ...(options.headers as Record<string, string>) },
		body: options.body === undefined ? undefined : JSON.stringify(options.body),
	});
	const text = await res.text();
	let body: unknown = text;
	try {
		body = text ? JSON.parse(text) : '';
	} catch {
		// not JSON
	}
	return { statusCode: res.status, body, headers: Object.fromEntries(res.headers) };
};
const withKey = async (_type: string, options: IHttpRequestOptions) =>
	http({ ...options, headers: { ...(options.headers ?? {}), Authorization: `Bearer ${key}` } });

const ctx = (params: IDataObject, extra: IDataObject = {}) => ({
	getNode: () => ({
		id: 'live',
		name: 'ReputeCast',
		type: 'n8n-nodes-reputecast.reputeCast',
		typeVersion: 1,
		position: [0, 0],
		parameters: {},
	}),
	getWorkflow: () => ({ id: 'wf_live', name: 'Live test → ReputeCast', active: true }),
	getCredentials: async () => credentials,
	getInputData: () => [{ json: {} }],
	getNodeParameter: (name: string, a?: unknown, b?: unknown) =>
		name in params ? params[name] : typeof a === 'number' ? b : a,
	continueOnFail: () => false,
	helpers: {
		httpRequestWithAuthentication: withKey,
		returnJsonArray: (items: IDataObject[]) => items.map((json) => ({ json })),
	},
	logger: { warn: () => {}, info: () => {}, debug: () => {}, error: () => {} },
	...extra,
});

describe.skipIf(!url || !key)('a live instance', () => {
	const node = new ReputeCast();
	const trigger = new ReputeCastTrigger();

	it('learns the workspace with the credential', async () => {
		const out = await new ReputeCastApi().preAuthentication.call(
			{ helpers: { httpRequest: http } } as never,
			{ instanceUrl: url, apiKey: key },
		);
		expect(out.workspace).toBeTruthy();
		credentials.workspace = out.workspace;
	});

	it('creates and tags a contact, and records an event', async () => {
		const lists = await node.methods.loadOptions.getLists.call(
			ctx({}) as unknown as ILoadOptionsFunctions,
		);
		const email = `n8n-live+${Date.now()}@example.com`;
		const [upserted] = await node.execute.call(
			ctx({
				resource: 'contact',
				operation: 'upsert',
				email,
				additionalFields: {
					firstName: 'Live',
					tags: 'live-test',
					lists: [lists[0].value],
					consent: 'single',
				},
			}) as unknown as IExecuteFunctions,
		);
		expect(upserted[0].json.created).toBe(true);
		const [tagged] = await node.execute.call(
			ctx({
				resource: 'contact',
				operation: 'addTag',
				email,
				tags: 'customer',
			}) as unknown as IExecuteFunctions,
		);
		expect(tagged[0].json.tags).toEqual(['customer', 'live-test']);
		const [event] = await node.execute.call(
			ctx({
				resource: 'event',
				operation: 'record',
				eventName: 'live.test',
				email,
				properties: { property: [{ name: 'plan', value: 'Team' }] },
			}) as unknown as IExecuteFunctions,
		);
		expect(event[0].json.accepted).toBe(true);
	});

	it('sends a published transactional template', async () => {
		const templates = await node.methods.loadOptions.getTemplates.call(
			ctx({}) as unknown as ILoadOptionsFunctions,
		);
		if (!templates.length) {
			return;
		}
		const templateKey = String(templates[0].value);
		const { fields } = await node.methods.resourceMapping.getTemplateVariables.call(
			ctx({ templateKey }) as unknown as ILoadOptionsFunctions,
		);
		const value: IDataObject = {};
		for (const f of fields) {
			value[f.id] =
				f.type === 'number'
					? 3
					: f.type === 'boolean'
						? true
						: f.type === 'url'
							? 'https://example.com/x'
							: f.type === 'array'
								? '["one"]'
								: 'Live';
		}
		const [out] = await node.execute.call(
			ctx({
				resource: 'transactionalEmail',
				operation: 'send',
				templateKey,
				toEmail: 'n8n-live@example.com',
				variables: { mappingMode: 'defineBelow', value, matchingColumns: [], schema: fields },
				additionalFields: { idempotencyKey: `live-${Date.now()}` },
			}) as unknown as IExecuteFunctions,
		);
		expect(['queued', 'scheduled', 'not_delivered']).toContain(out[0].json.status);
	});

	it('turns a trigger on, receives a signed delivery, and turns it off', async () => {
		const deliveries: Array<{ headers: IDataObject; raw: Buffer }> = [];
		const server = createServer((req, res) => {
			const chunks: Buffer[] = [];
			req.on('data', (c: Buffer) => chunks.push(c));
			req.on('end', () => {
				res.end('ok');
				deliveries.push({ headers: req.headers as IDataObject, raw: Buffer.concat(chunks) });
			});
		});
		await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
		const webhookUrl = `http://127.0.0.1:${(server.address() as AddressInfo).port}/webhook/live/reputecast`;
		const staticData: IDataObject = {};
		const hookCtx = ctx(
			{ event: 'contact.subscribed' },
			{ getWorkflowStaticData: () => staticData, getNodeWebhookUrl: () => webhookUrl },
		) as unknown as IHookFunctions;
		const hooks = trigger.webhookMethods.default;
		try {
			expect(await hooks.checkExists.call(hookCtx)).toBe(false);
			expect(await hooks.create.call(hookCtx)).toBe(true);
			expect(await hooks.checkExists.call(hookCtx)).toBe(true);
			const lists = await new ReputeCast().methods.loadOptions.getLists.call(
				ctx({}) as unknown as ILoadOptionsFunctions,
			);
			const email = `n8n-hook+${Date.now()}@example.com`;
			await node.execute.call(
				ctx({
					resource: 'contact',
					operation: 'upsert',
					email,
					additionalFields: { lists: [lists[0].value], consent: 'single' },
				}) as unknown as IExecuteFunctions,
			);
			const deadline = Date.now() + 60000;
			let delivery: { headers: IDataObject; raw: Buffer } | undefined;
			while (!delivery && Date.now() < deadline) {
				delivery = deliveries.find(
					(d) =>
						(JSON.parse(d.raw.toString()) as { data?: { email?: string } }).data?.email === email,
				);
				if (!delivery) await new Promise((resolve) => setTimeout(resolve, 500));
			}
			expect(delivery, 'the contact.subscribed delivery arrived').toBeDefined();
			const body = JSON.parse(delivery!.raw.toString()) as IDataObject;
			const out = await trigger.webhook.call(
				ctx(
					{ event: 'contact.subscribed' },
					{
						getWorkflowStaticData: () => staticData,
						getHeaderData: () => delivery!.headers,
						getBodyData: () => body,
						getRequestObject: () => ({ rawBody: delivery!.raw }),
					},
				) as unknown as IWebhookFunctions,
			);
			expect(out.workflowData?.[0][0].json.email).toBe(email);
		} finally {
			expect(await hooks.delete.call(hookCtx)).toBe(true);
			server.close();
		}
	}, 90000);
});
