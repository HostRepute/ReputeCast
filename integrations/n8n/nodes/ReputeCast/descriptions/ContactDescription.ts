import type { INodeProperties } from 'n8n-workflow';

const optIn: INodeProperties = {
	displayName: 'Opt-In',
	name: 'consent',
	type: 'options',
	options: [
		{
			name: "Each List's Own Setting",
			value: '',
		},
		{
			name: 'Double Opt-In',
			value: 'double',
			description: 'Send the confirmation email first',
		},
		{
			name: 'Single Opt-In',
			value: 'single',
			description: 'Add them to the list now',
		},
	],
	default: '',
	description:
		'How the contact joins. ReputeCast never adds someone back to a list they left, and never adds a suppressed address: such items come out with skipped set to true and the reason.',
};

export const contactOperations: INodeProperties[] = [
	{
		displayName: 'Operation',
		name: 'operation',
		type: 'options',
		noDataExpression: true,
		displayOptions: {
			show: {
				resource: ['contact'],
			},
		},
		options: [
			{
				name: 'Add Tag',
				value: 'addTag',
				description: 'Add one or more tags to a contact',
				action: 'Add a tag to a contact',
			},
			{
				name: 'Add to List',
				value: 'addToList',
				description: 'Add a contact to a list, with single or double opt-in',
				action: 'Add a contact to a list',
			},
			{
				name: 'Create or Update',
				value: 'upsert',
				description: 'Create a new record, or update the current one if it already exists (upsert)',
				action: 'Create or update a contact',
			},
		],
		default: 'upsert',
	},
];

export const contactFields: INodeProperties[] = [
	{
		displayName: 'List Name or ID',
		name: 'listId',
		type: 'options',
		typeOptions: {
			loadOptionsMethod: 'getLists',
		},
		required: true,
		default: '',
		description:
			'The list to add the contact to. Choose from the list, or specify an ID using an <a href="https://docs.n8n.io/code/expressions/">expression</a>.',
		displayOptions: {
			show: {
				resource: ['contact'],
				operation: ['addToList'],
			},
		},
	},
	{
		displayName: 'Email',
		name: 'email',
		type: 'string',
		placeholder: 'name@email.com',
		required: true,
		default: '',
		description: 'The contact is matched by this address',
		displayOptions: {
			show: {
				resource: ['contact'],
			},
		},
	},
	{
		...optIn,
		displayOptions: {
			show: {
				resource: ['contact'],
				operation: ['addToList'],
			},
		},
	},
	{
		displayName: 'Tags',
		name: 'tags',
		type: 'string',
		required: true,
		default: '',
		placeholder: 'customer, beta-applicant',
		description: 'Tags to add, separated by commas. Tags the contact already has stay.',
		displayOptions: {
			show: {
				resource: ['contact'],
				operation: ['addTag'],
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
				resource: ['contact'],
				operation: ['upsert'],
			},
		},
		options: [
			{
				displayName: 'Company',
				name: 'company',
				type: 'string',
				default: '',
			},
			{
				displayName: 'Custom Fields',
				name: 'customFields',
				type: 'fixedCollection',
				placeholder: 'Add Custom Field',
				typeOptions: {
					multipleValues: true,
				},
				default: {},
				description: 'Values of the workspace’s custom fields (Audience › Fields)',
				options: [
					{
						displayName: 'Field',
						name: 'field',
						values: [
							{
								displayName: 'Field Name or ID',
								name: 'key',
								type: 'options',
								typeOptions: {
									loadOptionsMethod: 'getFields',
								},
								default: '',
								description:
									'Choose from the list, or specify an ID using an <a href="https://docs.n8n.io/code/expressions/">expression</a>',
							},
							{
								displayName: 'Value',
								name: 'value',
								type: 'string',
								default: '',
								description:
									'A number for number fields, true or false for yes/no fields, a date like 2026-10-30 for date fields',
							},
						],
					},
				],
			},
			{
				displayName: 'First Name',
				name: 'firstName',
				type: 'string',
				default: '',
			},
			{
				displayName: 'Language',
				name: 'language',
				type: 'string',
				default: '',
				placeholder: 'es-MX',
				description:
					'A language code such as es or es-MX. Names like Spanish and codes like es_MX work too.',
			},
			{
				displayName: 'Last Name',
				name: 'lastName',
				type: 'string',
				default: '',
			},
			{
				displayName: 'List Names or IDs',
				name: 'lists',
				type: 'multiOptions',
				typeOptions: {
					loadOptionsMethod: 'getLists',
				},
				default: [],
				description:
					'Lists to add the contact to. Choose from the list, or specify IDs using an <a href="https://docs.n8n.io/code/expressions/">expression</a>.',
			},
			{
				...optIn,
			},
			{
				displayName: 'Tags',
				name: 'tags',
				type: 'string',
				default: '',
				placeholder: 'beta-applicant',
				description: 'Tags to add, separated by commas. Tags the contact already has stay.',
			},
		],
	},
];
