import type {
	IDataObject,
	IExecuteFunctions,
	ILoadOptionsFunctions,
	INodeExecutionData,
	INodePropertyOptions,
	INodeType,
	INodeTypeDescription,
	ResourceMapperFields,
	ResourceMapperValue,
} from 'n8n-workflow';
import { NodeConnectionTypes, NodeOperationError } from 'n8n-workflow';

import { asNodeError, reputeCastApiRequest, Skipped } from './GenericFunctions';
import { contactFields, contactOperations } from './descriptions/ContactDescription';
import { eventFields, eventOperations } from './descriptions/EventDescription';
import {
	transactionalFields,
	transactionalOperations,
} from './descriptions/TransactionalDescription';

interface TemplateVariable {
	name: string;
	type: string;
	required: boolean;
	sensitive: boolean;
	description?: string;
}

interface Template {
	key: string;
	name: string;
	version: number;
	variables: TemplateVariable[];
}

const list = (v: unknown): string[] =>
	(Array.isArray(v) ? v : String(v ?? '').split(',')).map((s) => String(s).trim()).filter(Boolean);

const clean = (obj: IDataObject): IDataObject => {
	const out: IDataObject = {};
	for (const [k, v] of Object.entries(obj)) {
		if (v === undefined || v === null || v === '') {
			continue;
		}
		if (Array.isArray(v) && v.length === 0) {
			continue;
		}
		if (typeof v === 'object' && !Array.isArray(v) && Object.keys(v as IDataObject).length === 0) {
			continue;
		}
		out[k] = typeof v === 'string' ? v.trim() : v;
	}
	return out;
};

// A template variable's value as the template expects it.
const typed = (type: string | undefined, v: unknown): unknown => {
	if (v === undefined || v === null || v === '') {
		return undefined;
	}
	switch (type) {
		case 'number': {
			const n = typeof v === 'number' ? v : Number(String(v).trim());
			return Number.isFinite(n) ? n : v;
		}
		case 'boolean': {
			if (typeof v === 'boolean') {
				return v;
			}
			const s = String(v).trim().toLowerCase();
			return ['true', 'yes', '1'].includes(s) ? true : ['false', 'no', '0'].includes(s) ? false : v;
		}
		case 'list':
		case 'object':
			if (typeof v === 'string') {
				try {
					return JSON.parse(v);
				} catch {
					return type === 'list' ? list(v) : v;
				}
			}
			return v;
		default:
			return typeof v === 'string' ? v : String(v);
	}
};

// Back from the mapping's field type to the template's variable type.
const variableType = (type: string | undefined): string => {
	switch (type) {
		case 'number':
		case 'boolean':
		case 'url':
			return type;
		case 'array':
			return 'list';
		case 'object':
			return 'object';
		default:
			return 'text';
	}
};

const fieldType = (type: string) => {
	switch (type) {
		case 'number':
			return 'number' as const;
		case 'boolean':
			return 'boolean' as const;
		case 'url':
			return 'url' as const;
		case 'list':
			return 'array' as const;
		case 'object':
			return 'object' as const;
		default:
			return 'string' as const;
	}
};

export class ReputeCast implements INodeType {
	description: INodeTypeDescription = {
		displayName: 'ReputeCast',
		name: 'reputeCast',
		icon: { light: 'file:../../icons/reputecast.svg', dark: 'file:../../icons/reputecast.svg' },
		group: ['output'],
		version: 1,
		subtitle: '={{$parameter["operation"] + ": " + $parameter["resource"]}}',
		description:
			'Create or update contacts, tag them, record events and send transactional email in ReputeCast',
		defaults: {
			name: 'ReputeCast',
		},
		// n8n's AI Agent can use it as a tool, limited to what the key may do:
		// Send a transactional email starts off when connecting n8n.
		usableAsTool: true,
		inputs: [NodeConnectionTypes.Main],
		outputs: [NodeConnectionTypes.Main],
		credentials: [
			{
				name: 'reputeCastApi',
				required: true,
			},
		],
		properties: [
			{
				displayName: 'Resource',
				name: 'resource',
				type: 'options',
				noDataExpression: true,
				options: [
					{
						name: 'Contact',
						value: 'contact',
					},
					{
						name: 'Event',
						value: 'event',
					},
					{
						name: 'Transactional Email',
						value: 'transactionalEmail',
					},
				],
				default: 'contact',
			},
			...contactOperations,
			...contactFields,
			...eventOperations,
			...eventFields,
			...transactionalOperations,
			...transactionalFields,
		],
	};

