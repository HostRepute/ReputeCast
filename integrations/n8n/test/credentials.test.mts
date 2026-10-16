import type {
	ICredentialDataDecryptedObject,
	IHttpRequestHelper,
	IHttpRequestOptions,
} from 'n8n-workflow';
import { describe, expect, it } from 'vitest';

import { ReputeCastApi } from '../credentials/ReputeCastApi.credentials';
import { instanceBase, messageOf, workflowHeader } from '../nodes/ReputeCast/GenericFunctions';
import { BASE, KEY, problem } from './fakes.mts';
import { checkRequest } from '../../contract/openapi-contract.mjs';

const cred = new ReputeCastApi();

const helper = (statusCode: number, body: unknown) => {
	const calls: IHttpRequestOptions[] = [];
	const h = {
		helpers: {
			httpRequest: async (options: IHttpRequestOptions) => {
				calls.push(options);
				return { statusCode, body, headers: {} };
			},
		},
	} as unknown as IHttpRequestHelper;
	return { h, calls };
};

describe('the ReputeCast API credential', () => {
	it('asks for the instance URL and the key, and keeps the workspace hidden', () => {
		expect(cred.properties.map((p) => [p.name, p.type, p.typeOptions ?? {}])).toEqual([
			['instanceUrl', 'string', {}],
			['apiKey', 'string', { password: true }],
			['workspace', 'hidden', { expirable: true }],
		]);
	});

	it('learns the key’s workspace before the first call', async () => {
		const me = {
			workspace: 'acme-newsletter',
			workspaceName: 'Acme Newsletter',
			integration: 'n8n',
			key: 'n8n · Acme automations',
			triggers: [],
			actions: [],
			permissions: [],
		};
		const { h, calls } = helper(200, me);
		const creds = {
			instanceUrl: 'reputecast.acme.io/',
			apiKey: ` ${KEY} `,
		} as ICredentialDataDecryptedObject;
		expect(await cred.preAuthentication.call(h, creds)).toEqual({ workspace: 'acme-newsletter' });
		expect(calls[0].url).toBe(`${BASE}/api/v1/integration`);
		expect((calls[0].headers as Record<string, string>).Authorization).toBe(`Bearer ${KEY}`);
		expect(checkRequest({ method: 'GET', url: calls[0].url, body: undefined }).ok).toBe(true);
	});

	it('says why a key is refused, in ReputeCast’s words', async () => {
		const detail =
			'This key was revoked. Reconnect n8n in Settings › Integrations to get a new key.';
		const { h } = helper(401, problem(401, 'key_revoked', detail));
		await expect(
			cred.preAuthentication.call(h, { instanceUrl: BASE, apiKey: KEY }),
		).rejects.toThrow(detail);
	});

	it('tests the connection with GET /api/v1/integration', () => {
		expect(cred.test.request.url).toBe('/api/v1/integration');
		expect(cred.authenticate).toEqual({
			type: 'generic',
			properties: { headers: { Authorization: '=Bearer {{$credentials.apiKey.trim()}}' } },
		});
	});
});

describe('helpers', () => {
	it('cleans the instance URL', () => {
		expect(instanceBase('reputecast.acme.io')).toBe('https://reputecast.acme.io');
		expect(instanceBase(' https://reputecast.acme.io/api/v1/ ')).toBe('https://reputecast.acme.io');
		expect(instanceBase('http://10.0.4.20:8080/')).toBe('http://10.0.4.20:8080');
	});

	it('names the workflow, percent-encoding what isn’t plain ASCII', () => {
		expect(workflowHeader({ name: 'Unsubscribes → Acme Cloud CRM', id: 'wf_2Lm8' })).toBe(
			'Unsubscribes %E2%86%92 Acme Cloud CRM; id=wf_2Lm8',
		);
		expect(workflowHeader({ name: 'Weekly; 100% = "done"' })).toBe(
			'Weekly%3B 100%25 %3D %22done%22',
		);
		expect(workflowHeader({ id: 42 })).toBe('id=42');
		expect(workflowHeader({})).toBe('');
	});

	it('reads problems as sentences', () => {
		expect(
			messageOf(422, { detail: 'payment-failed v1 needs amount_due.', missing: ['amount_due'] }),
		).toBe('payment-failed v1 needs amount_due.');
		expect(
			messageOf(422, {
				detail: 'validation failed',
				errors: [{ location: 'body.email', message: 'expected length >= 3' }],
			}),
		).toBe('validation failed. email: expected length >= 3.');
		expect(messageOf(502, '')).toBe('ReputeCast answered HTTP 502.');
	});
});
