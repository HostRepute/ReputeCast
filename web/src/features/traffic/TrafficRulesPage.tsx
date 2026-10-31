import { Link } from '@tanstack/react-router'
import { useId, useMemo, useState, type ReactNode } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { problemReference } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Checkbox, Input, Select, Toggle } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { count, dayMonthDate, clock, duration, pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { mailboxName } from '@/features/live/model'
import { ProvidersTabs, throwIfForbidden } from '@/features/providers/common'
import { m as pm } from '@/features/providers/messages'
import {
  blockActions,
  canaryDelay,
  canaryOn,
  complaintActions,
  deferralActions,
  deferralSlowActions,
  durations,
  gateBounds,
  inBounds,
  lastMatched,
  mailboxKeys,
  numberText,
  parseNumber,
  peopleList,
  replyActions,
  ruleAction,
  ruleDuration,
  rulesOn,
  ruleWhen,
  savedText,
  thresholdBounds,
  toForm,
  unknownPauseActions,
  unknownWarnActions,
  watchHours,
  watchMinutes,
  whenCounts,
  windows,
  type Form,
  type ReplyRule,
  type Rules,
  type ThresholdField,
} from './traffic-model'
import { m } from './traffic.messages'
import './traffic.css'

/** The typed numbers, kept as text while people edit them. */
type Texts = Record<ThresholdField | 'percent' | 'gBounce' | 'gBlock' | 'gDeferral' | 'gComplaint' | 'gSeeds', string>

function textsOf(f: Form): Texts {
  const th = f.thresholds
  const g = f.canary.gates
  return {
    blockBounce: numberText(th.blockBounce),
    unknownWarn: numberText(th.unknownWarn),
    unknownPause: numberText(th.unknownPause),
    deferralSlow: numberText(th.deferralSlow),
    deferral: numberText(th.deferral),
    complaint: numberText(th.complaint, 2),
    percent: numberText(f.canary.percent),
    gBounce: numberText(g.bounce, 1),
    gBlock: numberText(g.blockBounce),
    gDeferral: numberText(g.deferral),
    gComplaint: numberText(g.complaint, 2),
    gSeeds: String(g.seedSpam),
  }
}

const bounds: Record<keyof Texts, readonly [number, number]> = {
  ...thresholdBounds,
  percent: [1, 20],
  gBounce: gateBounds.bounce,
  gBlock: gateBounds.blockBounce,
  gDeferral: gateBounds.deferral,
  gComplaint: gateBounds.complaint,
  gSeeds: [0, 100],
}

/** Sending providers › Traffic rules (ProvidersTrafficRules; C9). */
export function TrafficRulesPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/traffic-rules', { params: { path: { workspace: slug } } })
  throwIfForbidden(q.error)
  if (q.data) return <RulesBody key={q.data.savedAt ?? 'defaults'} data={q.data} onSaved={() => void q.refetch()} />
  return (
    <Page title={t(pm.title)} filters={<ProvidersTabs />}>
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} reference={problemReference(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {t(m.loadFailedBody)}
        </ErrorPanel>
      ) : (
        <Loading label={t(m.loading)} className="card flex flex-col gap-3 p-5">
          <Skeleton w={200} h={16} />
          <Skeleton h={120} />
          <Skeleton h={120} />
        </Loading>
      )}
    </Page>
  )
}

