import { keepPreviousData } from '@tanstack/react-query'
import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Note } from '@/components/ui/Note'
import { StepProp } from '@/features/push/AutomationStep'
import { useSettled } from '@/features/sms/AutomationStep'
import { useT } from '@/i18n'
import { clock, count, dayMonthDate, dayMonthYear, money } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './automation.messages'
import { categoryBadge, FilledBody, statusBadge, type Template } from './common'
import './whatsapp.css'

type Step = Schemas['AutomationWhatsAppStep']
type Var = Schemas['AutomationWhatsAppVar']
type NodeData = Schemas['AutomationNodeData']
type Estimate = Schemas['WhatsAppStepEstimate']

/** What the estimate needs of a step, as a string so a pause in typing settles on equal values. */
function estimateKey(s: Step): string {
  return JSON.stringify({
    consentListId: s.consentListId || undefined,
    mode: s.mode || undefined,
    template: s.template || undefined,
    language: s.language || undefined,
    variables: s.variables?.length ? s.variables : undefined,
    fallbackTemplate: s.fallbackTemplate || undefined,
  })
}

/** The step's templates, reach, cost and preview: one request per step, shared by its panel and its node. */
export function useWhatsAppStepEstimate(s: Step | undefined, enabled = true): Estimate | undefined {
  const { slug } = useWorkspace()
  const key = useSettled(estimateKey(s ?? {}), 400)
  const q = $api.useQuery(
    'post',
    '/api/v1/workspaces/{workspace}/whatsapp/step-estimate',
    { params: { path: { workspace: slug } }, body: JSON.parse(key) as Schemas['WhatsAppStepEstimateInput'] },
    { enabled, placeholderData: keepPreviousData, staleTime: 30_000, retry: false },
  )
  return q.data
}

/** The canvas node's header word: "template" or "free text". */
export function whatsAppNodeHead(s: Step | undefined, t: ReturnType<typeof useT>): string {
  return s?.mode === 'text' ? t(m.headText) : t(m.headTemplate)
}

/** The canvas node's lines for a Send WhatsApp step: who has consent, and the cost or the window rule. */
export function whatsAppNodeLines(s: Step | undefined, t: ReturnType<typeof useT>, est?: Estimate): string[] {
  if (!s?.consentListId || !est?.connected) return []
  const out = [t(m.nodeConsent, { n: count(est.consent) })]
  out.push(s.mode === 'text' && s.outsideWindow !== 'template' ? t(m.nodeWindow) : t(m.nodeCost, { cost: money(est.cost.amount) }))
  return out
}

/** The step panel's subtitle: the account's number once WhatsApp is connected. */
export function WhatsAppStepSub({ step, fallback }: { step: Step | undefined; fallback: string }) {
  const t = useT()
  const est = useWhatsAppStepEstimate(step)
  return <>{est?.business ? t(m.sub, { number: est.business.number }) : fallback}</>
}

/** A rate card's date, read at noon UTC so no time zone moves it to the day before. */
function rateDate(iso: string): string {
  return dayMonthYear(`${iso.slice(0, 10)}T12:00:00Z`)
}

/**
 * The Send WhatsApp step's settings (AutomationChannels › WhatsApp step
 * selected): an approved template with its variables mapped to contact
 * fields, or free text with the window rule; the locked consent rule with
 * who it reaches; quiet hours; the preview as someone on the list; the
 * estimated cost; and what the customer service window means.
 */
