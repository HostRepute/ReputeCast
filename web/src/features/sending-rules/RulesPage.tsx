import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useId, useRef, useState, type ReactNode, type Ref } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard, Select, Textarea, Toggle } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, clock, decimal, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { m } from './messages'
import './sending-rules.css'

type View = Schemas['SendingRulesView']
type Rule = Schemas['SendingRule']
type RuleInput = Schemas['SendingRuleInput']
type Kind = RuleInput['kind']
type Operator = RuleInput['operator']
type Scope = RuleInput['appliesTo']
type Severity = RuleInput['severity']

const listPath = '/api/v1/workspaces/{workspace}/sending-rules' as const
const rulePath = '/api/v1/workspaces/{workspace}/sending-rules/{ruleId}' as const
const defaultsPath = '/api/v1/workspaces/{workspace}/send-defaults' as const

/** The operators of each kind of rule, in the order the board lists them. */
const operators: Record<Kind, Operator[]> = {
  link_param: ['on_every', 'on_none'],
  link_domain: ['not_one_of', 'one_of'],
  subject_length: ['at_most', 'at_least'],
  contains_text: ['footer_contains', 'footer_lacks', 'body_contains', 'subject_contains'],
}

const SHORTENERS = 'shorteners'

/** SR-10 Settings › Sending rules: workspace rules Send Readiness checks on every send, and the audience defaults. */
export function SendingRulesPage() {
  const t = useT()
  const { slug, me } = useWorkspace()
  const q = $api.useQuery('get', listPath, { params: { path: { workspace: slug } } })
  return (
    <Page title={t(m.settings)} subtitle={me.workspace.name} tabs={settingsTabs(slug)} documentTitle={t(m.documentTitle)}>
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.isPending ? (
        <Loading label={t(m.loading)}>
          <div className="card sr-skel">
            <Skeleton w="40%" h={16} />
            <Skeleton w="90%" />
            <Skeleton w="85%" />
            <Skeleton w="70%" />
          </div>
        </Loading>
      ) : (
        <RulesBody view={q.data} />
      )}
    </Page>
  )
}