function RulesBody({ data, onSaved }: { data: Rules; onSaved: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const saved = useMemo(() => toForm(data), [data])
  const savedTexts = useMemo(() => textsOf(saved), [saved])
  const [form, setForm] = useState<Form>(saved)
  const [texts, setTexts] = useState<Texts>(savedTexts)
  const [tried, setTried] = useState(false)
  const ro = !data.canManage
  const dirty = JSON.stringify(form) !== JSON.stringify(saved) || JSON.stringify(texts) !== JSON.stringify(savedTexts)
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/traffic-rules', {
    onError: (e) => toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) }),
  })

  // Every typed number in its bounds; each level below the next.
  const errors: Partial<Record<keyof Texts, string>> = {}
  for (const k of Object.keys(bounds) as (keyof Texts)[]) {
    const [min, max] = bounds[k]
    if (!inBounds(texts[k], bounds[k]) || (k === 'gSeeds' && !Number.isInteger(parseNumber(texts[k])))) errors[k] = t(m.pctRange, { min: numberText(min), max: numberText(max) })
  }
  const num = (k: keyof Texts) => parseNumber(texts[k]) ?? 0
  if (!errors.unknownWarn && !errors.unknownPause && num('unknownWarn') >= num('unknownPause')) errors.unknownPause = t(m.levelsOrder)
  if (!errors.deferralSlow && !errors.deferral && num('deferralSlow') >= num('deferral')) errors.deferral = t(m.levelsOrder)
  const problems = Object.keys(errors).length

  const setText = (k: keyof Texts) => (v: string) => setTexts((x) => ({ ...x, [k]: v }))
  const setThreshold = <K extends keyof Form['thresholds']>(k: K, v: Form['thresholds'][K]) => setForm((f) => ({ ...f, thresholds: { ...f.thresholds, [k]: v } }))
  const setCanary = <K extends keyof Form['canary']>(k: K, v: Form['canary'][K]) => setForm((f) => ({ ...f, canary: { ...f.canary, [k]: v } }))
  const setWatch = <K extends keyof Form['watch']>(k: K, v: Form['watch'][K]) => setForm((f) => ({ ...f, watch: { ...f.watch, [k]: v } }))
  const setRules = (rules: ReplyRule[]) => setForm((f) => ({ ...f, replyRules: rules }))

  const submit = () => {
    setTried(true)
    if (problems) {
      toast({ kind: 'error', title: t(m.fixFirst, { count: problems }) })
      return
    }
    const canary = {
      ...form.canary,
      enabled: canaryOn(form.canary),
      percent: num('percent'),
      gates: { bounce: num('gBounce'), blockBounce: num('gBlock'), deferral: num('gDeferral'), complaint: num('gComplaint'), seedSpam: Math.round(num('gSeeds')) },
    }
    const body: Form = {
      thresholds: {
        ...form.thresholds,
        blockBounce: num('blockBounce'),
        unknownWarn: num('unknownWarn'),
        unknownPause: num('unknownPause'),
        deferralSlow: num('deferralSlow'),
        deferral: num('deferral'),
        complaint: num('complaint'),
      },
      // The rule's history is the server's (read only).
      replyRules: form.replyRules.map(({ id, mailbox, contains, count, windowMinutes, action, durationMinutes, enabled, note }) => ({
        id,
        mailbox,
        contains,
        count,
        windowMinutes,
        action,
        durationMinutes,
        enabled,
        note,
      })),
      canary,
      watch: form.watch,
    }
    save.mutate(
      { params: { path: { workspace: slug } }, body },
      {
        onSuccess: () => {
          toast({ kind: 'success', title: t(m.saved), detail: t(m.savedDetail) })
          onSaved()
        },
      },
    )
  }

  const field = (k: keyof Texts) => ({
    value: texts[k],
    onChange: setText(k),
    error: tried || texts[k] !== savedTexts[k] ? errors[k] : undefined,
    disabled: ro,
  })

  return (
    <Page
      title={t(pm.title)}
      filters={<ProvidersTabs />}
      actions={
        <>
          <span className="k" aria-live="polite">
            {savedText(data, dirty)}
          </span>
          {ro ? null : (
            <Button variant="primary" onClick={submit} disabled={!dirty} loading={save.isPending}>
              {t(m.save)}
            </Button>
          )}
        </>
      }
    >
      {ro ? <p className="tr-ro k">{t(m.readOnly)}</p> : null}
      <div className="tr-cols">
        <div className="tr-main">
          <ThresholdsCard form={form} field={field} ro={ro} setThreshold={setThreshold} />
          <RepliesCard rules={form.replyRules} ro={ro} onChange={setRules} />
          <CanaryCard data={data} form={form} field={field} ro={ro} setCanary={setCanary} />
          <AfterCard data={data} form={form} ro={ro} setWatch={setWatch} />
        </div>
        <Aside data={data} />
      </div>
    </Page>
  )
}

type FieldProps = { value: string; onChange: (v: string) => void; error?: string; disabled: boolean }

/** A percent typed in a rule's line, with its error under the line. */
function PctInput({ label, f, id }: { label: string; f: FieldProps; id: string }) {
  return (
    <Input
      id={id}
      className="tr-pct"
      type="text"
      inputMode="decimal"
      value={f.value}
      onChange={(e) => f.onChange(e.target.value)}
      aria-label={label}
      aria-invalid={f.error ? true : undefined}
      aria-describedby={f.error ? `${id}-err` : undefined}
      disabled={f.disabled}
    />
  )
}