	methods = {
		loadOptions: {
			async getLists(this: ILoadOptionsFunctions): Promise<INodePropertyOptions[]> {
				const res = (await reputeCastApiRequest.call(
					this,
					'GET',
					'/workspaces/{workspace}/integration/lists',
				)) as {
					items: Array<{ id: string; name: string; doubleOptIn: boolean }>;
				};
				return res.items.map((l) => ({
					name: l.doubleOptIn ? `${l.name} (double opt-in)` : l.name,
					value: l.id,
				}));
			},

			async getFields(this: ILoadOptionsFunctions): Promise<INodePropertyOptions[]> {
				const res = (await reputeCastApiRequest.call(
					this,
					'GET',
					'/workspaces/{workspace}/integration/fields',
				)) as {
					items: Array<{ key: string; label: string; type: string }>;
				};
				return res.items.map((f) => ({
					name: f.label || f.key,
					value: f.key,
					description: `${f.type} field`,
				}));
			},

			async getTemplates(this: ILoadOptionsFunctions): Promise<INodePropertyOptions[]> {
				const res = (await reputeCastApiRequest.call(
					this,
					'GET',
					'/workspaces/{workspace}/integration/templates',
				)) as {
					items: Template[];
				};
				return res.items.map((t) => ({
					name: `${t.name} (${t.key})`,
					value: t.key,
					description: `Live version ${t.version}`,
				}));
			},
		},

		resourceMapping: {
			// The chosen template's variables, required as the template says.
			async getTemplateVariables(this: ILoadOptionsFunctions): Promise<ResourceMapperFields> {
				const key = this.getNodeParameter('templateKey', '') as string;
				if (!key) {
					return { fields: [], emptyFieldsNotice: 'Choose a template first' };
				}
				const res = (await reputeCastApiRequest.call(
					this,
					'GET',
					'/workspaces/{workspace}/integration/templates',
				)) as {
					items: Template[];
				};
				const t = res.items.find((x) => x.key === key);
				if (!t) {
					return { fields: [], emptyFieldsNotice: `No published template ${key}` };
				}
				return {
					fields: t.variables.map((v) => ({
						id: v.name,
						displayName: v.sensitive
							? `${v.name} (sensitive: masked in ReputeCast's call log)`
							: v.name,
						required: v.required,
						defaultMatch: false,
						canBeUsedToMatch: false,
						display: true,
						type: fieldType(v.type),
					})),
					emptyFieldsNotice: 'This template has no variables',
				};
			},
		},
	};

