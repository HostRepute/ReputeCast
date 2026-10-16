import type { IExecuteFunctions, ILoadOptionsFunctions } from 'n8n-workflow';
import { NodeApiError } from 'n8n-workflow';
import { describe, expect, it } from 'vitest';

import { ReputeCast } from '../nodes/ReputeCast/ReputeCast.node';
import { BASE, WS, context, mockServer, problem } from './fakes.mts';

const node = new ReputeCast();
const run = (ctx: unknown) => node.execute.call(ctx as IExecuteFunctions);
const contactId = '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e01';
const beta = '0192a4c1-0000-7000-8000-00000000b3a3';
const upserted = {
	contactId,
	email: 'priya.n@example.com',
	created: true,
	lists: [{ id: beta, name: 'Beta Program', status: 'pending' }],
	tags: ['beta-applicant'],
	confirmationSent: true,
};

describe('the ReputeCast node', () => {
	it('offers the hub’s five actions under three resources', () => {
		const ops = node.description.properties.filter((p) => p.name === 'operation');
		expect(
			ops.flatMap((p) => (p.options ?? []).map((o) => ('action' in o ? o.action : ''))),
		).toEqual([
			'Add a tag to a contact',
			'Add a contact to a list',
			'Create or update a contact',
			'Record an event',
			'Send a transactional email',
		]);
		expect(node.description.credentials).toEqual([{ name: 'reputeCastApi', required: true }]);
	});

	it('creates or updates a contact with what was mapped, naming the workflow', async () => {
		const server = mockServer([
			{ method: 'POST', url: `${WS}/contacts/upsert`, body: upserted, operation: 'upsert-contact' },
		]);
		const { ctx } = context({
			server,
			workflow: { id: 'wf_2Lm8', name: 'Beta application → ReputeCast' },
			params: {
				resource: 'contact',
				operation: 'upsert',
				email: ' priya.n@example.com ',
				additionalFields: {
					firstName: 'Priya',
					language: 'es_MX',
					tags: 'beta-applicant, ',
					lists: [beta],
					consent: 'double',
					customFields: {
						field: [
							{ key: 'plan', value: 'trial' },
							{ key: '', value: 'x' },
						],
					},
				},
			},
		});
		const [out] = await run(ctx);
		expect(out[0].json).toEqual(upserted);
		expect(out[0].pairedItem).toEqual({ item: 0 });
		const call = server.calls[0];
		expect(call.credential).toBe('reputeCastApi');
		expect(call.body).toEqual({
			email: 'priya.n@example.com',
			firstName: 'Priya',
			language: 'es_MX',
			fields: { plan: 'trial' },
			tags: ['beta-applicant'],
			lists: [beta],
			consent: 'double',
		});
		expect(call.headers['X-ReputeCast-Workflow']).toBe(
			'Beta application %E2%86%92 ReputeCast; id=wf_2Lm8',
		);
		expect(call.headers['User-Agent']).toMatch(/^ReputeCast-n8n\/\d+\.\d+\.\d+$/);
		server.done();
	});

	it('learns the key’s workspace on the first run', async () => {
		const me = {
			workspace: 'acme-newsletter',
			workspaceName: 'Acme Newsletter',
			integration: 'n8n',
			key: 'n8n · Acme automations',
			triggers: [],
			actions: ['add_tag'],
			permissions: ['subscribers:edit'],
		};
		const server = mockServer([
			{ method: 'GET', url: `${BASE}/api/v1/integration`, body: me, operation: 'get-integration' },
			{
				method: 'POST',
				url: `${WS}/contacts/tags`,
				body: { ...upserted, created: false, lists: [], tags: ['customer'] },
				operation: 'tag-contact',
			},
		]);
		const { ctx } = context({
			server,
			credentials: { workspace: '' },
			params: {
				resource: 'contact',
				operation: 'addTag',
				email: 'dmitri@fabrikam.example',
				tags: 'customer',
			},
		});
		await run(ctx);
		expect(server.calls.map((c) => c.url)).toEqual([
			`${BASE}/api/v1/integration`,
			`${WS}/contacts/tags`,
		]);
		server.done();
	});

	it('learns it again when the workspace’s address changed', async () => {
		const me = {
			workspace: 'acme-news',
			workspaceName: 'Acme Newsletter',
			integration: 'n8n',
			key: 'n8n · Acme automations',
			triggers: [],
			actions: [],
			permissions: [],
		};
		const server = mockServer([
			{
				method: 'POST',
				url: `${WS}/contacts/tags`,
				status: 404,
				body: problem(404, 'workspace_not_found', 'Workspace not found.'),
				operation: 'tag-contact',
			},
			{ method: 'GET', url: `${BASE}/api/v1/integration`, body: me, operation: 'get-integration' },
			{
				method: 'POST',
				url: `${BASE}/api/v1/workspaces/acme-news/contacts/tags`,
				body: { ...upserted, lists: [], tags: ['customer'] },
				operation: 'tag-contact',
			},
		]);
		const { ctx } = context({
			server,
			params: {
				resource: 'contact',
				operation: 'addTag',
				email: 'a@example.com',
				tags: 'customer',
			},
		});
		const [out] = await run(ctx);
		expect(out[0].json.tags).toEqual(['customer']);
		server.done();
	});

	it('keeps consent: an item for someone who left the list says so, and the workflow goes on', async () => {
		const detail =
			"rob@example.net left Beta Program on 2 Oct. ReputeCast doesn't add people back to a list they left.";
		const server = mockServer([
			{
				method: 'POST',
				url: `${WS}/lists/${beta}/contacts`,
				status: 409,
				body: problem(409, 'unsubscribed', detail, {
					meta: { list: 'Beta Program', listId: beta, leftAt: '2026-10-02T13:41:00Z' },
				}),
				operation: 'add-contact-to-list',
			},
		]);
		const { ctx } = context({
			server,
			params: {
				resource: 'contact',
				operation: 'addToList',
				listId: beta,
				email: 'rob@example.net',
				consent: 'single',
			},
		});
		const [out] = await run(ctx);
		expect(out[0].json).toEqual({
			skipped: true,
			code: 'unsubscribed',
			reason: detail,
			list: 'Beta Program',
			listId: beta,
			leftAt: '2026-10-02T13:41:00Z',
		});
		expect(server.calls[0].body).toEqual({ email: 'rob@example.net', consent: 'single' });
		server.done();
	});

	it('fails in ReputeCast’s words, or carries on when the node is set to', async () => {
		const detail =
			'This key belongs to n8n and works only for what n8n can do. Change what n8n can do in Settings › Integrations.';
		const routes = () => [
			{
				method: 'POST',
				url: `${WS}/automations/events`,
				status: 403,
				body: problem(403, 'integration_operation_denied', detail),
				operation: 'send-automation-event',
			},
		];
		const params = {
			resource: 'event',
			operation: 'record',
			eventName: 'workspace.created',
			email: 'a@example.com',
			properties: {},
		};
		const s1 = mockServer(routes());
		const failing = run(context({ server: s1, params }).ctx);
		await expect(failing).rejects.toBeInstanceOf(NodeApiError);
		await expect(failing).rejects.toMatchObject({ message: detail, httpCode: '403' });
		const s2 = mockServer(routes());
		const [out] = await run(context({ server: s2, params, continueOnFail: true }).ctx);
		expect(out[0].json).toEqual({ error: detail });
		s1.done();
		s2.done();
	});

	it('records an event with its properties', async () => {
		const server = mockServer([
			{
				method: 'POST',
				url: `${WS}/automations/events`,
				status: 202,
				body: { accepted: true, contactId, flows: 1 },
				operation: 'send-automation-event',
			},
		]);
		const { ctx } = context({
			server,
			params: {
				resource: 'event',
				operation: 'record',
				eventName: 'workspace.created',
				email: 'priya.n@example.com',
				properties: { property: [{ name: 'plan', value: 'Team' }] },
			},
		});
		const [out] = await run(ctx);
		expect(out[0].json.flows).toBe(1);
		expect(server.calls[0].body).toEqual({
			event: 'workspace.created',
			email: 'priya.n@example.com',
			properties: { plan: 'Team' },
		});
		server.done();
	});

	it('sends a transactional email with typed variables, a language and an idempotency key', async () => {
		const server = mockServer([
			{
				method: 'POST',
				url: `${WS}/transactional/messages`,
				status: 202,
				body: { id: '0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44', status: 'queued' },
				operation: 'send-transactional-message',
			},
		]);
		const schema = [
			{
				id: 'first_name',
				displayName: 'first_name',
				required: false,
				defaultMatch: false,
				display: true,
				type: 'string',
			},
			{
				id: 'seats',
				displayName: 'seats',
				required: false,
				defaultMatch: false,
				display: true,
				type: 'number',
			},
			{
				id: 'items',
				displayName: 'items',
				required: false,
				defaultMatch: false,
				display: true,
				type: 'array',
			},
		];
		const { ctx } = context({
			server,
			params: {
				resource: 'transactionalEmail',
				operation: 'send',
				templateKey: 'payment-failed',
				toEmail: 'finance@initech.example',
				variables: {
					mappingMode: 'defineBelow',
					value: { first_name: 'Priya', seats: '12', items: '["Team plan"]' },
					matchingColumns: [],
					schema,
				},
				additionalFields: { toName: 'Priya', language: 'es', idempotencyKey: 'evt_1QKx' },
			},
		});
		const [out] = await run(ctx);
		expect(out[0].json.status).toBe('queued');
		const call = server.calls[0];
		expect(call.body).toEqual({
			template: 'payment-failed',
			to: { email: 'finance@initech.example', name: 'Priya' },
			data: { first_name: 'Priya', seats: 12, items: ['Team plan'] },
			language: 'es',
		});
		expect(call.headers['Idempotency-Key']).toBe('n8n-wf_2Lm8-evt_1QKx');
		server.done();
	});

	it('maps the input item’s fields to variables automatically', async () => {
		const server = mockServer([
			{
				method: 'POST',
				url: `${WS}/transactional/messages`,
				status: 202,
				body: { id: '0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c45', status: 'queued' },
				operation: 'send-transactional-message',
			},
		]);
		const schema = [
			{
				id: 'amount_due',
				displayName: 'amount_due',
				required: true,
				defaultMatch: false,
				display: true,
				type: 'string',
			},
			{
				id: 'update_url',
				displayName: 'update_url',
				required: true,
				defaultMatch: false,
				display: true,
				type: 'url',
			},
		];
		const { ctx } = context({
			server,
			items: [
				{ amount_due: '$49.00', update_url: 'https://billing.acme.io/u/8812', unrelated: true },
			],
			params: {
				resource: 'transactionalEmail',
				operation: 'send',
				templateKey: 'payment-failed',
				toEmail: 'finance@initech.example',
				variables: { mappingMode: 'autoMapInputData', value: null, matchingColumns: [], schema },
				additionalFields: {},
			},
		});
		await run(ctx);
		expect(server.calls[0].body).toEqual({
			template: 'payment-failed',
			to: { email: 'finance@initech.example' },
			data: { amount_due: '$49.00', update_url: 'https://billing.acme.io/u/8812' },
		});
		expect(server.calls[0].headers['Idempotency-Key']).toBeUndefined();
		server.done();
	});

	it('names the variables a run is missing', async () => {
		const server = mockServer([
			{
				method: 'POST',
				url: `${WS}/transactional/messages`,
				status: 422,
				body: problem(422, 'missing_variables', 'Some variables are missing.', {
					missing: ['amount_due'],
				}),
				operation: 'send-transactional-message',
			},
		]);
		const { ctx } = context({
			server,
			params: {
				resource: 'transactionalEmail',
				operation: 'send',
				templateKey: 'payment-failed',
				toEmail: 'a@example.com',
				variables: { mappingMode: 'defineBelow', value: {}, matchingColumns: [], schema: [] },
				additionalFields: {},
			},
		});
		await expect(run(ctx)).rejects.toMatchObject({
			message: 'Some variables are missing. Missing variables: amount_due.',
		});
		server.done();
	});
});