function FieldError({ f, id }: { f: FieldProps; id: string }) {
  return f.error ? (
    <span className="err" id={`${id}-err`}>
      {f.error}
    </span>
  ) : null
}

/** A card's heading: title, caption and a badge. */
function SectionHead({ id, title, caption, badge }: { id: string; title: string; caption: ReactNode; badge?: ReactNode }) {
  return (
    <div className="tr-sech">
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <h2 className="h2" id={id}>
          {title}
        </h2>
        <span className="k">{caption}</span>
      </div>
      {badge ? <span className="flex-none">{badge}</span> : null}
    </div>
  )
}

function ThresholdsCard({
  form,
  field,
  ro,
  setThreshold,
}: {
  form: Form
  field: (k: keyof Texts) => FieldProps
  ro: boolean
  setThreshold: <K extends keyof Form['thresholds']>(k: K, v: Form['thresholds'][K]) => void
}) {
  const t = useT()
  const id = useId()
  const th = form.thresholds
  const level = (props: {
    k: ThresholdField
    rule: string
    lvl?: 'first' | 'second'
    unit: string
    window?: { value: number; set: (v: number) => void }
    action: { value: string; options: string[]; set: (v: string) => void }
  }) => {
    const f = field(props.k)
    const fid = `${id}-${props.k}`
    const lvl = props.lvl ?? 'only'
    return (
      <div className="flex flex-col gap-1">
        <div className="tr-line">
          <span>{t(m.at)}</span>
          <PctInput id={fid} f={f} label={t(m.limitLabel, { rule: props.rule, level: lvl })} />
          <span>{props.unit}</span>
          {props.window ? (
            <>
              <Select
                className="tr-win"
                value={String(props.window.value)}
                onChange={(e) => props.window!.set(Number(e.target.value))}
                aria-label={t(m.windowLabel, { rule: props.rule, level: lvl })}
                disabled={ro}
              >
                {windows.map((w) => (
                  <option key={w} value={w}>
                    {t(m.minutes, { minutes: w })}
                  </option>
                ))}
              </Select>
              <span>{t(m.atOneMailbox)}</span>
            </>
          ) : null}
          <span className="tr-arr" aria-hidden="true">
            →
          </span>
          <Select value={props.action.value} onChange={(e) => props.action.set(e.target.value)} aria-label={t(m.actionLabel, { rule: props.rule, level: lvl })} disabled={ro}>
            {props.action.options.map((a) => (
              <option key={a} value={a}>
                {t(m.thresholdAction, { action: a })}
              </option>
            ))}
          </Select>
        </div>
        <FieldError f={f} id={fid} />
      </div>
    )
  }
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <SectionHead
        id={`${id}-h`}
        title={t(m.thresholdsTitle)}
        caption={<T msg={m.thresholdsCaption} values={{ b: (c) => <b>{c}</b> }} />}
        badge={<Badge tone="neutral">{t(m.workspaceDefault)}</Badge>}
      />
      <Rule title={t(m.blockTitle)} body={t(m.blockBody)}>
        {level({
          k: 'blockBounce',
          rule: t(m.blockTitle),
          unit: t(m.pctMailbox),
          action: { value: th.blockAction, options: blockActions, set: (v) => setThreshold('blockAction', v as typeof th.blockAction) },
        })}
      </Rule>
      <Rule title={t(m.unknownTitle)} body={t(m.unknownBody)}>
        {level({
          k: 'unknownWarn',
          rule: t(m.unknownTitle),
          lvl: 'first',
          unit: t(m.pctSend),
          action: { value: th.unknownWarnAction, options: unknownWarnActions, set: (v) => setThreshold('unknownWarnAction', v as typeof th.unknownWarnAction) },
        })}
        {level({
          k: 'unknownPause',
          rule: t(m.unknownTitle),
          lvl: 'second',
          unit: t(m.pctSend),
          action: { value: th.unknownPauseAction, options: unknownPauseActions, set: (v) => setThreshold('unknownPauseAction', v as typeof th.unknownPauseAction) },
        })}
      </Rule>
      <Rule title={t(m.softTitle)} body={t(m.softBody)}>
        {level({
          k: 'deferralSlow',
          rule: t(m.softTitle),
          lvl: 'first',
          unit: t(m.pctOver),
          window: { value: th.deferralSlowWindow, set: (v) => setThreshold('deferralSlowWindow', v as typeof th.deferralSlowWindow) },
          action: { value: th.deferralSlowAction, options: deferralSlowActions, set: (v) => setThreshold('deferralSlowAction', v as typeof th.deferralSlowAction) },
        })}
        {level({
          k: 'deferral',
          rule: t(m.softTitle),
          lvl: 'second',
          unit: t(m.pctOver),
          window: { value: th.deferralWindow, set: (v) => setThreshold('deferralWindow', v as typeof th.deferralWindow) },
          action: { value: th.deferralAction, options: deferralActions, set: (v) => setThreshold('deferralAction', v as typeof th.deferralAction) },
        })}
      </Rule>
      <Rule title={t(m.complaintTitle)} body={t(m.complaintBody)}>
        {level({
          k: 'complaint',
          rule: t(m.complaintTitle),
          unit: t(m.pctMailbox),
          action: { value: th.complaintAction, options: complaintActions, set: (v) => setThreshold('complaintAction', v as typeof th.complaintAction) },
        })}
      </Rule>
      <div className="tr-fns">
        <p className="tr-fn">{t(m.thresholdsNote)}</p>
      </div>
    </section>
  )
}

