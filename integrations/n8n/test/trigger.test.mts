import { createHmac } from 'node:crypto';

import type {
	IDataObject,
	IHookFunctions,
	ILoadOptionsFunctions,
	IWebhookFunctions,
} from 'n8n-workflow';
import { describe, expect, it } from 'vitest';

import { ReputeCastTrigger } from '../nodes/ReputeCastTrigger/ReputeCastTrigger.node';
import { WS, context, mockServer, problem } from './fakes.mts';

const trigger = new ReputeCastTrigger();
const hooks = trigger.webhookMethods.default;
const hookId = '0192b7e2-11aa-7c1e-9a40-3f1b2c8d5e99';
const secret = 'whsec_3f9a1c2b7e4d4c1a9b8e7f6a5d4c3b2a';
const webhookUrl = 'https://n8n.acme.io/webhook/7c1e4b2a-91d3-4f0e-8a6b-2c5d7e9f1a3b/reputecast';
const hook = (cx: unknown) => cx as IHookFunctions;

const created = {
	id: hookId,
	event: 'contact.unsubscribed',
	url: webhookUrl,
	secret,
	signatureHeader: 'X-ReputeCast-Signature',
	timestampHeader: 'X-ReputeCast-Timestamp',
	createdAt: '2026-09-29T11:58:00Z',
};

describe('the ReputeCast Trigger node', () => {
	it('is a trigger with the hub’s six events', () => {
		const d = trigger.description;
		expect(d.inputs).toEqual([]);
		expect(d.webhooks?.[0].path).toBe('reputecast');
		const event = d.properties.find((p) => p.name === 'event');
		expect((event?.options ?? []).map((o) => ('value' in o ? o.value : ''))).toEqual([
			'campaign.completed',
			'contact.subscribed',
			'contact.unsubscribed',
			'contact.updated',
			'form.submitted',
			'message.bounced',
		]);
	});

	it('creates its webhook when the workflow is turned on, and keeps the secret', async () => {
		const server = mockServer([
			{
				method: 'POST',
				url: `${WS}/integration/hooks`,
				status: 201,
				body: created,
				operation: 'subscribe-integration-hook',
			},
		]);
		const { ctx, staticData } = context({
			server,
			webhookUrl,
			params: { event: 'contact.unsubscribed' },
			workflow: { id: 'wf_9', name: 'Unsubscribes → Acme Cloud CRM' },
		});
		expect(await hooks.create.call(hook(ctx))).toBe(true);
		expect(server.calls[0].body).toEqual({ event: 'contact.unsubscribed', url: webhookUrl });
		expect(server.calls[0].headers['X-ReputeCast-Workflow']).toBe(
			'Unsubscribes %E2%86%92 Acme Cloud CRM; id=wf_9',
		);
		expect(staticData).toEqual({
			webhookId: hookId,
			webhookSecret: secret,
			webhookEvent: 'contact.unsubscribed',
		});
		server.done();
	});

	it('checks the webhook still exists, for this event and address', async () => {
		const ok = mockServer([
			{
				method: 'GET',
				url: `${WS}/integration/hooks/${hookId}`,
				body: { ...created, secret: undefined },
				operation: 'get-integration-hook',
			},
		]);
		const a = context({
			server: ok,
			webhookUrl,
			params: { event: 'contact.unsubscribed' },
			staticData: { webhookId: hookId, webhookSecret: secret },
		});
		expect(await hooks.checkExists.call(hook(a.ctx))).toBe(true);
		ok.done();

		// Gone (a 410 Gone removed it): forget it, so n8n creates it again.
		const gone = mockServer([
			{
				method: 'GET',
				url: `${WS}/integration/hooks/${hookId}`,
				status: 404,
				body: problem(404, 'not_found', 'Webhook not found.'),
				operation: 'get-integration-hook',
			},
		]);
		const b = context({
			server: gone,
			webhookUrl,
			params: { event: 'contact.unsubscribed' },
			staticData: { webhookId: hookId, webhookSecret: secret },
		});
		expect(await hooks.checkExists.call(hook(b.ctx))).toBe(false);
		expect(b.staticData).toEqual({});
		gone.done();

		// Made for another event: create it again.
		const other = mockServer([
			{
				method: 'GET',
				url: `${WS}/integration/hooks/${hookId}`,
				body: { ...created, secret: undefined },
				operation: 'get-integration-hook',
			},
		]);
		const c = context({
			server: other,
			webhookUrl,
			params: { event: 'contact.updated' },
			staticData: { webhookId: hookId },
		});
		expect(await hooks.checkExists.call(hook(c.ctx))).toBe(false);
		other.done();

		// Nothing made yet: no call.
		const none = mockServer([]);
		expect(
			await hooks.checkExists.call(
				hook(context({ server: none, params: { event: 'contact.updated' } }).ctx),
			),
		).toBe(false);
		none.done();
	});

	it('removes its webhook when the workflow is turned off, even one already gone', async () => {
		const server = mockServer([
			{
				method: 'DELETE',
				url: `${WS}/integration/hooks/${hookId}`,
				status: 204,
				operation: 'unsubscribe-integration-hook',
			},
			{
				method: 'DELETE',
				url: `${WS}/integration/hooks/${hookId}`,
				status: 404,
				body: problem(404, 'not_found', 'Webhook not found.'),
				operation: 'unsubscribe-integration-hook',
			},
		]);
		const a = context({
			server,
			params: { event: 'contact.unsubscribed' },
			staticData: { webhookId: hookId, webhookSecret: secret },
		});
		expect(await hooks.delete.call(hook(a.ctx))).toBe(true);
		expect(a.staticData).toEqual({});
		const b = context({
			server,
			params: { event: 'contact.unsubscribed' },
			staticData: { webhookId: hookId },
		});
		expect(await hooks.delete.call(hook(b.ctx))).toBe(true);
		server.done();
	});
});

