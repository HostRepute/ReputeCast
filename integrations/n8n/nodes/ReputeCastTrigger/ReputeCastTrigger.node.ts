import type {
	IDataObject,
	IHookFunctions,
	ILoadOptionsFunctions,
	INodePropertyOptions,
	INodeType,
	INodeTypeDescription,
	IWebhookFunctions,
	IWebhookResponseData,
} from 'n8n-workflow';
import { NodeConnectionTypes } from 'n8n-workflow';

import { flatten, reputeCastApiRequest, verifySignature } from '../ReputeCast/GenericFunctions';

// Instant triggers through a managed webhook: turning the workflow on
// subscribes (ReputeCast shows the webhook read-only as "Managed by n8n"),
// turning it off unsubscribes. When n8n answers 410 Gone, ReputeCast
// removes the webhook itself.

interface Hook {
	id: string;
	event: string;
	url: string;
	secret?: string;
}

interface HookData extends IDataObject {
	webhookId?: string;
	webhookSecret?: string;
	webhookEvent?: string;
}

export class ReputeCastTrigger implements INodeType {
	description: INodeTypeDescription = {
		displayName: 'ReputeCast Trigger',
		name: 'reputeCastTrigger',
		icon: { light: 'file:../../icons/reputecast.svg', dark: 'file:../../icons/reputecast.svg' },
		group: ['trigger'],
		version: 1,
		subtitle: '={{$parameter["event"]}}',
		description:
			'Starts the workflow when contacts subscribe, unsubscribe or change, a form is submitted, a campaign is sent or a transactional email bounces',
		defaults: {
			name: 'ReputeCast Trigger',
		},
		inputs: [],
		outputs: [NodeConnectionTypes.Main],
		credentials: [
			{
				name: 'reputeCastApi',
				required: true,
			},
		],
		webhooks: [
			{
				name: 'default',
				httpMethod: 'POST',
				responseMode: 'onReceived',
				path: 'reputecast',
			},
		],
		properties: [
			{
				displayName: 'Event',
				name: 'event',
				type: 'options',
				required: true,
				default: 'contact.subscribed',
				options: [
					{
						name: 'Campaign Sent',
						value: 'campaign.completed',
						description: 'A campaign finishes sending, with its totals',
					},
					{
						name: 'Contact Subscribed',
						value: 'contact.subscribed',
						description: 'A contact subscribes to a list',
					},
					{
						name: 'Contact Unsubscribed',
						value: 'contact.unsubscribed',
						description: 'A contact unsubscribes from a list',
					},
					{
						name: 'Contact Updated',
						value: 'contact.updated',
						description: "A contact's names, fields, language, tags or lists change",
					},
					{
						name: 'Form Submitted',
						value: 'form.submitted',
						description: 'A signup through a form is confirmed',
					},
					{
						name: 'Transactional Message Bounced',
						value: 'message.bounced',
						description: 'A transactional email bounces',
					},
				],
			},
			{
				displayName: 'Form Name or ID',
				name: 'form',
				type: 'options',
				typeOptions: {
					loadOptionsMethod: 'getForms',
				},
				default: '',
				description:
					'Only this form’s signups start the workflow; leave it empty for every form. Choose from the list, or specify an ID using an <a href="https://docs.n8n.io/code/expressions/">expression</a>.',
				displayOptions: {
					show: {
						event: ['form.submitted'],
					},
				},
			},
		],
	};

	methods = {
		loadOptions: {
			async getForms(this: ILoadOptionsFunctions): Promise<INodePropertyOptions[]> {
				const res = (await reputeCastApiRequest.call(
					this,
					'GET',
					'/workspaces/{workspace}/integration/forms',
				)) as {
					items: Array<{ id: string; name: string; status: string }>;
				};
				return [
					{ name: 'Any Form', value: '' },
					...res.items.map((f) => ({
						name: f.status === 'live' ? f.name : `${f.name} (${f.status})`,
						value: f.id,
					})),
				];
			},
		},
	};

	webhookMethods = {
		default: {
			// The webhook this workflow made still exists, for this event and URL.
			async checkExists(this: IHookFunctions): Promise<boolean> {
				const data = this.getWorkflowStaticData('node') as HookData;
				if (!data.webhookId) {
					return false;
				}
				const hook = (await reputeCastApiRequest.call(
					this,
					'GET',
					`/workspaces/{workspace}/integration/hooks/${encodeURIComponent(data.webhookId)}`,
					{
						allow: [404],
					},
				)) as Partial<Hook> & { status?: number };
				const event = this.getNodeParameter('event') as string;
				if (
					hook?.id === data.webhookId &&
					hook.url === this.getNodeWebhookUrl('default') &&
					hook.event === event
				) {
					return true;
				}
				// Gone (a 410 Gone removed it, or n8n was reconnected), or made for
				// another event or address: forget it, and create it again.
				delete data.webhookId;
				delete data.webhookSecret;
				delete data.webhookEvent;
				return false;
			},

			async create(this: IHookFunctions): Promise<boolean> {
				const url = this.getNodeWebhookUrl('default');
				const event = this.getNodeParameter('event') as string;
				const hook = (await reputeCastApiRequest.call(
					this,
					'POST',
					'/workspaces/{workspace}/integration/hooks',
					{
						body: { event, url },
					},
				)) as Hook;
				const data = this.getWorkflowStaticData('node') as HookData;
				data.webhookId = hook.id;
				// The signing secret, shown only in this answer.
				data.webhookSecret = hook.secret;
				data.webhookEvent = event;
				return true;
			},

			async delete(this: IHookFunctions): Promise<boolean> {
				const data = this.getWorkflowStaticData('node') as HookData;
				if (data.webhookId) {
					// 404: already gone, which is what was asked.
					await reputeCastApiRequest.call(
						this,
						'DELETE',
						`/workspaces/{workspace}/integration/hooks/${encodeURIComponent(data.webhookId)}`,
						{
							allow: [404],
						},
					);
				}
				delete data.webhookId;
				delete data.webhookSecret;
				delete data.webhookEvent;
				return true;
			},
		},
	};

	async webhook(this: IWebhookFunctions): Promise<IWebhookResponseData> {
		const data = this.getWorkflowStaticData('node') as HookData;
		const headers = this.getHeaderData();
		const req = this.getRequestObject() as unknown as { rawBody?: Buffer };
		const body = this.getBodyData();
		if (data.webhookSecret) {
			if (!req.rawBody) {
				// n8n keeps the raw body of webhook requests; without it the
				// signature can't be recomputed byte for byte.
				this.logger.warn(
					'ReputeCast Trigger: the raw body is missing, so the delivery’s signature wasn’t checked',
				);
			} else {
				const check = verifySignature({
					secret: data.webhookSecret,
					body: req.rawBody,
					signature: String(headers['x-reputecast-signature'] ?? ''),
					timestamp: String(headers['x-reputecast-timestamp'] ?? ''),
				});
				if (!check.ok) {
					this.logger.warn(`ReputeCast Trigger: ignored a delivery, ${check.why}`);
					const res = this.getResponseObject();
					res.status(401).json({ message: `Ignored: ${check.why}.` });
					return { noWebhookResponse: true };
				}
			}
		}
		const event = this.getNodeParameter('event') as string;
		if (body.type && body.type !== event) {
			return { webhookResponse: { ignored: true } };
		}
		const item = flatten(body);
		const form = this.getNodeParameter('form', '') as string;
		if (event === 'form.submitted' && form && item.form_id !== form) {
			return { webhookResponse: { ignored: true } };
		}
		return { workflowData: [this.helpers.returnJsonArray([item])] };
	}
}