function Rule({ title, body, children }: { title: string; body: string; children: ReactNode }) {
  return (
    <div className="tr-rule">
      <div className="flex flex-col gap-0.5 text-[13px]">
        <b>{title}</b>
        <span className="k">{body}</span>
      </div>
      {children}
    </div>
  )
}

/** Mailbox provider responses: the rules, then the add form. */
function RepliesCard({ rules, ro, onChange }: { rules: ReplyRule[]; ro: boolean; onChange: (rules: ReplyRule[]) => void }) {
  const t = useT()
  const id = useId()
  const [draft, setDraft] = useState({ mailbox: 'gmail' as ReplyRule['mailbox'], contains: '', count: 1, action: 'slow' as ReplyRule['action'], duration: 30 })
  const [missing, setMissing] = useState(false)
  const [said, setSaid] = useState('')
  const add = () => {
    const contains = draft.contains.trim()
    if (contains.length < 2) {
      setMissing(true)
      return
    }
    onChange([
      ...rules,
      { mailbox: draft.mailbox, contains, count: draft.count, windowMinutes: 10, action: draft.action, durationMinutes: draft.duration, enabled: true, means: t(m.ruleMeansNew) },
    ])
    setDraft({ ...draft, contains: '' })
    setMissing(false)
    setSaid(t(m.ruleAdded))
  }
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <SectionHead
        id={`${id}-h`}
        title={t(m.repliesTitle)}
        caption={t(m.repliesCaption)}
        badge={rules.length ? <Badge tone="neutral">{rulesOn(rules)}</Badge> : undefined}
      />
      {rules.length === 0 ? <p className="tr-rrule k">{t(m.noRules)}</p> : null}
      {rules.map((r, i) => {
        const mbx = mailboxName(r.mailbox)
        return (
          <div className="tr-rrule" key={r.id ?? `new-${i}`}>
            <Toggle
              label={t(m.ruleSwitch, { mailbox: mbx, code: r.contains })}
              hideLabel
              className="tr-swl"
              checked={r.enabled}
              disabled={ro}
              onChange={(e) => onChange(rules.map((x, j) => (j === i ? { ...x, enabled: e.target.checked } : x)))}
            />
            <div className="flex min-w-0 flex-col gap-[3px]">
              <span className="text-[13px]">
                <T msg={m.ruleHead} values={{ mailbox: mbx, code: r.contains, b: (c) => <b>{c}</b>, c: (c) => <span className="mono tr-code">{c}</span> }} />
              </span>
              <span className="text-[13px]">
                {ruleWhen(r)}{' '}
                <span className="tr-arr" aria-hidden="true">
                  →
                </span>{' '}
                <b>{ruleAction(r)}</b> · {ruleDuration(r)}
              </span>
              {r.means ? <span className="k">{r.means}</span> : null}
              <span className="k">{t(m.lastMatched, { when: lastMatched(r) })}</span>
            </div>
            {ro ? null : (
              <Button
                size="sm"
                onClick={() => {
                  onChange(rules.filter((_, j) => j !== i))
                  setSaid(t(m.ruleRemoved))
                }}
                aria-label={t(m.removeRule, { mailbox: mbx, code: r.contains })}
              >
                {t(m.remove)}
              </Button>
            )}
          </div>
        )
      })}
      {ro ? null : (
        <div className="tr-addf" role="group" aria-labelledby={`${id}-add`}>
          <h3 className="h2 text-[14px]" id={`${id}-add`}>
            {t(m.addRule)}
          </h3>
          <div className="tr-addgrid">
            <label className="lbl">
              {t(m.fieldMailbox)}
              <Select value={draft.mailbox} onChange={(e) => setDraft({ ...draft, mailbox: e.target.value as ReplyRule['mailbox'] })}>
                {mailboxKeys.map((k) => (
                  <option key={k} value={k}>
                    {mailboxName(k)}
                  </option>
                ))}
              </Select>
            </label>
            <label className="lbl">
              {t(m.fieldContains)}
              <Input
                className="mono"
                value={draft.contains}
                placeholder={t(m.containsPlaceholder)}
                onChange={(e) => {
                  setDraft({ ...draft, contains: e.target.value })
                  if (missing) setMissing(false)
                }}
                aria-invalid={missing ? true : undefined}
                aria-describedby={`${id}-help`}
                maxLength={120}
              />
            </label>
            <label className="lbl">
              {t(m.fieldWhen)}
              <Select value={String(draft.count)} onChange={(e) => setDraft({ ...draft, count: Number(e.target.value) })}>
                {whenCounts.map((c) => (
                  <option key={c} value={c}>
                    {t(m.whenOption, { count: c })}
                  </option>
                ))}
              </Select>
            </label>
            <label className="lbl">
              {t(m.fieldAction)}
              <Select value={draft.action} onChange={(e) => setDraft({ ...draft, action: e.target.value as ReplyRule['action'] })}>
                {replyActions.map((a) => (
                  <option key={a} value={a}>
                    {t(m.actionOption, { action: a })}
                  </option>
                ))}
              </Select>
            </label>
            <label className="lbl">
              {t(m.fieldDuration)}
              <Select value={String(draft.duration)} onChange={(e) => setDraft({ ...draft, duration: Number(e.target.value) })}>
                {durations.map((d) => (
                  <option key={d} value={d}>
                    {t(m.durationOption, { kind: d > 0 ? 'for' : 'until', duration: d > 0 ? duration(d) : '' })}
                  </option>
                ))}
              </Select>
            </label>
          </div>
          <div className="flex flex-wrap items-center gap-2.5">
            {missing ? (
              <span className="k text-bad-fg" id={`${id}-help`} role="alert">
                {t(m.containsNeeded)}
              </span>
            ) : (
              <span className="k" id={`${id}-help`}>
                {t(m.containsHelp)}
              </span>
            )}
            <Button className="ml-auto" onClick={add}>
              {t(m.addRule)}
            </Button>
          </div>
          <span className="vh" aria-live="polite">
            {said}
          </span>
        </div>
      )}
    </section>
  )
}