function RulesBody({ view }: { view: View }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [editing, setEditing] = useState<Rule | null>(null)
  const [busyRule, setBusyRule] = useState<string>()
  const formRef = useRef<HTMLElement>(null)
  const read = !view.canManage
  const reason = read ? view.reason : undefined
  const refresh = () => qc.invalidateQueries({ queryKey: ['get', listPath] })

  const save = async (rule: Rule, patch: Partial<RuleInput>, done: string) => {
    setBusyRule(rule.id)
    try {
      const body: RuleInput = { ...inputOf(rule), ...patch }
      await http.PUT(rulePath, { params: { path: { workspace: slug, ruleId: rule.id } }, body })
      toast({ kind: 'success', title: done })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: t(m.ruleFailed), detail: errorMessage(e) })
    } finally {
      setBusyRule(undefined)
    }
  }
  const startEdit = (r: Rule | null) => {
    setEditing(r)
    // The form is beside or under the table: bring it into view and move focus to its first field.
    requestAnimationFrame(() => {
      formRef.current?.scrollIntoView({ block: 'start', behavior: 'smooth' })
      formRef.current?.querySelector<HTMLInputElement>('input')?.focus({ preventScroll: true })
    })
  }
  const on = view.rules.filter((r) => r.enabled).length
  const names = view.managers.length ? andList(view.managers) : 'none'
  return (
    <div className="sr-page contents">
      <section className="card" aria-labelledby="sr-rules">
        <div className="sec">
          <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-0.5">
            <h2 className="h2" id="sr-rules">
              {t(m.rulesTitle)}
            </h2>
            <span className="k">{t(m.rulesCaption)}</span>
          </div>
          <Button variant="primary" disabledReason={reason} onClick={() => startEdit(null)}>
            {t(m.addRule)}
          </Button>
        </div>
        {view.rules.length === 0 ? (
          <p className="pad k">{t(m.noRules)}</p>
        ) : (
          <div className="sr-scroll">
            <table className="sr-tab">
              <caption className="vh">{t(m.tableCaption, { count: on })}</caption>
              <thead>
                <tr>
                  <th scope="col">{t(m.colRule)}</th>
                  <th scope="col">{t(m.colSeverity)}</th>
                  <th scope="col">{t(m.colScope)}</th>
                  <th scope="col">{t(m.colStatus)}</th>
                  <th scope="col">{t(m.colTriggered)}</th>
                  <th scope="col">
                    <span className="vh">{t(m.colActions)}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {view.rules.map((r) => (
                  <RuleRow
                    key={r.id}
                    rule={r}
                    read={read}
                    busy={busyRule === r.id}
                    onSeverity={(v) => save(r, { severity: v }, t(m.ruleSaved, { name: r.name }))}
                    onScope={(v) => save(r, { appliesTo: v }, t(m.ruleSaved, { name: r.name }))}
                    onToggle={(v) => save(r, { enabled: v }, t(v ? m.ruleOn : m.ruleOff, { name: r.name }))}
                    onEdit={() => startEdit(r)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
        <div className="pad sr-foot">
          <span className="k">{t(m.applyNote)}</span>
          <span className="k">{t(m.whoCanChange, { names })}</span>
        </div>
      </section>
      <div className="sr-cols">
        <RuleForm
          key={editing?.id ?? 'new'}
          ref={formRef}
          rule={editing}
          read={read}
          reason={reason}
          onDone={async () => {
            setEditing(null)
            await refresh()
          }}
          onCancel={() => setEditing(null)}
        />
        <DefaultsCard key={JSON.stringify(view.defaults)} view={view} read={read} reason={reason} onSaved={refresh} />
      </div>
    </div>
  )
}

function inputOf(r: Rule): RuleInput {
  return { name: r.name, kind: r.kind, operator: r.operator, value: r.value, severity: r.severity, appliesTo: r.appliesTo, enabled: r.enabled, message: r.message }
}

function RuleRow({
  rule: r,
  read,
  busy,
  onSeverity,
  onScope,
  onToggle,
  onEdit,
}: {
  rule: Rule
  read: boolean
  busy: boolean
  onSeverity: (v: Severity) => void
  onScope: (v: Scope) => void
  onToggle: (v: boolean) => void
  onEdit: () => void
}) {
  const t = useT()
  const tr = r.triggered
  return (
    <tr aria-busy={busy || undefined}>
      <td className="sr-rn">
        <span className="font-semibold">{r.name}</span>
        <div className="k">
          {r.description}
          {/* A footer rule for automations: broadcasts get the footer check built in. */}
          {r.kind === 'contains_text' && r.operator === 'footer_contains' && r.appliesTo === 'automation' ? <>. {t(m.builtInNote)}</> : null}
        </div>
      </td>
      <td data-label={t(m.colSeverity)} className="sr-sev">
        <Select aria-label={t(m.severityOf, { name: r.name })} value={r.severity} disabled={read || busy} onChange={(e) => onSeverity(e.target.value as Severity)}>
          <option value="warn">{t(m.warn)}</option>
          <option value="block">{t(m.block)}</option>
        </Select>
      </td>
      <td data-label={t(m.colScope)} className="sr-sc">
        <Select aria-label={t(m.scopeOf, { name: r.name })} value={r.appliesTo} disabled={read || busy} onChange={(e) => onScope(e.target.value as Scope)}>
          <ScopeOptions />
        </Select>
      </td>
      <td data-label={t(m.colStatus)} className="sr-st">
        <label className="sr-stat">
          <input type="checkbox" className="tg" role="switch" checked={r.enabled} disabled={read || busy} aria-label={r.name} onChange={(e) => onToggle(e.target.checked)} />
          <span>{r.enabled ? t(m.on) : t(m.off)}</span>
        </label>
      </td>
      <td data-label={t(m.colTriggered)} className="sr-lt">
        {tr ? (
          <>
            <span>{t(m.dayTime, { day: shortDate(tr.at), time: clock(tr.at) })}</span>
            <div className="k">{tr.fixed ? t(m.triggeredFixed, { campaign: tr.campaign }) : tr.count > 0 ? t(m.triggeredLinks, { campaign: tr.campaign, count: tr.count }) : tr.campaign}</div>
          </>
        ) : (
          <>
            <span>{t(m.never)}</span>
            <div className="k">{r.addedBy ? t(m.addedBy, { day: shortDate(r.addedAt), name: r.addedBy }) : t(m.addedOn, { day: shortDate(r.addedAt) })}</div>
          </>
        )}
      </td>
      <td className="sr-ed">
        <Button size="sm" aria-label={t(m.editRuleNamed, { name: r.name })} disabled={read} onClick={onEdit}>
          {t(common.edit)}
        </Button>
      </td>
    </tr>
  )
}

function ScopeOptions() {
  const t = useT()
  return (
    <>
      <option value="broadcast">{t(m.broadcasts)}</option>
      <option value="automation">{t(m.automations)}</option>
      <option value="both">{t(m.both)}</option>
    </>
  )
}

const kindWords = { link_param: m.kindParam, link_domain: m.kindDomain, subject_length: m.kindSubject, contains_text: m.kindText } as const
const opWords = {
  on_every: m.opOnEvery,
  on_none: m.opOnNone,
  not_one_of: m.opNotOneOf,
  one_of: m.opOneOf,
  at_most: m.opAtMost,
  at_least: m.opAtLeast,
  footer_contains: m.opFooterContains,
  footer_lacks: m.opFooterLacks,
  body_contains: m.opBodyContains,
  subject_contains: m.opSubjectContains,
} as const
const valueWords = { link_param: m.valueParam, link_domain: m.valueDomains, subject_length: m.valueSubject, contains_text: m.valueText } as const
const helpWords = { link_param: m.helpParam, link_domain: m.helpDomain, subject_length: m.helpSubject, contains_text: m.helpText } as const
const placeholders: Record<Kind, string> = { link_param: 'utm_campaign', link_domain: 'bit.ly, t.co, tinyurl.com', subject_length: '70', contains_text: '500 Market St' }

const empty: RuleInput = { name: '', kind: 'link_param', operator: 'on_every', value: '', severity: 'warn', appliesTo: 'both', enabled: true, message: '' }

/** Add a rule, or edit one: the condition, severity, scope and the message senders see, with a preview of the Send Readiness row. */
function RuleForm({
  rule,
  read,
  reason,
  onDone,
  onCancel,
  ref,
}: {
  rule: Rule | null
  read: boolean
  reason?: string
  onDone: () => Promise<void>
  onCancel: () => void
  ref: Ref<HTMLElement>
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const [v, setV] = useState<RuleInput>(rule ? inputOf(rule) : empty)
  const [errors, setErrors] = useState<Partial<Record<'name' | 'value', string>>>({})
  const [busy, setBusy] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const set = (patch: Partial<RuleInput>) => setV((x) => ({ ...x, ...patch }))
  const shorteners = v.kind === 'link_domain' && v.value === SHORTENERS
  const submit = async () => {
    const errs: typeof errors = {}
    if (!v.name.trim()) errs.name = t(m.ruleName)
    if (!v.value.trim()) errs.value = t(valueWords[v.kind])
    setErrors(errs)
    if (Object.keys(errs).length) return
    setBusy(true)
    try {
      if (rule) {
        await http.PUT(rulePath, { params: { path: { workspace: slug, ruleId: rule.id } }, body: v })
        toast({ kind: 'success', title: t(m.ruleSaved, { name: v.name.trim() }) })
      } else {
        await http.POST(listPath, { params: { path: { workspace: slug } }, body: v })
        toast({ kind: 'success', title: t(m.ruleAdded, { name: v.name.trim() }) })
        setV(empty)
      }
      await onDone()
    } catch (e) {
      toast({ kind: 'error', title: t(m.ruleFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const remove = async () => {
    if (!rule) return
    try {
      await http.DELETE(rulePath, { params: { path: { workspace: slug, ruleId: rule.id } } })
      toast({ kind: 'success', title: t(m.ruleDeleted, { name: rule.name }) })
      setConfirmDelete(false)
      await onDone()
    } catch (e) {
      toast({ kind: 'error', title: t(m.deleteFailed), detail: errorMessage(e) })
    }
  }
  return (
    <section className="card" id="add-rule" aria-labelledby="sr-form" ref={ref}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="sr-form">
            {rule ? t(m.formEdit) : t(m.formAdd)}
          </h2>
          <span className="k">{t(m.formCaption)}</span>
        </div>
      </div>
      <fieldset className="pad sr-form" disabled={read || busy}>
        <Field label={t(m.ruleName)} error={errors.name}>
          <Input value={v.name} maxLength={120} placeholder={t(m.ruleNamePlaceholder)} onChange={(e) => set({ name: e.target.value })} />
        </Field>
        <fieldset className="sr-fs">
          <legend>{t(m.condition)}</legend>
          <div className="sr-cond">
            <Field label={t(m.check)}>
              <Select
                value={v.kind}
                onChange={(e) => {
                  const kind = e.target.value as Kind
                  set({ kind, operator: operators[kind][0]!, value: '' })
                }}
              >
                {(Object.keys(operators) as Kind[]).map((k) => (
                  <option key={k} value={k}>
                    {t(kindWords[k])}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label={t(m.operator)}>
              <Select value={v.operator} onChange={(e) => set({ operator: e.target.value as Operator })}>
                {operators[v.kind].map((o) => (
                  <option key={o} value={o}>
                    {t(opWords[o])}
                  </option>
                ))}
              </Select>
            </Field>
            {shorteners ? (
              <Field label={t(valueWords[v.kind])}>
                <Select value={SHORTENERS} onChange={(e) => set({ value: e.target.value === SHORTENERS ? SHORTENERS : '' })}>
                  <option value={SHORTENERS}>{t(m.shortenersValue)}</option>
                  <option value="">{t(m.valueDomains)}</option>
                </Select>
              </Field>
            ) : (
              <Field label={t(valueWords[v.kind])} error={errors.value}>
                <Input
                  value={v.value}
                  inputMode={v.kind === 'subject_length' ? 'numeric' : undefined}
                  placeholder={placeholders[v.kind]}
                  onChange={(e) => set({ value: e.target.value })}
                />
              </Field>
            )}
          </div>
          <span className="k">{t(helpWords[v.kind])}</span>
        </fieldset>
        <fieldset className="sr-fs">
          <legend>{t(m.severity)}</legend>
          <div className="sr-optrow">
            <OptionCard name="sr-sev" value="warn" checked={v.severity === 'warn'} onChange={() => set({ severity: 'warn' })} title={t(m.warn)} description={t(m.warnHelp)} />
            <OptionCard name="sr-sev" value="block" checked={v.severity === 'block'} onChange={() => set({ severity: 'block' })} title={t(m.block)} description={t(m.blockHelp)} />
          </div>
        </fieldset>
        <Field label={t(m.appliesTo)}>
          <Select value={v.appliesTo} onChange={(e) => set({ appliesTo: e.target.value as Scope })}>
            <ScopeOptions />
          </Select>
        </Field>
        <Field label={t(m.message)} hint={t(m.messageHelp)}>
          <Textarea rows={2} value={v.message ?? ''} maxLength={500} placeholder={t(m.messagePlaceholder)} onChange={(e) => set({ message: e.target.value })} />
        </Field>
        <div className="sr-prev">
          <span className={v.severity === 'block' ? 'badge bad' : 'badge warn'}>
            <span className="dot" />
            {v.severity === 'block' ? t(m.block) : t(m.warn)}
          </span>
          <div className="flex min-w-0 flex-col gap-0.5">
            <span className="text-[13px] font-medium">{t(m.previewTitle, { name: v.name.trim() ? lowerFirst(v.name.trim()) : t(m.previewName) })}</span>
            <span className="k">{t(m.previewCaption)}</span>
          </div>
        </div>
        <div className="sr-actions">
          {rule ? (
            <Button variant="danger" className="mr-auto" onClick={() => setConfirmDelete(true)}>
              {t(m.deleteRule)}
            </Button>
          ) : null}
          <Button onClick={onCancel}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} busyLabel={t(m.saving)} disabledReason={reason} onClick={() => void submit()}>
            {rule ? t(m.saveRule) : t(m.addRule)}
          </Button>
        </div>
      </fieldset>
      {rule ? (
        <ConfirmDialog
          open={confirmDelete}
          onOpenChange={setConfirmDelete}
          tone="bad"
          badge={t(m.deleteRule)}
          title={t(m.deleteTitle, { name: rule.name })}
          description={t(m.deleteBody)}
          confirmLabel={t(m.deleteRule)}
          busyLabel={t(m.deleting)}
          onConfirm={remove}
        />
      ) : null}
    </section>
  )
}

function lowerFirst(s: string): string {
  return s.length > 1 && s[1] === s[1]!.toLowerCase() ? s[0]!.toLowerCase() + s.slice(1) : s
}

/** Audience defaults: leave out the unengaged, and when a sender types the count. */
function DefaultsCard({ view, read, reason, onSaved }: { view: View; read: boolean; reason?: string; onSaved: () => Promise<unknown> }) {
  const t = useT()
  const { slug } = useWorkspace()
  const d = view.defaults
  const [exclude, setExclude] = useState(d.excludeUnengaged)
  const [multiple, setMultiple] = useState(decimal(d.typedMultiple, 1))
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const days = view.sunsetDays
  const id = useId()
  const save = async () => {
    const n = Number(multiple.replace(',', '.'))
    if (!Number.isFinite(n) || n < 1.1 || n > 20) {
      setError(t(m.typedRange))
      return
    }
    setError(undefined)
    setBusy(true)
    try {
      await http.PUT(defaultsPath, { params: { path: { workspace: slug } }, body: { excludeUnengaged: exclude, typedMultiple: n } })
      toast({ kind: 'success', title: t(m.defaultsSaved) })
      await onSaved()
    } catch (e) {
      toast({ kind: 'error', title: t(m.defaultsFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const link = (chunks: ReactNode[]) => (
    <Link to="/w/$workspace/audience/health" params={{ workspace: slug }}>
      {chunks}
    </Link>
  )
  return (
    <section className="card" aria-labelledby="sr-defaults">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="sr-defaults">
            {t(m.defaultsTitle)}
          </h2>
          <span className="k">{t(m.defaultsCaption)}</span>
        </div>
      </div>
      <div className="pad sr-defaults">
        <Toggle
          className="sr-trow"
          label={t(m.excludeUnengaged, { days })}
          description={t(m.excludeUnengagedHelp, { days })}
          checked={exclude}
          disabled={read || busy}
          onChange={(e) => setExclude(e.target.checked)}
        />
        <span className="k sr-sunset">
          <T msg={m.editSunset} values={{ link }} />
        </span>
        <div className="sr-trow col">
          <label htmlFor={`${id}-mult`} className="font-semibold">
            {t(m.typedLabel)}
          </label>
          <div className="sr-mult">
            <Input
              id={`${id}-mult`}
              value={multiple}
              inputMode="decimal"
              disabled={read || busy}
              invalid={Boolean(error)}
              aria-describedby={[error ? `${id}-err` : '', `${id}-unit`, `${id}-help`].filter(Boolean).join(' ')}
              onChange={(e) => setMultiple(e.target.value)}
            />
            <span id={`${id}-unit`} className="font-semibold whitespace-nowrap">
              {t(m.typedUnit)}
            </span>
          </div>
          {error ? (
            <span className="err" id={`${id}-err`} role="alert">
              {error}
            </span>
          ) : null}
          <span className="k" id={`${id}-help`}>
            {t(m.typedHelp)}
          </span>
        </div>
        <div className="sr-actions sr-save">
          <Button variant="primary" loading={busy} busyLabel={t(m.saving)} disabledReason={reason} onClick={() => void save()}>
            {t(m.save)}
          </Button>
        </div>
      </div>
    </section>
  )
}
