import { $api } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Popover } from '@/components/ui/Menu'
import { useT, type MessageDescriptor } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './contacts.messages'

/**
 * The contacts table's columns and orders (review 2026-11 AU-9,
 * ContactsAtScale › Columns and order): a view keeps both. Custom fields
 * are columns too ("field:plan").
 */
export type BuiltinColumn = 'lists' | 'tier' | 'score' | 'consent' | 'mailbox' | 'last_open' | 'joined' | 'source' | 'paid' | 'phone'
export type ColumnId = BuiltinColumn | `field:${string}`
export type ContactSort = 'activity' | 'score' | 'joined' | 'last_open' | 'email'

export const columnLabels: Record<BuiltinColumn, MessageDescriptor> = {
  lists: m.colLists,
  tier: m.colTier,
  score: m.colScore,
  consent: m.colConsent,
  mailbox: m.colMailbox,
  last_open: m.colLastOpen,
  joined: m.colJoined,
  source: m.colSource,
  paid: m.colPaid,
  phone: m.colPhone,
}

const chooseable: BuiltinColumn[] = ['lists', 'tier', 'score', 'consent', 'mailbox', 'last_open', 'joined', 'source']

/** The usual columns: the table's own, with Paid and Phone once those features are set up. */
export function defaultColumns(paid: boolean, phones: boolean): ColumnId[] {
  const out: ColumnId[] = ['lists', 'tier', 'score', 'consent', 'mailbox', 'last_open', 'joined', 'source']
  if (paid) out.splice(0, 0, 'paid')
  if (phones) out.splice(paid ? 1 : 0, 0, 'phone')
  return out
}

/** Reads ?cols=lists,tier,field:plan; undefined for the usual columns. */
export function parseColumns(raw: string | undefined): ColumnId[] | undefined {
  if (!raw) return undefined
  const out = raw
    .split(',')
    .map((c) => c.trim())
    .filter((c): c is ColumnId => c.startsWith('field:') || (Object.keys(columnLabels) as string[]).includes(c))
  return out.length ? [...new Set(out)] : undefined
}

/** The custom field keys among the columns, for the list request. */
export function fieldKeys(cols: ColumnId[]): string[] {
  return cols.filter((c) => c.startsWith('field:')).map((c) => c.slice('field:'.length))
}

/** "Columns · 8": the table's columns and the workspace's custom fields. */
export function ColumnChooser({ columns, onChange, onReset }: { columns: ColumnId[]; onChange: (c: ColumnId[]) => void; onReset: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const fields = $api.useQuery('get', '/api/v1/workspaces/{workspace}/fields', { params: { path: { workspace: slug } } })
  const toggle = (c: ColumnId, on: boolean) => onChange(on ? [...columns, c] : columns.filter((x) => x !== c))
  const box = (c: ColumnId, label: string) => (
    <label key={c} className="flex min-h-8 items-center gap-2.5 rounded-lg px-2.5 text-[13px] hover:bg-ground">
      <input type="checkbox" checked={columns.includes(c)} onChange={(e) => toggle(c, e.target.checked)} />
      {label}
    </label>
  )
  return (
    <Popover
      align="end"
      label={t(m.columns)}
      className="flex max-h-(--radix-popover-content-available-height) w-[300px] flex-col gap-3 overflow-y-auto p-3.5"
      trigger={<button className="btn">{t(m.columnsCount, { n: columns.length })}</button>}
    >
      <fieldset className="m-0 flex flex-col gap-0.5 border-0 p-0">
        <legend className="mb-1.5 text-xs font-semibold text-ink-2">{t(m.columns)}</legend>
        {chooseable.map((c) => box(c, t(columnLabels[c])))}
      </fieldset>
      {fields.data && fields.data.items.length > 0 ? (
        <fieldset className="m-0 flex flex-col gap-0.5 border-0 p-0">
          <legend className="mb-1.5 text-[11px] font-bold tracking-[.05em] text-ink-3 uppercase">{t(m.customFields)}</legend>
          {fields.data.items.map((f) => box(`field:${f.key}`, f.label || f.key))}
        </fieldset>
      ) : null}
      <div className="flex items-center justify-between gap-2">
        <Button size="sm" onClick={onReset}>
          {t(m.resetColumns)}
        </Button>
        <span className="k">{t(m.columnsInView)}</span>
      </div>
    </Popover>
  )
}

/**
 * A sortable header: the first click sorts by the column (newest or
 * highest first, A to Z for email), the next goes back to last activity.
 */
export function SortHeader({ label, sort, current, onSort }: { label: string; sort: ContactSort; current: ContactSort; onSort: (s: ContactSort) => void }) {
  const t = useT()
  const on = current === sort
  return (
    <th scope="col" aria-sort={on ? (sort === 'email' ? 'ascending' : 'descending') : undefined}>
      <button type="button" className="sc-sortbtn" onClick={() => onSort(on ? 'activity' : sort)} title={on ? t(m.sortByActivity) : t(m.sortBy, { column: label })}>
        {label}
        <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          {on ? sort === 'email' ? <path d="M8 13V3M4 7l4-4 4 4" /> : <path d="M8 3v10M4 9l4 4 4-4" /> : <path d="M5 6l3-3 3 3M5 10l3 3 3-3" />}
        </svg>
      </button>
    </th>
  )
}