	async execute(this: IExecuteFunctions): Promise<INodeExecutionData[][]> {
		const items = this.getInputData();
		const returnData: INodeExecutionData[] = [];
		const resource = this.getNodeParameter('resource', 0) as string;
		const operation = this.getNodeParameter('operation', 0) as string;

		for (let i = 0; i < items.length; i++) {
			try {
				let result: unknown;
				if (resource === 'contact' && operation === 'upsert') {
					const extra = this.getNodeParameter('additionalFields', i, {}) as IDataObject;
					const fields: IDataObject = {};
					const custom = (extra.customFields as IDataObject | undefined)?.field as
						| IDataObject[]
						| undefined;
					for (const f of custom ?? []) {
						if (f.key) {
							fields[String(f.key)] = f.value;
						}
					}
					const body = clean({
						email: this.getNodeParameter('email', i) as string,
						firstName: extra.firstName,
						lastName: extra.lastName,
						company: extra.company,
						language: extra.language,
						fields: clean(fields),
						tags: list(extra.tags),
						lists: list(extra.lists),
						consent: extra.consent,
					});
					result = await reputeCastApiRequest.call(
						this,
						'POST',
						'/workspaces/{workspace}/contacts/upsert',
						{ body },
					);
				} else if (resource === 'contact' && operation === 'addToList') {
					const listId = this.getNodeParameter('listId', i) as string;
					const body = clean({
						email: this.getNodeParameter('email', i) as string,
						consent: this.getNodeParameter('consent', i, ''),
					});
					result = await reputeCastApiRequest.call(
						this,
						'POST',
						`/workspaces/{workspace}/lists/${encodeURIComponent(listId)}/contacts`,
						{ body },
					);
				} else if (resource === 'contact' && operation === 'addTag') {
					const body = {
						email: (this.getNodeParameter('email', i) as string).trim(),
						tags: list(this.getNodeParameter('tags', i)),
					};
					result = await reputeCastApiRequest.call(
						this,
						'POST',
						'/workspaces/{workspace}/contacts/tags',
						{ body },
					);
				} else if (resource === 'event' && operation === 'record') {
					const props: IDataObject = {};
					const pairs = (this.getNodeParameter('properties', i, {}) as IDataObject).property as
						| IDataObject[]
						| undefined;
					for (const p of pairs ?? []) {
						if (p.name) {
							props[String(p.name)] = p.value;
						}
					}
					const body = clean({
						event: this.getNodeParameter('eventName', i) as string,
						email: this.getNodeParameter('email', i) as string,
						properties: props,
					});
					result = await reputeCastApiRequest.call(
						this,
						'POST',
						'/workspaces/{workspace}/automations/events',
						{ body },
					);
				} else if (resource === 'transactionalEmail' && operation === 'send') {
					const templateKey = this.getNodeParameter('templateKey', i) as string;
					const extra = this.getNodeParameter('additionalFields', i, {}) as IDataObject;
					const mapper = this.getNodeParameter('variables', i, {
						mappingMode: 'defineBelow',
						value: null,
					}) as ResourceMapperValue;
					let values: IDataObject = {};
					if (mapper.mappingMode === 'autoMapInputData') {
						for (const f of mapper.schema ?? []) {
							if (!f.removed && f.id in items[i].json) {
								values[f.id] = items[i].json[f.id];
							}
						}
					} else {
						values = (mapper.value ?? {}) as IDataObject;
					}
					// The variables' types come with the mapping's schema.
					const types = new Map((mapper.schema ?? []).map((f) => [f.id, variableType(f.type)]));
					const data: IDataObject = {};
					for (const [name, v] of Object.entries(values)) {
						const value = typed(types.get(name), v);
						if (value !== undefined) {
							data[name] = value as IDataObject[keyof IDataObject];
						}
					}
					const body: IDataObject = {
						template: templateKey,
						to: clean({ email: this.getNodeParameter('toEmail', i) as string, name: extra.toName }),
					};
					if (Object.keys(data).length) {
						body.data = data;
					}
					if (extra.language) {
						body.language = String(extra.language).trim();
					}
					const headers: IDataObject = {};
					if (extra.idempotencyKey) {
						headers['Idempotency-Key'] =
							`n8n-${this.getWorkflow().id ?? ''}-${String(extra.idempotencyKey).trim()}`.slice(
								0,
								255,
							);
					}
					result = await reputeCastApiRequest.call(
						this,
						'POST',
						'/workspaces/{workspace}/transactional/messages',
						{ body, headers },
					);
				} else {
					throw new NodeOperationError(
						this.getNode(),
						`The operation "${operation}" isn't supported for "${resource}"`,
						{ itemIndex: i },
					);
				}
				const json = result instanceof Skipped ? result.toJSON() : ((result ?? {}) as IDataObject);
				returnData.push({ json, pairedItem: { item: i } });
			} catch (error) {
				if (this.continueOnFail()) {
					returnData.push({ json: { error: (error as Error).message }, pairedItem: { item: i } });
					continue;
				}
				throw asNodeError(this, error, i);
			}
		}
		return [returnData];
	}
}