const deliver = (body: object, { key = secret, ts = Math.floor(Date.now() / 1000) } = {}) => {
	const raw = Buffer.from(JSON.stringify(body));
	const signature = createHmac('sha256', key).update(`${ts}.`).update(raw).digest('hex');
	return {
		body: body as IDataObject,
		rawBody: raw,
		headers: { 'x-reputecast-signature': signature, 'x-reputecast-timestamp': String(ts) },
	};
};

const envelope = (type: string, data: object) => ({
	id: 'evt_01JA9ZK7Q2X8M',
	type,
	created_at: '2026-10-15T13:05:00Z',
	workspace: 'acme-newsletter',
	data,
});

describe('deliveries', () => {
	const receive = (cx: unknown) => trigger.webhook.call(cx as IWebhookFunctions);

	it('starts the workflow with a signed delivery, its data at the top', async () => {
		const server = mockServer([]);
		const d = deliver(
			envelope('contact.unsubscribed', { email: 'rob@example.net', list_name: 'Beta Program' }),
		);
		const { ctx } = context({
			server,
			...d,
			params: { event: 'contact.unsubscribed' },
			staticData: { webhookId: hookId, webhookSecret: secret },
		});
		const out = await receive(ctx);
		expect(out.workflowData?.[0][0].json).toEqual({
			email: 'rob@example.net',
			list_name: 'Beta Program',
			id: 'evt_01JA9ZK7Q2X8M',
			event: 'contact.unsubscribed',
			created_at: '2026-10-15T13:05:00Z',
			workspace: 'acme-newsletter',
		});
	});

	it('refuses a delivery that isn’t signed with this webhook’s secret, or is too old', async () => {
		const server = mockServer([]);
		const body = envelope('contact.unsubscribed', { email: 'rob@example.net' });
		for (const d of [
			deliver(body, { key: 'whsec_other' }),
			deliver(body, { ts: Math.floor(Date.now() / 1000) - 3600 }),
			{ ...deliver(body), headers: {} },
		]) {
			const c = context({
				server,
				...d,
				params: { event: 'contact.unsubscribed' },
				staticData: { webhookId: hookId, webhookSecret: secret },
			});
			const out = await receive(c.ctx);
			expect(out.workflowData).toBeUndefined();
			expect(out.noWebhookResponse).toBe(true);
			expect(c.response.statusCode).toBe(401);
			expect(c.warnings).toHaveLength(1);
		}
	});

	it('keeps only the chosen form’s signups', async () => {
		const server = mockServer([]);
		const mine = deliver(
			envelope('form.submitted', { form_id: 'frm_b7a412', email: 'priya.n@example.com' }),
		);
		const other = deliver(
			envelope('form.submitted', { form_id: 'frm_8c2e1a', email: 'lucia@example.com' }),
		);
		const params = { event: 'form.submitted', form: 'frm_b7a412' };
		const staticData = { webhookId: hookId, webhookSecret: secret };
		expect(
			(await receive(context({ server, ...mine, params, staticData }).ctx)).workflowData,
		).toHaveLength(1);
		expect(
			(await receive(context({ server, ...other, params, staticData }).ctx)).workflowData,
		).toBeUndefined();
		expect(
			(
				await receive(
					context({ server, ...other, params: { event: 'form.submitted', form: '' }, staticData })
						.ctx,
				)
			).workflowData,
		).toHaveLength(1);
	});

	it('lists the forms, any form first', async () => {
		const server = mockServer([
			{
				method: 'GET',
				url: `${WS}/integration/forms`,
				body: {
					items: [
						{ id: 'frm_8c2e1a', name: 'Newsletter signup', status: 'live' },
						{ id: 'frm_b7a412', name: 'Beta apply', status: 'paused' },
					],
				},
				operation: 'list-integration-forms',
			},
		]);
		const out = await trigger.methods.loadOptions.getForms.call(
			context({ server }).ctx as unknown as ILoadOptionsFunctions,
		);
		expect(out).toEqual([
			{ name: 'Any Form', value: '' },
			{ name: 'Newsletter signup', value: 'frm_8c2e1a' },
			{ name: 'Beta apply (paused)', value: 'frm_b7a412' },
		]);
		server.done();
	});
});
