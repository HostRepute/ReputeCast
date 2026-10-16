import type { INodeProperties } from 'n8n-workflow';

export const transactionalOperations: INodeProperties[] = [
	{
		displayName: 'Operation',
		name: 'operation',
		type: 'options',
		noDataExpression: true,
		displayOptions: {
			show: {
				resource: ['transactionalEmail'],
			},
		},
		options: [
			{
				name: 'Send',
				value: 'send',
				description: 'Send a published transactional template to one recipient',
				action: 'Send a transactional email',
			},
		],
		default: 'send',
	},
];

export const transactionalFields: INodeProperties[] = [
	{
		displayName: 'Template Name or ID',
		name: 'templateKey',
		type: 'options',
		typeOptions: {
			loadOptionsMethod: 'getTemplates',
		},
		required: true,
		default: '',
		description:
			'A published transactional template; its live version is sent. Choose from the list, or specify an ID using an <a href="https://docs.n8n.io/code/expressions/">expression</a>.',
		displayOptions: {
			show: {
				resource: ['transactionalEmail'],
				operation: ['send'],
			},
		},
	},
	{
		displayName: 'Recipient Email',
		name: 'toEmail',
		type: 'string',
		placeholder: 'name@email.com',
		required: true,
		default: '',
		displayOptions: {
			show: {
				resource: ['transactionalEmail'],
				operation: ['send'],
			},
		},
	},
	{
		displayName: 'Variables',
		name: 'variables',
		type: 'resourceMapper',
		noDataExpression: true,
		default: {
			mappingMode: 'defineBelow',
			value: null,
		},
		required: true,
		typeOptions: {
			loadOptionsDependsOn: ['templateKey'],
			resourceMapper: {
				resourceMapperMethod: 'getTemplateVariables',
				mode: 'add',
				fieldWords: {
					singular: 'variable',
					plural: 'variables',
				},
				addAllFields: true,
				multiKeyMatch: false,
				supportAutoMap: true,
			},
		},
		displayOptions: {
			show: {
				resource: ['transactionalEmail'],
				operation: ['send'],
			},
		},
	},
	{
		displayName: 'Additional Fields',
		name: 'additionalFields',
		type: 'collection',
		placeholder: 'Add Field',
		default: {},
		displayOptions: {
			show: {
				resource: ['transactionalEmail'],
				operation: ['send'],
			},
		},
		options: [
			{
				displayName: 'Idempotency Key',
				name: 'idempotencyKey',
				type: 'string',
				default: '',
				description:
					'A value unique to this email, such as the triggering event’s ID. If it runs again with the same value within 24 hours, ReputeCast sends nothing new.',
			},
			{
				displayName: 'Language',
				name: 'language',
				type: 'string',
				default: '',
				placeholder: 'es',
				description:
					"The language version to send. Without it, the contact's language is used, then the workspace's fallback.",
			},
			{
				displayName: 'Recipient Name',
				name: 'toName',
				type: 'string',
				default: '',
			},
		],
	},
];