export function WhatsAppStepProps({ data: d, lists, editable, onChange }: { data: NodeData; lists: { id: string; name: string }[]; editable: boolean; onChange: (data: NodeData) => void }) {
  const t = useT()
  const s: Step = d.whatsapp ?? {}
  const set = (patch: Partial<Step>) => onChange({ ...d, whatsapp: { ...s, ...patch } })
  const disabled = !editable
  const est = useWhatsAppStepEstimate(s)
  const mode = s.mode === 'text' ? 'text' : 'template'
  const templates = est?.templates ?? []
  const approved = templates.filter((x) => x.status === 'approved')
  const others = templates.filter((x) => x.status !== 'approved')
  const keyOf = (x: Template) => `${x.name}|${x.language}`
  const find = (name?: string, language?: string) => approved.find((x) => x.name === name && (!language || x.language === language))
  const tpl = find(s.template, s.language)
  const fallbackTpl = find(s.fallbackTemplate)
  const listName = lists.find((l) => l.id === s.consentListId)?.name
  const vars = s.variables ?? []
  const varOf = (key: string): Var => vars.find((v) => v.key === key) ?? { key, source: 'field', field: '', fallback: '' }
  const setVar = (key: string, patch: Partial<Var>) =>
    set({ variables: [...vars.filter((v) => v.key !== key), { ...varOf(key), ...patch }].sort((a, b) => Number(a.key) - Number(b.key)) })
  const pickTemplate = (value: string) => {
    const next = approved.find((x) => keyOf(x) === value)
    set(next ? { template: next.name, language: next.language, variables: next.variables.map((v) => varOf(v.key)) } : { template: undefined, language: undefined, variables: [] })
  }
  const option = (x: Template) =>
    x.status === 'approved'
      ? t(m.templateOption, { name: x.name, category: t(categoryBadge[x.category].msg), language: x.languageLabel || x.language })
      : t(m.templateOther, { name: x.name, status: t(statusBadge[x.status].msg) })
  const synced = tpl?.syncedAt
  const until = est?.windowExample?.until
  return (
    <>
      <StepProp title={t(m.name)} id="wa-step-name">
        <input aria-labelledby="wa-step-name" className="in" disabled={disabled} placeholder={t(m.namePlaceholder)} value={d.label ?? ''} onChange={(e) => onChange({ ...d, label: e.target.value || undefined })} />
      </StepProp>
      {est && !est.connected ? (
        <div className="fe-prop">
          <Note tone="warn">{t(m.notConnected)}</Note>
        </div>
      ) : null}
      <StepProp title={t(m.message)} id="wa-step-mode">
        <label className="wa-mode">
          <input type="radio" name="wa-step-mode" checked={mode === 'template'} disabled={disabled} onChange={() => set({ mode: 'template' })} />
          <b>{t(m.modeTemplate)}</b>
          <span className="k">{t(m.modeTemplateHint)}</span>
        </label>
        <label className="wa-mode">
          <input type="radio" name="wa-step-mode" checked={mode === 'text'} disabled={disabled} onChange={() => set({ mode: 'text', outsideWindow: s.outsideWindow || 'skip' })} />
          <b>{t(m.modeText)}</b>
          <span className="k">{t(m.modeTextHint)}</span>
        </label>
      </StepProp>
      {mode === 'template' ? (
        <>
          <StepProp title={t(m.template)} id="wa-step-template">
            <select aria-labelledby="wa-step-template" className="in" disabled={disabled} value={tpl ? keyOf(tpl) : ''} onChange={(e) => pickTemplate(e.target.value)}>
              {!tpl ? <option value="">{t(m.templatePick)}</option> : null}
              {approved.map((x) => (
                <option key={keyOf(x)} value={keyOf(x)}>
                  {option(x)}
                </option>
              ))}
              {others.map((x) => (
                <option key={keyOf(x)} value={keyOf(x)} disabled>
                  {option(x)}
                </option>
              ))}
            </select>
            {tpl ? (
              <span className="flex flex-wrap items-center gap-1.5">
                <Badge tone={statusBadge[tpl.status].tone}>{t(statusBadge[tpl.status].msg)}</Badge>
                {synced ? (
                  <span className="k">
                    {dayMonthDate(synced) === dayMonthDate(new Date().toISOString())
                      ? t(m.synced, { time: clock(synced) })
                      : t(m.syncedDay, { date: dayMonthDate(synced), time: clock(synced) })}
                  </span>
                ) : null}
              </span>
            ) : est?.connected && !approved.length ? (
              <span className="k">{t(m.noTemplates)}</span>
            ) : null}
          </StepProp>
          <StepProp title={t(m.variables)} id="wa-step-vars">
            {tpl && tpl.variables.length ? (
              tpl.variables.map((tv) => {
                const v = varOf(tv.key)
                return (
                  <div key={tv.key} className="wa-svar">
                    <span className="flex min-w-0 items-center gap-1.5">
                      <span className="wa-tok">{`{{${tv.key}}}`}</span>
                      <span className="k truncate">{tv.label}</span>
                    </span>
                    <select
                      className="in"
                      aria-label={t(m.varSourceFor, { key: tv.key })}
                      disabled={disabled}
                      value={v.source === 'text' ? '__text' : (v.field ?? '')}
                      onChange={(e) =>
                        setVar(tv.key, e.target.value === '__text' ? { source: 'text', field: undefined, fallback: undefined } : { source: 'field', field: e.target.value, text: undefined })
                      }
                    >
                      {v.source === 'field' && !v.field ? <option value="">–</option> : null}
                      <option value="__text">{t(m.fixedText)}</option>
                      {(est?.fields ?? []).map((f) => (
                        <option key={f.key} value={f.key}>
                          {t(m.fieldOption, { label: f.label })}
                        </option>
                      ))}
                    </select>
                    <input
                      className="in"
                      aria-label={v.source === 'text' ? t(m.textFor, { key: tv.key }) : t(m.fallbackFor, { key: tv.key })}
                      disabled={disabled}
                      maxLength={500}
                      placeholder={v.source === 'text' ? '' : t(m.fallbackPlaceholder)}
                      value={(v.source === 'text' ? v.text : v.fallback) ?? ''}
                      onChange={(e) => setVar(tv.key, v.source === 'text' ? { text: e.target.value } : { fallback: e.target.value })}
                    />
                  </div>
                )
              })
            ) : (
              <span className="k">{tpl ? t(m.noVariables) : t(m.templatePick)}</span>
            )}
          </StepProp>
        </>
      ) : (
        <>
          <StepProp title={t(m.text)} id="wa-step-text">
            <textarea aria-labelledby="wa-step-text" className="in" rows={4} disabled={disabled} maxLength={1000} placeholder={t(m.textPlaceholder)} value={s.text ?? ''} onChange={(e) => set({ text: e.target.value })} />
          </StepProp>
          <StepProp title={t(m.outsideWindow)} id="wa-step-outwin">
            <label className="fe-chk">
              <input type="radio" name="wa-step-outwin" checked={(s.outsideWindow || 'skip') === 'skip'} disabled={disabled} onChange={() => set({ outsideWindow: 'skip' })} />
              {t(m.outsideWindowSkip)}
            </label>
            <label className="fe-chk">
              <input type="radio" name="wa-step-outwin" checked={s.outsideWindow === 'template'} disabled={disabled} onChange={() => set({ outsideWindow: 'template' })} />
              {t(m.outsideWindowTemplate)}
            </label>
            {s.outsideWindow === 'template' ? (
              <select className="in" aria-label={t(m.fallbackTemplate)} disabled={disabled} value={fallbackTpl?.name ?? ''} onChange={(e) => set({ fallbackTemplate: e.target.value || undefined })}>
                {!fallbackTpl ? <option value="">{t(m.templatePick)}</option> : null}
                {approved.map((x) => (
                  <option key={keyOf(x)} value={x.name}>
                    {option(x)}
                  </option>
                ))}
              </select>
            ) : null}
          </StepProp>
        </>
      )}
      <StepProp title={t(m.consent)} id="wa-step-consent">
        <select aria-labelledby="wa-step-consent" className="in" disabled={disabled} value={s.consentListId ?? ''} onChange={(e) => set({ consentListId: e.target.value || undefined })}>
          {!s.consentListId ? <option value="">{t(m.pickList)}</option> : null}
          {lists.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
        <span className="k">{listName && est ? t(m.consentNote, { list: listName, n: count(est.consent) }) : t(m.consentPick)}</span>
      </StepProp>
      <StepProp title={t(m.outside)} id="wa-step-outside">
        <label className="fe-chk">
          <input type="radio" name="wa-step-outside" checked={(s.outside || 'wait') === 'wait'} disabled={disabled} onChange={() => set({ outside: 'wait' })} />
          {t(m.outsideWait)}
        </label>
        <label className="fe-chk">
          <input type="radio" name="wa-step-outside" checked={s.outside === 'skip'} disabled={disabled} onChange={() => set({ outside: 'skip' })} />
          {t(m.outsideSkip)}
        </label>
        {est ? <span className="k">{est.quiet ? t(m.outsideHint, { window: est.window }) : t(m.quietOff)}</span> : null}
      </StepProp>
      <StepProp title={mode === 'template' && est?.preview ? t(m.previewAs, { name: est.preview.contact }) : t(m.preview)} id="wa-step-preview">
        {mode === 'text' ? (
          <span className="k">{t(m.previewText)}</span>
        ) : est?.preview && tpl ? (
          <>
            <div className="wa-chat">
              <div className="hd">
                <span className="wa-mark" aria-hidden="true">
                  {(est.business?.displayName || 'A').charAt(0).toUpperCase()}
                </span>
                <span className="flex min-w-0 flex-col">
                  <b className="text-[14px]">{est.business?.displayName}</b>
                  <span className="k mono">{est.business?.number}</span>
                </span>
              </div>
              <div className="bd">
                <div className="bub">
                  <FilledBody body={tpl.body} text={est.preview.text} />
                </div>
                {est.preview.buttons.map((b, i) => (
                  <span key={i} className="qr">
                    {b.text}
                  </span>
                ))}
              </div>
            </div>
            <span className="k">{t(m.previewCaption, { category: t(categoryBadge[tpl.category].msg), business: est.business?.displayName ?? '' })}</span>
          </>
        ) : (
          <span className="k">{t(m.previewEmpty)}</span>
        )}
      </StepProp>
      {s.consentListId && est?.connected ? (
        <StepProp title={t(m.cost)} id="wa-step-cost">
          <span className="text-[13px] font-semibold">{t(m.costLine, { amount: money(est.cost.amount), n: est.cost.messages })}</span>
          <span className="k">{mode === 'text' && s.outsideWindow !== 'template' ? t(m.costNoteText) : t(m.costNote, { date: rateDate(est.cost.ratesAsOf) })}</span>
        </StepProp>
      ) : null}
      <StepProp title={t(m.windowTitle)} id="wa-step-window">
        <span className="k">
          {est?.windowExample && until
            ? t(m.windowText, { name: est.windowExample.name, until: `${dayMonthDate(until)} ${clock(until)}` })
            : t(m.windowTextNobody)}
        </span>
      </StepProp>
    </>
  )
}

/** The step's messages over 30 days (Step stats), with the engine's skips. */
export function WhatsAppStepStats({ nodeId, automationId, hasStats, skipped }: { nodeId: string; automationId: string; hasStats: boolean; skipped: number }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations/{id}/whatsapp-stats', { params: { path: { workspace: slug, id: automationId } } }, { enabled: hasStats })
  const st = q.data?.steps.find((x) => x.nodeId === nodeId)
  const ran = Boolean(st && st.sent)
  const cells: { label: string; value: string }[] = [
    { label: t(m.sSent), value: ran ? count(st!.sent) : '–' },
    { label: t(m.sDelivered), value: ran ? count(st!.delivered) : '–' },
    { label: t(m.sRead), value: ran ? count(st!.read) : '–' },
    { label: t(m.sFailed), value: ran ? count(st!.failed) : '–' },
    { label: t(m.sSkipped), value: ran || skipped ? count(skipped) : '–' },
    { label: t(m.sCost), value: ran ? money(st!.cost) : '–' },
  ]
  return (
    <StepProp title={t(m.statsTitle)} id="wa-step-stats">
      {!ran ? <span className="k">{t(m.noMessages)}</span> : null}
      {st?.waiting ? <span className="k">{t(m.waiting, { count: st.waiting })}</span> : null}
      <div className="fe-stats">
        {cells.map((c) => (
          <span key={c.label}>
            {c.label} <b className="n">{c.value}</b>
          </span>
        ))}
      </div>
      {skipped ? <span className="k">{t(m.skippedNote)}</span> : null}
    </StepProp>
  )
}
