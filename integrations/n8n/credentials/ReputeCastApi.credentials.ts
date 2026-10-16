import type {
	IAuthenticateGeneric,
	ICredentialDataDecryptedObject,
	ICredentialTestRequest,
	ICredentialType,
	IDataObject,
	IHttpRequestHelper,
	Icon,
	INodeProperties,
} from 'n8n-workflow';

import { USER_AGENT, instanceBase, messageOf } from '../nodes/ReputeCast/GenericFunctions';

// The instance URL and the key ReputeCast showed once when n8n was
// connected (Settings › Integrations › n8n). The key knows its workspace:
// the credential asks GET /api/v1/integration which one, once, and keeps it.
export class ReputeCastApi implements ICredentialType {
	name = 'reputeCastApi';

	displayName = 'ReputeCast API';

	icon: Icon = { light: 'file:../icons/reputecast.svg', dark: 'file:../icons/reputecast.svg' };

	documentationUrl =
		'https://github.com/hostrepute/reputecast/blob/main/integrations/n8n/README.md#credentials';

	properties: INodeProperties[] = [
		{
			displayName: 'Instance URL',
			name: 'instanceUrl',
			type: 'string',
			default: '',
			required: true,
			placeholder: 'https://reputecast.acme.io',
			description:
				'Your ReputeCast address, as shown when you connect n8n in ReputeCast (Settings › Integrations › n8n › Connect)',
		},
		{
			displayName: 'Key',
			name: 'apiKey',
			type: 'string',
			typeOptions: { password: true },
			default: '',
			required: true,
			placeholder: 'rc_live_…',
			description:
				'The key ReputeCast showed once when you connected n8n. It works only for the triggers and actions you turned on.',
		},
		{
			displayName: 'Workspace',
			name: 'workspace',
			type: 'hidden',
			typeOptions: { expirable: true },
			default: '',
		},
	];

	// Learns the key's workspace before the first call, and again when the
	// key is refused (401): the answer then says why, in ReputeCast's words.
	async preAuthentication(
		this: IHttpRequestHelper,
		credentials: ICredentialDataDecryptedObject,
	): Promise<IDataObject> {
		const base = instanceBase(credentials.instanceUrl);
		const response = (await this.helpers.httpRequest({
			method: 'GET',
			url: `${base}/api/v1/integration`,
			headers: {
				Authorization: `Bearer ${String(credentials.apiKey ?? '').trim()}`,
				Accept: 'application/json',
				'User-Agent': USER_AGENT,
			},
			json: true,
			returnFullResponse: true,
			ignoreHttpStatusErrors: true,
		})) as { statusCode: number; body: IDataObject };
		if (response.statusCode >= 400) {
			throw new Error(messageOf(response.statusCode, response.body));
		}
		if (!response.body?.workspace) {
			throw new Error(`${base} answered, but not as ReputeCast does. Check the instance URL.`);
		}
		return { workspace: response.body.workspace };
	}

	authenticate: IAuthenticateGeneric = {
		type: 'generic',
		properties: {
			headers: {
				Authorization: '=Bearer {{$credentials.apiKey.trim()}}',
			},
		},
	};

	// Test connection: the call ReputeCast's connect dialog waits for
	// ("n8n called at 14:31 · Test connection · 200").
	test: ICredentialTestRequest = {
		request: {
			baseURL:
				'={{$credentials.instanceUrl.trim().replace(/\\/+$/, "").replace(/\\/api\\/v1$/i, "").replace(/^(?!https?:\\/\\/)/i, "https://")}}',
			url: '/api/v1/integration',
			method: 'GET',
			headers: { Accept: 'application/json', 'User-Agent': USER_AGENT },
		},
	};
}
