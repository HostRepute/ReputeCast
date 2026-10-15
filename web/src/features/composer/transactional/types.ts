import type { Schemas } from '@/api/client'

export type Template = Schemas['TransactionalTemplate']
export type Variable = Schemas['TransactionalVariable']
export type VarType = Variable['type']
export type TestData = Schemas['TransactionalTestData']
export type Checks = Schemas['TransactionalChecks']
export type Check = Schemas['TransactionalCheck']
export type CheckFix = Schemas['TransactionalCheckFix']
export type Settings = Schemas['TransactionalSettings']
export type Category = Schemas['TransactionalCategory']
export type Preview = Schemas['TransactionalPreview']
export type PublishPreview = Schemas['TransactionalPublishPreview']
export type Version = Schemas['TransactionalVersion']
export type ApprovalState = Schemas['TransactionalApprovalState']

/** The editor's views: the canvas, the MJML and compiled HTML code views, and the plain text. */
export type TxView = 'visual' | 'mjml' | 'html' | 'text'

/** Right panel tabs (TransactionalTemplate board). */
export type TxTab = 'design' | 'variables' | 'testdata' | 'checks' | 'api'

export const VAR_TYPES: VarType[] = ['text', 'url', 'number', 'boolean', 'date', 'list', 'object']

/** Data a render gets: what an API call sends in `data`. */
export type Data = Record<string, unknown>

/** The recipient previews render with (the server's own default). */
export const PREVIEW_TO = { email: 'recipient@example.com', name: 'Sam Rivera' }

/** Names ReputeCast fills in at send time; templates use them without declaring them. */
export const SYSTEM_NAMES = ['to', 'unsubscribe_url', 'preferences_url'] as const
