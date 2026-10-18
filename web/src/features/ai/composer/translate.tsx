import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { useComposer } from '@/features/composer/context'
import { nameIn, useCampaignLanguages, useLanguagesCache, type Lang, type Langs } from '@/features/languages/composer'
import { useLanguageNav, versionLanguage } from '@/features/languages/composerParts'
import '@/features/languages/languages.css'

import { estimateMicros } from '../format'
import { m, ms } from '../messages'
import { AINote } from '../parts'
import { Refusal, retryable, RetryButton, sendingText, sentText, Sparkle, useOutcome, Writing, type Assist } from '../suggest'
import { useComposerAssist, type Translation } from './state'

type Block = Schemas['AITranslationBlock']
const marksPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/ai-translations' as const

/** The language versions written by AI that nobody has reviewed yet. */
export function useAIMarks(campaignId: string): Schemas['AITranslationMark'][] {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', marksPath, { params: { path: { workspace: slug, campaignId } } }, { staleTime: 30_000 })
  return q.data?.items ?? []
}

/** Why "Translate … with AI" can't be used now, or undefined when it can. */
function translateBlocked(t: ReturnType<typeof useT>, a: Assist): string | undefined {
  if (!a.configured) return t(ms.translateOptionNone)
  if (!a.featureOn('translate')) return t(ms.translateOff)
  return undefined
}

/**
 * "Translate Spanish with AI" in the language menu (AIComposer · the
 * language menu; AIStates · a feature turned off): the version open, or
 * the workspace languages without an up-to-date version.
 */
export function TranslateMenuItems({ data, current, close }: { data: Langs; current: string; close: () => void }) {
  const ca = useComposerAssist()
  const t = useT()
  if (!ca || !ca.assist.can || !data.canEdit) return null
  const a = ca.assist
  const blocked = translateBlocked(t, a)
  const onSource = current === data.source
  const targets: Lang[] = onSource
    ? data.versions.filter((v) => v.offered && (v.status === 'missing' || v.status === 'outdated')).slice(0, 3)
    : data.versions.filter((v) => v.language === current)
  return (
    <>
      {targets.map((v) => (
        <button
          key={v.language}
          type="button"
          disabled={Boolean(blocked)}
          aria-describedby={blocked ? `ai-tr-${v.language}` : undefined}
          className="ai-tr-item"
          onClick={() => {
            close()
            ca.translate(v.language)
          }}
        >
          <Sparkle color={blocked ? '#9CA3AF' : 'var(--accent)'} />
          <span className="flex min-w-0 flex-col items-start">
            <span>{t(ms.translateWith, { language: v.name })}</span>
            {blocked ? (
              <span className="k" id={`ai-tr-${v.language}`}>
                {blocked}
              </span>
            ) : null}
          </span>
        </button>
      ))}
    </>
  )
}

/** The Add language dialog's "Translate with AI" (AIComposer · Add language). */
export function useAddLanguageAI(): { title: string; detail: string; disabledReason?: string; onTranslate: (language: string) => void } | undefined {
  const ca = useComposerAssist()
  const t = useT()
  if (!ca || !ca.assist.can) return undefined
  const a = ca.assist
  return {
    title: t(ms.translateOption),
    detail: t(ms.translateOptionSub, { model: a.model || a.provider }),
    disabledReason: translateBlocked(t, a),
    onTranslate: (language) => ca.translate(language),
  }
}

/** "Translate German with AI" in the compare view's footer, for the version compared. */
export function CompareTranslateButton({ language, name }: { language: string; name: string }) {
  const ca = useComposerAssist()
  const t = useT()
  if (!ca || !ca.assist.can) return null
  const blocked = translateBlocked(t, ca.assist)
  return (
    <Button icon={undefined} className="ai-assist" disabledReason={blocked} onClick={() => ca.translate(language)}>
      <Sparkle />
      {t(ms.translateWith, { language: name })}
    </Button>
  )
}