describe('dropdowns', () => {
	const load = (fn: (this: ILoadOptionsFunctions) => Promise<unknown>, ctx: unknown) =>
		fn.call(ctx as ILoadOptionsFunctions);

	it('lists lists, custom fields and templates', async () => {
		const server = mockServer([
			{
				method: 'GET',
				url: `${WS}/integration/lists`,
				body: { items: [{ id: beta, name: 'Beta Program', doubleOptIn: true }] },
				operation: 'list-integration-lists',
			},
			{
				method: 'GET',
				url: `${WS}/integration/fields`,
				body: { items: [{ key: 'plan', label: 'Plan', type: 'select' }] },
				operation: 'list-integration-fields',
			},
			{
				method: 'GET',
				url: `${WS}/integration/templates`,
				body: {
					items: [{ key: 'payment-failed', name: 'Payment failed', version: 1, variables: [] }],
				},
				operation: 'list-integration-templates',
			},
		]);
		const { ctx } = context({ server });
		expect(await load(node.methods.loadOptions.getLists, ctx)).toEqual([
			{ name: 'Beta Program (double opt-in)', value: beta },
		]);
		expect(await load(node.methods.loadOptions.getFields, ctx)).toEqual([
			{ name: 'Plan', value: 'plan', description: 'select field' },
		]);
		expect(await load(node.methods.loadOptions.getTemplates, ctx)).toEqual([
			{
				name: 'Payment failed (payment-failed)',
				value: 'payment-failed',
				description: 'Live version 1',
			},
		]);
		server.done();
	});

	it('turns the chosen template’s variables into mapping fields', async () => {
		const server = mockServer([
			{
				method: 'GET',
				url: `${WS}/integration/templates`,
				body: {
					items: [
						{
							key: 'payment-failed',
							name: 'Payment failed',
							version: 1,
							variables: [
								{ name: 'amount_due', type: 'text', required: true, sensitive: false },
								{ name: 'update_url', type: 'url', required: true, sensitive: true },
								{ name: 'seats', type: 'number', required: false, sensitive: false },
							],
						},
					],
				},
				operation: 'list-integration-templates',
			},
		]);
		const { ctx } = context({ server, params: { templateKey: 'payment-failed' } });
		const out = (await load(node.methods.resourceMapping.getTemplateVariables, ctx)) as {
			fields: Array<{ id: string; required: boolean; type: string; displayName: string }>;
		};
		expect(out.fields.map((f) => [f.id, f.required, f.type])).toEqual([
			['amount_due', true, 'string'],
			['update_url', true, 'url'],
			['seats', false, 'number'],
		]);
		expect(out.fields[1].displayName).toContain('sensitive');
		server.done();
	});
});