function CanaryCard({
  data,
  form,
  field,
  ro,
  setCanary,
}: {
  data: Rules
  form: Form
  field: (k: keyof Texts) => FieldProps
  ro: boolean
  setCanary: <K extends keyof Form['canary']>(k: K, v: Form['canary'][K]) => void
}) {
  const t = useT()
  const id = useId()
  const c = form.canary
  const on = canaryOn(c)
  const gate = (k: 'gBounce' | 'gBlock' | 'gDeferral' | 'gComplaint', label: string) => {
    const f = field(k)
    const fid = `${id}-${k}`
    return (
      <div className="flex flex-col gap-1">
        <div className="tr-gate">
          <span>{label}</span>
          <span className="tr-gi">
            <PctInput id={fid} f={f} label={t(m.gateLabel, { gate: label })} />%
          </span>
        </div>
        <FieldError f={f} id={fid} />
      </div>
    )
  }
  const size = field('percent')
  const seedsF = field('gSeeds')
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <SectionHead
        id={`${id}-h`}
        title={t(m.canaryTitle)}
        caption={t(m.canaryCaption)}
        badge={
          <Badge tone={on ? 'ok' : 'neutral'} dot>
            {on ? t(m.canaryOn) : t(m.canaryOff)}
          </Badge>
        }
      />
      <div className="flex flex-col gap-4 px-5 py-4">
        <fieldset className="tr-fs">
          <legend>{t(m.runWhen)}</legend>
          <Checkbox
            label={<span className="font-semibold">{t(m.reasonNewDomain)}</span>}
            hint={t(m.reasonNewDomainHint)}
            checked={c.newDomain}
            disabled={ro}
            onChange={(e) => setCanary('newDomain', e.target.checked)}
          />
          <Checkbox
            label={<span className="font-semibold">{t(m.reasonNeverMailed)}</span>}
            hint={t(m.reasonNeverMailedHint)}
            checked={c.neverMailed}
            disabled={ro}
            onChange={(e) => setCanary('neverMailed', e.target.checked)}
          />
          <Checkbox
            label={<span className="font-semibold">{t(m.reasonOverUsual)}</span>}
            hint={data.usual > 0 ? t(m.reasonOverUsualHint, { usual: count(data.usual) }) : t(m.reasonOverUsualNone)}
            checked={c.overUsual}
            disabled={ro}
            onChange={(e) => setCanary('overUsual', e.target.checked)}
          />
        </fieldset>
        <div className="tr-cgrid">
          <div className="lbl">
            <span id={`${id}-size`}>{t(m.size)}</span>
            <div className="flex items-center gap-2">
              <Input
                id={`${id}-percent`}
                className="tr-pct"
                type="text"
                inputMode="decimal"
                value={size.value}
                onChange={(e) => size.onChange(e.target.value)}
                aria-labelledby={`${id}-size ${id}-size-u`}
                aria-invalid={size.error ? true : undefined}
                aria-describedby={size.error ? `${id}-percent-err` : undefined}
                disabled={ro}
              />
              <span id={`${id}-size-u`} className="font-medium text-ink">
                {t(m.sizeUnit)}
              </span>
            </div>
            <FieldError f={size} id={`${id}-percent`} />
          </div>
          {data.seeds > 0 ? (
            <Checkbox className="self-center" label={t(m.seeds, { count: data.seeds })} checked={c.seeds} disabled={ro} onChange={(e) => setCanary('seeds', e.target.checked)} />
          ) : (
            <span className="k self-center">{t(m.seedsNone)}</span>
          )}
          <label className="lbl">
            {t(m.watchTime)}
            <Select value={String(c.watchMinutes)} onChange={(e) => setCanary('watchMinutes', Number(e.target.value) as typeof c.watchMinutes)} disabled={ro}>
              {watchMinutes.map((w) => (
                <option key={w} value={w}>
                  {t(m.minutes, { minutes: w })}
                </option>
              ))}
            </Select>
          </label>
        </div>
        <fieldset className="tr-fs">
          <legend>{t(m.gates)}</legend>
          <div className="tr-ggrid">
            {gate('gBounce', t(m.gateBounce))}
            {gate('gBlock', t(m.gateBlock))}
            {gate('gDeferral', t(m.gateDeferral))}
            {gate('gComplaint', t(m.gateComplaint))}
            {data.seeds > 0 ? (
              <div className="flex flex-col gap-1">
                <div className="tr-gate">
                  <span>{t(m.gateSeeds)}</span>
                  <span className="tr-gi">
                    <Input
                      id={`${id}-seeds`}
                      className="tr-pct"
                      type="text"
                      inputMode="numeric"
                      value={seedsF.value}
                      onChange={(e) => seedsF.onChange(e.target.value)}
                      aria-label={t(m.gateSeedsLabel)}
                      aria-invalid={seedsF.error ? true : undefined}
                      aria-describedby={seedsF.error ? `${id}-seeds-err` : undefined}
                      disabled={ro}
                    />
                    {t(m.gateSeedsOf, { count: data.seeds })}
                  </span>
                </div>
                <FieldError f={seedsF} id={`${id}-seeds`} />
              </div>
            ) : null}
          </div>
        </fieldset>
        <p className="tr-fn">{t(m.canaryNote, { minutes: canaryDelay(c) })}</p>
      </div>
    </section>
  )
}