/** Above the canvas on a version written by AI: "Translated by AI · not reviewed" and Mark reviewed. */
export function AIReviewNote() {
  const c = useComposer()
  const { slug } = useWorkspace()
  const t = useT()
  const qc = useQueryClient()
  const lang = versionLanguage(c.campaign)
  const marks = useAIMarks(c.campaign.id)
  const langs = useCampaignLanguages(c.campaign.id)
  const [busy, setBusy] = useState(false)
  const mark = marks.find((x) => x.language === lang && !x.reviewed)
  if (!lang || !mark || c.kind === 'transactional') return null
  const name = nameIn(langs.data, lang)
  const review = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/ai-reviewed', { params: { path: { workspace: slug, campaignId: c.campaign.id, language: lang } } })
      await qc.invalidateQueries({ queryKey: ['get', marksPath] })
      toast({ kind: 'success', title: t(ms.markedReviewed, { language: name }) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <AINote tone="info" className="ai-review">
      <div className="flex flex-col items-start gap-2">
        <span>{t(ms.notReviewedNote, { language: name })}</span>
        {c.canEdit ? (
          <Button size="sm" loading={busy} onClick={review}>
            {t(ms.markReviewed, { language: name })}
          </Button>
        ) : null}
      </div>
    </AINote>
  )
}

const FIRST_ROWS = 2

/** A segment's HTML as the reader sees it. */
function textOf(html: string): string {
  // A block's end is a space for the reader ("Beta corner Flag rules…").
  const spaced = html.replace(/<\/(?:p|div|li|h[1-6]|td|th|blockquote)>|<br\s*\/?>/gi, (tag) => `${tag} `)
  const d = new DOMParser().parseFromString(`<div>${spaced}</div>`, 'text/html')
  return (d.body.textContent ?? '').replace(/\s+/g, ' ').trim()
}

function key(b: Block): string {
  return `b:${b.index}`
}

/**
 * Translate this variant (AIComposer · the compare view, all tokens kept):
 * the source next to the suggestion, block by block, with Accept block,
 * Edit, Accept all, Reject all and Try again. Tokens and links the model
 * changed are restored from the source, and the row says so.
 */
export function AITranslateView() {
  const ca = useComposerAssist()!
  const c = useComposer()
  const { slug } = useWorkspace()
  const t = useT()
  const outcome = useOutcome()
  const go = useLanguageNav()
  const langs = useCampaignLanguages(c.campaign.id)
  const cache = useLanguagesCache(c.campaign.id)
  const qc = useQueryClient()
  const lang = ca.translating ?? ''
  const s = ca.translation.state
  const tr: Translation | undefined = s.phase === 'done' ? s.data : undefined
  const [accepted, setAccepted] = useState<{ id?: string; keys: Set<string> }>({ keys: new Set() })
  const [edits, setEdits] = useState<Record<string, string>>({})
  const [editing, setEditing] = useState<string | null>(null)
  const [saving, setSaving] = useState<string | null>(null)
  const [all, setAll] = useState(false)
  const data = langs.data
  const source = nameIn(data, data?.source ?? c.campaign.sourceLanguage ?? 'en')
  const name = nameIn(data, lang)
  const version = data?.versions.find((v) => v.language === lang)
  // Acceptances belong to one answer: Try again starts over.
  const keys = accepted.id && tr?.requestId === accepted.id ? accepted.keys : new Set<string>()
  const items = tr ? tr.blocks.length + (tr.subjectSuggestion ? 1 : 0) + (tr.preheaderSuggestion ? 1 : 0) : 0
  // The version open on the canvas is the one being translated: its canvas reloads after an accept.
  const onCanvas = versionLanguage(c.campaign) === lang
  const leave = (open?: string) => {
    ca.translate(null)
    if (!open) return
    if (open === versionLanguage(c.campaign)) void c.reloadFromServer()
    else void go(open)
  }
  const save = async (next: Set<string>, which: string) => {
    if (!tr?.requestId) return
    setSaving(which)
    try {
      // Edits waiting on the canvas go first, so the accept writes on top of them.
      if (onCanvas && c.canEdit && c.status.kind === 'dirty') await c.save({ quiet: true })
      const edited = (k: string) => edits[k] !== undefined
      const out = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/ai-translation', {
        params: { path: { workspace: slug, campaignId: c.campaign.id, language: lang } },
        body: {
          requestId: tr.requestId,
          sourceVersion: tr.sourceVersion,
          subject: next.has('subject') ? (edits.subject ?? tr.subjectSuggestion) : undefined,
          preheader: next.has('preheader') ? (edits.preheader ?? tr.preheaderSuggestion) : undefined,
          blocks: tr.blocks
            .filter((b) => next.has(key(b)))
            .map((b) => ({
              index: b.index,
              segments: b.segments.map((sg, i) => ({ path: sg.path, kind: sg.kind, text: edits[`${key(b)}:${i}`] ?? sg.suggestion })),
              edited: b.segments.some((_, i) => edited(`${key(b)}:${i}`)),
            })),
          items,
          edited: edited('subject') || edited('preheader'),
        },
      })
      setAccepted({ id: tr.requestId, keys: next })
      void cache.refresh()
      void qc.invalidateQueries({ queryKey: ['get', marksPath] })
      if (out.data?.status === 'up_to_date') {
        toast({ kind: 'success', title: t(ms.translationAccepted, { language: name }), detail: t(ms.translationAcceptedDetail, { accepted: out.data.accepted, items: out.data.items }) })
        leave(lang)
      }
    } catch (e) {
      toast({ kind: 'error', title: t(ms.translateFailed), detail: errorMessage(e) })
    } finally {
      setSaving(null)
    }
  }
  const acceptOne = (k: string) => {
    const next = new Set(keys)
    next.add(k)
    setEditing(null)
    void save(next, k)
  }
  const acceptAll = () => {
    if (!tr) return
    const next = new Set<string>(tr.blocks.map(key))
    if (tr.subjectSuggestion) next.add('subject')
    if (tr.preheaderSuggestion) next.add('preheader')
    setEditing(null)
    void save(next, 'all')
  }
  const rejectAll = () => {
    if (tr?.requestId && !keys.size) outcome(tr.requestId, 'rejected')
    if (!keys.size) toast({ title: t(ms.translationRejected) })
    leave(keys.size ? lang : undefined)
  }
  const choices = (data?.versions ?? []).filter((v) => v.status !== 'source' && v.offered)
  const prices = ca.assist.status?.prices ?? { input: 0, output: 0 }
  const sentLine = tr
    ? sentText(t, ms.sentTranslate, tr.sent, { source, language: name })
    : sendingText(t, ms.sentTranslate, ca.assist.status, estimateMicros(4000, 1400, prices), { source, language: name })
  const rows: ReactNode[] = []
  if (tr) {
    const row = (k: string, label: ReactNode, badges: ReactNode, src: string, suggestion: ReactNode, editor: ReactNode) => {
      const isAccepted = keys.has(k)
      return (
        <div key={k} className="lng-cmpr">
          <div className="rh">
            {label}
            {badges}
            <span className="ra">
              {isAccepted ? (
                <Badge tone="ok" dot>
                  {t(ms.acceptedBlock)}
                </Badge>
              ) : (
                <>
                  <Button size="sm" variant="primary" loading={saving === k} onClick={() => acceptOne(k)} disabledReason={saving ? t(m.writing) : undefined}>
                    {t(ms.acceptBlock)}
                  </Button>
                  <Button size="sm" aria-pressed={editing === k} onClick={() => setEditing(editing === k ? null : k)}>
                    {editing === k ? t(ms.doneEditing) : t(ms.editBlock)}
                  </Button>
                </>
              )}
            </span>
          </div>
          <div className="lng-cmpc">
            <div className="src">
              <span className="tl">{t(ms.sourceHeadAi, { source, version: tr.sourceVersion })}</span>
              {src}
            </div>
            <div>
              <span className="tl">{name}</span>
              {editing === k && !isAccepted ? editor : <div className="ai-sugbox" lang={lang}>{suggestion}</div>}
            </div>
          </div>
        </div>
      )
    }
    const aiBadge = <Badge tone="info">{t(ms.aiSuggestion)}</Badge>
    const textEditor = (k: string, value: string) => (
      <textarea className="in ai-sugedit" rows={2} value={edits[k] ?? value} onChange={(e) => setEdits({ ...edits, [k]: e.target.value })} lang={lang} />
    )
    if (tr.subjectSuggestion) rows.push(row('subject', <span className="bn">{t(ms.subjectRow)}</span>, aiBadge, tr.subject, edits.subject ?? tr.subjectSuggestion, textEditor('subject', tr.subjectSuggestion)))
    if (tr.preheaderSuggestion) rows.push(row('preheader', <span className="bn">{t(ms.preheaderRow)}</span>, aiBadge, tr.preheader, edits.preheader ?? tr.preheaderSuggestion, textEditor('preheader', tr.preheaderSuggestion)))
    tr.blocks.forEach((b, n) => {
      const k = key(b)
      const badges = (
        <>
          {aiBadge}
          {b.tokens ? <Badge tone="ok">{t(ms.tokenKeptBadge, { count: b.tokens - b.tokensRestored })}</Badge> : null}
          {b.links ? <Badge tone="ok">{b.buttons && b.buttons === b.links ? t(ms.answerLinksKept) : t(ms.linkKeptBadge, { count: b.links - b.linksRestored })}</Badge> : null}
          {b.tokensRestored + b.linksRestored ? <Badge tone="warn">{t(ms.restoredBadge, { count: b.tokensRestored + b.linksRestored })}</Badge> : null}
        </>
      )
      // As text, the way the reader sees it; the HTML (links, tokens) is what Accept writes.
      const changed = b.segments.some((_, i) => edits[`${k}:${i}`] !== undefined)
      const suggestion = changed ? b.segments.map((sg, i) => textOf(edits[`${k}:${i}`] ?? sg.suggestion)).join(' · ') : b.suggestionText
      const editor = (
        <div className="flex flex-col gap-1.5">
          {b.segments.map((sg, i) => (
            <textarea key={i} className="in ai-sugedit" rows={sg.suggestion.length > 90 ? 3 : 1} value={edits[`${k}:${i}`] ?? sg.suggestion} onChange={(e) => setEdits({ ...edits, [`${k}:${i}`]: e.target.value })} lang={lang} />
          ))}
        </div>
      )
      rows.push(
        row(
          k,
          <>
            <span className="k mono" style={{ minWidth: 18 }}>
              {n + 1}
            </span>
            <span className="bn">{b.name}</span>
          </>,
          badges,
          b.sourceText,
          suggestion || b.suggestionText,
          editor,
        ),
      )
    })
  }
  const shown = all ? rows : rows.slice(0, (tr?.subjectSuggestion ? 1 : 0) + (tr?.preheaderSuggestion ? 1 : 0) + FIRST_ROWS)
  const hidden = rows.length - shown.length
  return (
    <div className="lng-cmp-wrap" id="main">
      <div className="card lng-cmp">
        <section aria-label={t(ms.translateTitle, { language: name })} className="flex min-w-0 flex-1 flex-col bg-white">
          <div className="lng-cmp-head">
            <h2 className="m-0 flex flex-wrap items-center gap-2" style={{ fontSize: 16, fontWeight: 700 }}>
              <T msg={ms.compareWith} values={{ source, k: (x) => <span className="k" style={{ fontSize: 13 }}>{x}</span> }} />
              <select className="in" aria-label={t(ms.compareLanguage)} style={{ height: 32, fontWeight: 600, width: 'auto' }} value={lang} onChange={(e) => ca.translate(e.target.value)}>
                {choices.map((v) => (
                  <option key={v.language} value={v.language}>
                    {t(ms.languageStatus, { language: v.name, status: v.status })}
                  </option>
                ))}
              </select>
            </h2>
            {tr && !tr.refusal ? (
              <span className="ai-trby">
                <Sparkle />
                {t(ms.translatedBy, { model: ca.assist.model, count: tr.blocks.length })}
              </span>
            ) : null}
            <span className="ml-auto flex flex-wrap gap-2">
              {tr && !tr.refusal ? (
                <>
                  <Button size="sm" variant="primary" loading={saving === 'all'} onClick={acceptAll}>
                    {t(ms.acceptAll)}
                  </Button>
                  <Button size="sm" onClick={rejectAll}>
                    {t(ms.rejectAll)}
                  </Button>
                </>
              ) : null}
              {s.phase === 'error' ? (
                retryable(s.error) ? <RetryButton error={s.error} onRetry={ca.translation.retry} /> : null
              ) : s.phase !== 'working' ? (
                <Button size="sm" icon="refresh" onClick={ca.translation.retry}>
                  {t(m.tryAgain)}
                </Button>
              ) : null}
              <Button size="sm" onClick={() => leave(keys.size ? lang : undefined)}>
                {s.phase === 'working' ? t(m.cancel) : t(m.close)}
              </Button>
            </span>
          </div>
          {s.phase === 'working' ? (
            <div className="pad flex flex-col gap-3">
              <span className="font-semibold">{t(ms.translating, { source, language: name })}</span>
              <Writing />
              <span className="k">{t(ms.translatingHint)}</span>
            </div>
          ) : s.phase === 'error' ? (
            <div className="pad">
              <AINote tone={s.error.kind === 'rate' ? 'warn' : 'bad'}>{s.error.message}</AINote>
            </div>
          ) : tr?.refusal ? (
            <div className="pad flex flex-col gap-2">
              <Refusal model={ca.assist.model} text={tr.refusal} />
            </div>
          ) : tr ? (
            <>
              <div className="ai-trchecks">
                <span>
                  <Icon name="check" size={13} strokeWidth={2.2} />
                  {tr.tokensRestored + tr.linksRestored
                    ? t(ms.restored, { count: tr.tokensRestored + tr.linksRestored })
                    : t(ms.allKept, { tokens: tr.tokens, links: tr.links })}
                </span>
                {tr.subjectSuggestion ? (
                  <span className={tr.subjectHeuristics.length ? 'ai-no' : undefined}>
                    <Icon name={tr.subjectHeuristics.length ? 'alertCircle' : 'check'} size={13} strokeWidth={2.2} />
                    {tr.subjectHeuristics.length ? t(ms.subjectWarn, { heuristics: tr.subjectHeuristics.join(', ') }) : t(ms.subjectChecked)}
                  </span>
                ) : null}
                {tr.readers !== undefined && tr.readers !== null ? (
                  <span>
                    <Icon name="check" size={13} strokeWidth={2.2} />
                    {t(ms.readersGet, { count: tr.readers, language: name })}
                  </span>
                ) : null}
              </div>
              <div className="lng-cmph">
                <div>{t(ms.sourceHeadAi, { source, version: tr.sourceVersion })}</div>
                <div>{version?.status === 'missing' || !version ? t(ms.versionHeadMissing, { language: name }) : t(ms.versionHeadReplaces, { language: name })}</div>
              </div>
              {shown}
              {hidden > 0 ? (
                <div className="ai-trmore">
                  <Icon name="chevronDown" size={14} />
                  <span>{t(ms.moreBlocks, { count: hidden, accepted: keys.size, total: items })}</span>
                  <Button size="sm" className="ml-auto" onClick={() => setAll(true)}>
                    {t(ms.showAll)}
                  </Button>
                </div>
              ) : null}
            </>
          ) : null}
          {/* An error card says what happened instead, as the other cards do. */}
          {ca.assist.status && s.phase !== 'error' ? <div className="ai-sug-f">{sentLine}</div> : null}
        </section>
      </div>
    </div>
  )
}

/** The status bar under the AI compare view: "Translate with AI › English to Spanish". */
export function AITranslateFooter() {
  const ca = useComposerAssist()
  const c = useComposer()
  const langs = useCampaignLanguages(c.campaign.id)
  const t = useT()
  if (!ca?.translating) return null
  const source = nameIn(langs.data, langs.data?.source ?? c.campaign.sourceLanguage ?? 'en')
  return (
    <footer className="cmp-foot">
      <div className="bc">
        <span>{t(ms.translateOption)}</span>
        <span aria-hidden="true">›</span>
        <span>{t(ms.translateCrumb, { source, language: nameIn(langs.data, ca.translating) })}</span>
      </div>
    </footer>
  )
}
