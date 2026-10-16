import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { dayMonthDate, clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { nameIn, useLanguagesCache, type Langs } from './composer'
import { mz } from './messages'
import './languages.css'

type Compare = Schemas['LanguageCompare']
type Row = Schemas['CompareRow']
type Translate = ReturnType<typeof useT>

const comparePath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/compare' as const

type Piece = { kind: 'same' | 'add' | 'del'; text: string }

/** A word diff of a source block's text then and now ("…ahead of time, in any timezone."). */
export function diffWords(before: string, after: string): Piece[] {
  const a = before.split(/(\s+)/).filter((w) => w !== '')
  const b = after.split(/(\s+)/).filter((w) => w !== '')
  // Longest common subsequence of the words; long texts fall back to all-new.
  if (a.length * b.length > 250_000) return [{ kind: 'del', text: before }, { kind: 'add', text: after }]
  const dp: number[][] = Array.from({ length: a.length + 1 }, () => new Array<number>(b.length + 1).fill(0))
  for (let i = a.length - 1; i >= 0; i--) {
    for (let j = b.length - 1; j >= 0; j--) {
      dp[i]![j] = a[i] === b[j] ? dp[i + 1]![j + 1]! + 1 : Math.max(dp[i + 1]![j]!, dp[i]![j + 1]!)
    }
  }
  const out: Piece[] = []
  const push = (kind: Piece['kind'], text: string) => {
    const last = out[out.length - 1]
    if (last && last.kind === kind) last.text += text
    else out.push({ kind, text })
  }
  let i = 0
  let j = 0
  while (i < a.length && j < b.length) {
    if (a[i] === b[j]) {
      push('same', a[i]!)
      i++
      j++
    } else if (dp[i + 1]![j]! >= dp[i]![j + 1]!) {
      push('del', a[i]!)
      i++
    } else {
      push('add', b[j]!)
      j++
    }
  }
  while (i < a.length) push('del', a[i++]!)
  while (j < b.length) push('add', b[j++]!)
  return out
}

/**
 * A diff around its changes (the inspector's "English · v6 → v9"): an
 * unchanged run keeps `words` words next to each change, and "…" stands
 * for the rest.
 */
export function excerpt(parts: Piece[], words = 8): Piece[] {
  const isWord = (w: string) => !/^\s+$/.test(w)
  // The tokens up to the n-th word from the start, or from the end.
  const head = (tokens: string[], n: number) => {
    let seen = 0
    const i = tokens.findIndex((w) => isWord(w) && ++seen === n)
    return tokens.slice(0, i + 1).join('')
  }
  const tail = (tokens: string[], n: number) => {
    let seen = 0
    for (let i = tokens.length - 1; i >= 0; i--) if (isWord(tokens[i]!) && ++seen === n) return tokens.slice(i).join('')
    return tokens.join('')
  }
  return parts.map((p, i): Piece => {
    if (p.kind !== 'same') return p
    const tokens = p.text.split(/(\s+)/).filter((w) => w !== '')
    const count = tokens.filter(isWord).length
    const first = i === 0
    const last = i === parts.length - 1
    if (first && last) return p
    if (first) return count <= words ? p : { kind: 'same', text: `…${tail(tokens, words)}` }
    if (last) return count <= words ? p : { kind: 'same', text: `${head(tokens, words)}…` }
    return count <= words * 2 ? p : { kind: 'same', text: `${head(tokens, words)} … ${tail(tokens, words)}` }
  })
}

function Diff({ before, after }: { before: string; after: string }) {
  return (
    <>
      {diffWords(before, after).map((p, i) =>
        p.kind === 'same' ? (
          <span key={i}>{p.text}</span>
        ) : p.kind === 'add' ? (
          <span key={i} className="lng-da">
            {p.text}
          </span>
        ) : (
          <span key={i} className="lng-dl">
            {p.text}
          </span>
        ),
      )}
    </>
  )
}

function stateBadge(state: Row['state'], source: string, language: string, t: Translate): { tone: Tone; text: string } {
  switch (state) {
    case 'up_to_date':
      return { tone: 'ok', text: t(mz.stateUpToDate) }
    case 'changed':
      return { tone: 'warn', text: t(mz.stateChanged, { source }) }
    case 'still':
      return { tone: 'neutral', text: t(mz.stateStill, { source }) }
    case 'new':
      return { tone: 'info', text: t(mz.stateNew, { source }) }
    case 'removed':
      return { tone: 'warn', text: t(mz.stateRemoved, { source }) }
  }
  return { tone: 'neutral', text: t(mz.stateOnly, { language }) }
}

/** Whether a row needs someone to look at it. */
function attention(r: Row): boolean {
  return r.state !== 'up_to_date' && r.state !== 'only'
}

/**
 * Compare a language version with the source, block by block
 * (ComposerCompare): each block's state, what changed in the source since
 * the version was translated, and the actions each state offers. The
 * version's text is edited on the canvas ("Show on canvas").
 */
/** The version the compare view shows: the one asked for when it has a version, else the first outdated one. */
export function compareLanguage(data: Langs, language: string): string | undefined {
  const choices = data.versions.filter((v) => v.status !== 'source' && v.status !== 'missing')
  return choices.some((c) => c.language === language) ? language : (choices.find((c) => c.status === 'outdated') ?? choices[0])?.language
}

/** The composer's status bar while comparing (ComposerCompare). */
export function CompareFooter({ campaignId, data, language, size, children }: { campaignId: string; data: Langs; language: string; size?: string; children?: ReactNode }) {
  const { slug } = useWorkspace()
  const t = useT()
  const lang = compareLanguage(data, language)
  const q = $api.useQuery('get', comparePath, { params: { path: { workspace: slug, campaignId, language: lang ?? '' } } }, { enabled: Boolean(lang) })
  const source = nameIn(data, data.source)
  const name = nameIn(data, lang ?? '')
  return (
    <footer className="cmp-foot">
      <div className="bc">
        <span>{t(mz.compareCrumb)}</span>
        {lang ? (
          <>
            <span aria-hidden="true">›</span>
            <span>{t(mz.compareCrumbLanguages, { source, language: name })}</span>
          </>
        ) : null}
      </div>
      <span className="k" style={{ marginLeft: 'auto' }}>
        {q.data
          ? t(mz.compareFoot, {
              language: name,
              size: size ?? 'none',
              changed: q.data.rows.filter((r) => r.state === 'changed').length,
              blocks: q.data.blocks,
              source,
              version: q.data.translatedFrom,
            })
          : null}
      </span>
      {children}
    </footer>
  )
}

export function CompareView({
  campaignId,
  data,
  language,
  onLanguage,
  onBack,
  onShow,
}: {
  campaignId: string
  data: Langs
  language: string
  onLanguage: (language: string) => void
  onBack: () => void
  /** Opens the version on the canvas at a block (its index in the version). */
  onShow: (index: number) => void
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const qc = useQueryClient()
  const cache = useLanguagesCache(campaignId)
  const [only, setOnly] = useState(false)
  const choices = data.versions.filter((v) => v.status !== 'source' && v.status !== 'missing')
  const lang = compareLanguage(data, language)
  const q = $api.useQuery('get', comparePath, { params: { path: { workspace: slug, campaignId, language: lang ?? '' } } }, { enabled: Boolean(lang) })
  const path = { workspace: slug, campaignId, language: lang ?? '' }
  const upToDate = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/mark-up-to-date')
  const copy = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/copy-block')
  const remove = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/remove-block')
  const source = nameIn(data, data.source)
  const name = nameIn(data, lang ?? '')
  const after = () => {
    void qc.invalidateQueries({ queryKey: ['get', comparePath] })
    void cache.refresh()
  }
  const failed = (e: unknown) => toast({ kind: 'error', title: t(mz.markFailed, { language: name }), detail: errorMessage(e) })
  const mark = (blocks?: string[]) =>
    upToDate.mutate({ params: { path }, body: { blocks } }, { onSuccess: (out) => (cache.set(out), after(), blocks ? undefined : toast({ title: t(mz.marked, { language: name }) })), onError: failed })
  const busy = upToDate.isPending || copy.isPending || remove.isPending
  if (!lang) {
    return (
      <div className="lng-cmp-wrap">
        <span className="k">{t(mz.nothingToShow)}</span>
      </div>
    )
  }
  const c = q.data
  const rows = c ? (only ? c.rows.filter(attention) : c.rows) : []
  return (
    <div className="lng-cmp-wrap" id="main">
      <div className="card lng-cmp">
        <section aria-label={t(mz.compareTitle, { source }).replace(/<\/?span>/g, '') + ' ' + name} className="flex min-w-0 flex-1 flex-col bg-white">
          <div className="lng-cmp-head">
            <h2 className="m-0 flex flex-wrap items-center gap-2" style={{ fontSize: 16, fontWeight: 700 }}>
              <T msg={mz.compareTitle} values={{ source, span: (s) => <span className="k" style={{ fontSize: 13 }}>{s}</span> }} />
              <select className="in" aria-label={t(mz.compareLanguage)} style={{ height: 32, fontWeight: 600, width: 'auto' }} value={lang} onChange={(e) => onLanguage(e.target.value)}>
                {choices.map((v) => (
                  <option key={v.language} value={v.language}>
                    {v.name}
                  </option>
                ))}
              </select>
            </h2>
            {c ? (
              <div className="flex items-center gap-2.5" style={{ minWidth: 220, flex: '0 1 300px' }}>
                <div className="bar" style={{ flex: 1 }}>
                  <i style={{ width: `${c.blocks ? Math.round((c.upToDate / c.blocks) * 100) : 0}%`, background: 'var(--accent)' }} />
                </div>
                <span className="k" style={{ whiteSpace: 'nowrap', color: 'var(--ink-nav)', fontWeight: 600 }}>
                  {t(mz.progress, { upToDate: c.upToDate, blocks: c.blocks })}
                </span>
              </div>
            ) : null}
            <label className="flex items-center gap-2" style={{ fontSize: 13, fontWeight: 600, marginLeft: 'auto' }}>
              <input type="checkbox" className="tg" checked={only} onChange={(e) => setOnly(e.target.checked)} />
              {t(mz.onlyAttention)}
            </label>
            <Button size="sm" onClick={onBack}>
              {t(mz.backToCanvas)}
            </Button>
          </div>
          {q.isError ? (
            <div className="pad">
              <ErrorPanel title={t(mz.compareFailed)}>{errorMessage(q.error)}</ErrorPanel>
            </div>
          ) : !c ? (
            <div className="pad flex flex-col gap-3" aria-busy="true">
              <Skeleton h={14} />
              <Skeleton h={60} r={8} />
              <Skeleton h={60} r={8} />
            </div>
          ) : (
            <>
              <div className="lng-cmph">
                <div>
                  {t(mz.sourceHead, { source, version: c.sourceVersion })}{' '}
                  <Badge tone="neutral" style={{ height: 20, fontSize: 11 }}>
                    {t(mz.readOnly)}
                  </Badge>
                </div>
                <div>
                  {c.status === 'draft' || !c.translatedBy || !c.translatedAt
                    ? t(mz.versionHeadDraft, { language: name })
                    : t(mz.versionHead, { language: name, version: c.translatedFrom, name: c.translatedBy, date: dayMonthDate(c.translatedAt) })}
                </div>
              </div>
              {!only || c.subject.state === 'changed' ? (
                <TextRow label={t(mz.subjectRow)} text={c.subject} source={source} language={name} t={t} />
              ) : null}
              {!only || c.preheader.state === 'changed' ? (
                <TextRow label={t(mz.preheaderRow)} text={c.preheader} source={source} language={name} t={t} />
              ) : null}
              {rows.map((r) => (
                <BlockRow
                  key={`${r.key}-${r.version?.index ?? 'x'}-${r.number}`}
                  row={r}
                  compare={c}
                  source={source}
                  language={name}
                  editable={data.canEdit}
                  busy={busy}
                  t={t}
                  onMark={() => mark([r.key])}
                  onShow={() => onShow(r.version?.index ?? 0)}
                  onCopy={() => copy.mutate({ params: { path }, body: { key: r.key } }, { onSuccess: after, onError: failed })}
                  onRemove={() => remove.mutate({ params: { path }, body: { index: r.version?.index } }, { onSuccess: after, onError: failed })}
                />
              ))}
              {only && !rows.length ? <div className="pad k">{t(mz.nothingToShow)}</div> : null}
              <div className="lng-cmp-foot">
                <span className="mr-auto flex flex-wrap items-center gap-2.5">
                  <Button icon="sparkle" disabledReason={t(mz.aiNotYet)}>
                    {t(mz.translateAi)}
                  </Button>
                  <span className="k">{t(mz.notYet)}</span>
                </span>
                {data.canEdit && c.status === 'outdated' ? (
                  <Button loading={upToDate.isPending} onClick={() => mark()}>
                    {t(mz.markLanguageUpToDate, { language: name })}
                  </Button>
                ) : null}
                <Button variant="primary" onClick={onBack}>
                  {t(mz.done)}
                </Button>
              </div>
            </>
          )}
        </section>
      </div>
    </div>
  )
}

function TextRow({ label, text, source, language, t }: { label: string; text: Schemas['CompareText']; source: string; language: string; t: Translate }) {
  const b = stateBadge(text.state === 'changed' ? 'changed' : 'up_to_date', source, language, t)
  return (
    <div className={`lng-cmpr${text.state === 'changed' ? ' att' : ''}`}>
      <div className="rh">
        <span className="bn">{label}</span>
        <Badge tone={b.tone} style={{ height: 20, fontSize: 11 }}>
          {b.text}
        </Badge>
        <span className="ra" />
      </div>
      <div className="lng-cmpc">
        <div className="src">
          <span className="tl">{t(mz.sourceLabel, { source })}</span>
          {text.source}
        </div>
        <div>
          <span className="tl">{language}</span>
          <div className="lng-text">{text.version}</div>
        </div>
      </div>
    </div>
  )
}

function BlockRow({
  row: r,
  compare: c,
  source,
  language,
  editable,
  busy,
  t,
  onMark,
  onShow,
  onCopy,
  onRemove,
}: {
  row: Row
  compare: Compare
  source: string
  language: string
  editable: boolean
  busy: boolean
  t: Translate
  onMark: () => void
  onShow: () => void
  onCopy: () => void
  onRemove: () => void
}) {
  const b = stateBadge(r.state, source, language, t)
  const name = r.source?.name ?? r.version?.name ?? ''
  const actions: ReactNode[] = []
  if (r.state === 'changed' && editable) {
    actions.push(
      <Button key="mark" size="sm" loading={busy} onClick={onMark}>
        {t(mz.markUpToDate)}
      </Button>,
    )
  }
  if ((r.state === 'changed' || r.state === 'still') && r.version) {
    actions.push(
      <Button key="show" size="sm" icon="eye" onClick={onShow}>
        {t(mz.showOnCanvas)}
      </Button>,
    )
  }
  if (r.state === 'new' && editable) {
    actions.push(
      <Button key="copy" size="sm" loading={busy} onClick={onCopy}>
        {t(mz.copyInto, { language })}
      </Button>,
    )
  }
  if (r.state === 'removed' && editable) {
    actions.push(
      <Button key="remove" size="sm" loading={busy} onClick={onRemove}>
        {t(mz.removeFrom, { language })}
      </Button>,
    )
  }
  const image = (r.source?.segments ?? []).every((s) => s.kind === 'image') && (r.source?.segments.length ?? 0) > 0
  return (
    <div className={`lng-cmpr${attention(r) ? (r.state === 'new' ? ' new' : r.state === 'removed' ? ' rem' : ' att') : ''}`}>
      <div className="rh">
        <span className="k mono" style={{ minWidth: 18 }}>
          {r.number || ''}
        </span>
        <span className="bn">{name}</span>
        <Badge tone={b.tone} style={{ height: 20, fontSize: 11 }}>
          {b.text}
        </Badge>
        <span className="ra">{actions}</span>
      </div>
      <div className="lng-cmpc">
        <div className="src">
          <span className="tl">{t(mz.sourceLabel, { source })}</span>
          {r.source ? (
            r.state === 'changed' && r.before ? (
              <Diff before={r.before} after={r.source.text} />
            ) : (
              r.source.text || <span className="k">{t(mz.noText)}</span>
            )
          ) : (
            '—'
          )}
          {r.state === 'changed' && r.changed ? (
            <div className="k" style={{ marginTop: 6 }}>
              {t(mz.changeLine, { from: r.fromVersion, to: c.sourceVersion, name: r.changed.by, date: `${dayMonthDate(r.changed.at)} ${clock(r.changed.at)}` })}
            </div>
          ) : null}
        </div>
        <div>
          <span className="tl">{language}</span>
          {r.version ? (
            image && r.state === 'up_to_date' ? (
              <div className="k" style={{ paddingTop: 9, color: 'var(--ink-nav)' }}>
                {t(mz.sameImage)}
              </div>
            ) : !r.version.text.trim() ? (
              <div className="k" style={{ paddingTop: 9 }}>
                {t(mz.noText)}
              </div>
            ) : (
              <div className={`lng-text${r.state === 'changed' ? ' on' : ''}`} lang={c.language}>
                {r.version.text}
              </div>
            )
          ) : (
            <div className="k" style={{ paddingTop: 9 }}>
              —
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