function AfterCard({
  data,
  form,
  ro,
  setWatch,
}: {
  data: Rules
  form: Form
  ro: boolean
  setWatch: <K extends keyof Form['watch']>(k: K, v: Form['watch'][K]) => void
}) {
  const t = useT()
  const id = useId()
  const w = form.watch
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <SectionHead id={`${id}-h`} title={t(m.afterTitle)} caption={t(m.afterCaption)} />
      <div className="flex flex-col gap-3.5 px-5 py-4">
        <label className="lbl max-w-[260px]">
          {t(m.watchFor)}
          <Select value={String(w.hours)} onChange={(e) => setWatch('hours', Number(e.target.value) as typeof w.hours)} disabled={ro}>
            {watchHours.map((h) => (
              <option key={h} value={h}>
                {t(m.hours, { hours: h })}
              </option>
            ))}
          </Select>
        </label>
        <div className="tr-swrow">
          <Toggle
            label={t(m.alertTitle)}
            description={data.resumers.length ? t(m.alertWho, { names: peopleList(data.resumers) }) : t(m.alertWhoNobody)}
            checked={w.alert}
            disabled={ro}
            onChange={(e) => setWatch('alert', e.target.checked)}
            className="tr-switch"
          />
        </div>
        <div className="tr-swrow">
          <Toggle
            label={t(m.postmasterTitle)}
            description={t(m.postmasterBody, { limit: pct(form.thresholds.complaint, 2) })}
            checked={w.postmaster}
            disabled={ro}
            onChange={(e) => setWatch('postmaster', e.target.checked)}
            className="tr-switch"
          />
        </div>
        <p className="tr-fn">{t(m.lateNote)}</p>
      </div>
    </section>
  )
}

