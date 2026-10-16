import type { INodeProperties } from 'n8n-workflow';

export const eventOperations: INodeProperties[] = [
	{
		displayName: 'Operation',
		name: 'operation',
		type: 'options',
		noDataExpression: true,
		displayOptions: {
			show: {
				resource: ['event'],
			},
		},
		options: [
			{
				name: 'Record',
				value: 'record',
				description:
					'Record an event for a contact, which starts the automations that listen for it',
				action: 'Record an event',
			},
		],
		default: 'record',
	},
];

export const eventFields: INodeProperties[] = [
	{
		displayName: 'Event Name',
		name: 'eventName',
		type: 'string',
		required: true,
		default: '',
		placeholder: 'workspace.created',
		description:
			'Letters, numbers and . _ : -. Automations that start with this API event, or have it as their goal, run for the contact.',
		displayOptions: {
			show: {
				resource: ['event'],
				operation: ['record'],
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
		description: 'The contact the event is about. They must be in ReputeCast already.',
		displayOptions: {
			show: {
				resource: ['event'],
				operation: ['record'],
			},
		},
	},
	{
		displayName: 'Properties',
		name: 'properties',
		type: 'fixedCollection',
		placeholder: 'Add Property',
		typeOptions: {
			multipleValues: true,
		},
		default: {},
		description:
			'Details of the event, such as plan: Team. Automations can use them in conditions and emails.',
		displayOptions: {
			show: {
				resource: ['event'],
				operation: ['record'],
			},
		},
		options: [
			{
				displayName: 'Property',
				name: 'property',
				values: [
					{
						displayName: 'Name',
						name: 'name',
						type: 'string',
						default: '',
					},
					{
						displayName: 'Value',
						name: 'value',
						type: 'string',
						default: '',
					},
				],
			},
		],
	},
];