/** How the rules apply, the live monitor's states, and what the rules did lately. */
function Aside({ data }: { data: Rules }) {
  const t = useT()
  const { slug } = useWorkspace()
  const names = data.resumers.length ? peopleList(data.resumers) : ''
  const legend: [string, 'neutral' | 'ok' | 'warn' | 'bad', string][] = [
    [t(m.legendCollecting), 'neutral', t(m.legendCollectingBody)],
    [t(m.legendWatching), 'ok', t(m.legendWatchingBody)],
    [t(m.legendNear), 'warn', t(m.legendNearBody)],
    [t(m.legendTripped), 'bad', t(m.legendTrippedBody)],
  ]
  return (
    <aside className="tr-aside" aria-label={t(m.asideLabel)}>
      <section className="card flex flex-col gap-2.5 px-5 py-[18px]">
        <h2 className="h2">{t(m.howTitle)}</h2>
        <ol className="tr-how">
          <li>{t(m.how1)}</li>
          <li>{t(m.how2)}</li>
          <li>{t(m.how3)}</li>
          <li>{names ? t(m.how4, { names }) : t(m.how4Nobody)}</li>
          <li>
            <T
              msg={m.how5}
              values={{
                a: (c) => (
                  <Link to="/w/$workspace/providers/events" params={{ workspace: slug }}>
                    {c}
                  </Link>
                ),
              }}
            />
          </li>
        </ol>
      </section>
      <section className="card flex flex-col gap-2.5 px-5 py-[18px]">
        <h2 className="h2">{t(m.legendTitle)}</h2>
        {legend.map(([label, tone, body]) => (
          <div key={label} className="tr-lg">
            <Badge tone={tone} dot>
              {label}
            </Badge>
            <span className="k">{body}</span>
          </div>
        ))}
      </section>
      <section className="card flex flex-col">
        <div className="flex flex-col gap-0.5 border-b border-line-2 px-5 py-3.5">
          <h2 className="h2">{t(m.recentTitle)}</h2>
          <span className="k">{t(m.recentCaption)}</span>
        </div>
        {data.recent.length === 0 ? <p className="tr-ra k">{t(m.recentNone)}</p> : null}
        {data.recent.map((r) => (
          <div key={`${r.at}-${r.campaignId}-${r.kind}`} className="tr-ra">
            <span className="mono k">
              {dayMonthDate(r.at)} {clock(r.at)}
            </span>
            <span>{r.text}</span>
            <Link className="k" to="/w/$workspace/campaigns/$campaignId/live" params={{ workspace: slug, campaignId: r.campaignId }}>
              {r.campaign}
            </Link>
          </div>
        ))}
        <div className="border-t border-line-2 px-5 py-1">
          <Link className="tr-ra-link" to="/w/$workspace/providers/events" params={{ workspace: slug }}>
            {t(m.allEvents)}
          </Link>
        </div>
      </section>
    </aside>
  )
}
