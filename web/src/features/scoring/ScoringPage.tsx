import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { problemReference } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { AudienceTabs } from '@/features/audience/AudienceTabs'
import { formatList, useT } from '@/i18n'
import { clock } from '@/lib/format'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { followRecalc, savedWhat, scoringPath, scoringQuery, type Recalc } from './recalc'
import { RulePanel, type PanelMode } from './RulePanel'
import { RulesCard, SuggestedRules } from './RulesCard'
import { onlyPoints, SaveDialog } from './SaveDialog'
import { BandTiles, DecayCard, SpreadCard, TiersCard, UsedByCard } from './ScoringCards'
import { changesOf, configProblems, newRuleID, sameConfig, useImpact, type Config, type ScoringPageData } from './useScoringDraft'
import { ruleFor, signed, type Rule } from './ruleText'
import '@/features/audience/audience.css'
import { TiersNote } from './TiersNote'
import './scoring.css'

interface PanelState {
  mode: PanelMode
  index: number
  /** The draft before the panel opened: Cancel puts it back. */
  before: Config | null
}

/**
 * Audience › Scoring (ScoringRules, ScoringRulesTablet, ScoringStates): the
 * bands, how contacts spread, bands by tier, the rules, decay and what uses
 * the score. Edits stay a draft, with the moves between bands previewed,
 * until Save; a rule change recalculates every score in the background.
 */
export function ScoringPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const q = $api.useQuery('get', scoringPath, { params: { path: { workspace: slug } } })
  const page = q.data
  // No access: the shell shows its 403 screen.
  if (!page && q.error instanceof ApiError && q.error.status === 403) throw q.error
  const canEdit = useCan('scoring:edit')
  const [draft, setDraft] = useState<Config | null>(null)
  const [panel, setPanel] = useState<PanelState | null>(null)
  const [saving, setSaving] = useState(false)
  const [conflict, setConflict] = useState(false)

  const rc = page?.recalculation
  useEffect(() => {
    if (rc?.status === 'running') followRecalc(qc, slug, rc)
  }, [qc, slug, rc])

  const reason = page && !canEdit ? deniedReason('scoring:edit', page.helpers, t(m.editingScoring)) : undefined
  const config = draft ?? page?.config
  const openNew = () => {
    if (!page || !config) return
    const r = ruleFor('api_event', newRuleID(), page.options)
    setPanel({ mode: 'new', index: config.rules.length, before: draft })
    setDraft({ ...config, rules: [...config.rules, r] })
  }

  return (
    <Page
      title={t(m.audience)}
      documentTitle={t(m.documentTitle)}
      filters={<AudienceTabs />}
      actions={
        page && (page.configured || draft) ? (
          <Button variant="primary" icon="plus" disabledReason={reason} onClick={openNew}>
            {t(m.addRule)}
          </Button>
        ) : null
      }
    >
      {!page ? (
        q.isError ? (
          <LoadError error={q.error} onRetry={() => void q.refetch()} />
        ) : (
          <ScoringSkeleton />
        )
      ) : (
        <ScoringBody
          page={page}
          draft={draft}
          setDraft={setDraft}
          panel={panel}
          setPanel={setPanel}
          canEdit={canEdit}
          helpers={page.helpers}
          saving={saving}
          setSaving={setSaving}
          conflict={conflict}
          setConflict={setConflict}
          onAdd={openNew}
          onReload={() => {
            setConflict(false)
            setDraft(null)
            void q.refetch()
          }}
        />
      )}
    </Page>
  )
}

function ScoringBody({
  page,
  draft,
  setDraft,
  panel,
  setPanel,
  canEdit,
  helpers,
  saving,
  setSaving,
  conflict,
  setConflict,
  onAdd,
  onReload,
}: {
  page: ScoringPageData
  draft: Config | null
  setDraft: (c: Config | null) => void
  panel: PanelState | null
  setPanel: (p: PanelState | null) => void
  canEdit: boolean
  helpers: string[]
  saving: boolean
  setSaving: (b: boolean) => void
  conflict: boolean
  setConflict: (b: boolean) => void
  onAdd: () => void
  onReload: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const config = draft ?? page.config
  const dirty = Boolean(draft) && !sameConfig(draft!, page.config)
  const changes = useMemo(() => changesOf(page.config, config, page.configured), [page.config, config, page.configured])
  // A field rule's panel shows how many match today, even before an edit.
  const fieldPanel = panel !== null && config.rules[panel.index]?.kind === 'field'
  const { impact, loading } = useImpact(dirty || fieldPanel ? config : null, canEdit && (dirty || fieldPanel))
  const [saveOpen, setSaveOpen] = useState(false)
  const [saveError, setSaveError] = useState<ReactNode>()
  const put = $api.useMutation('put', scoringPath)

  const setRules = (rules: Rule[]) => setDraft({ ...config, rules })
  const replace = (i: number, r: Rule) => setRules(config.rules.map((x, j) => (j === i ? r : x)))
  const shown = dirty ? impact : undefined

  const save = async (startAutomations: boolean) => {
    setSaving(true)
    setSaveError(undefined)
    try {
      const next = await put.mutateAsync({ params: { path: { workspace: slug } }, body: { baseVersion: page.version, config, startAutomations } })
      qc.setQueryData(scoringQuery(slug).queryKey, next)
      setDraft(null)
      setSaveOpen(false)
      if (next.recalculation?.status === 'running' && next.recalculation.id !== page.recalculation?.id) {
        followRecalc(qc, slug, next.recalculation)
        toast({ title: t(m.savedRules) })
      } else {
        toast({ title: t(m.savedBands), detail: t(m.savedBandsDetail) })
      }
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        setSaveOpen(false)
        setConflict(true)
      } else {
        setSaveError(errorMessage(e))
      }
    } finally {
      setSaving(false)
    }
  }

  const startSave = () => {
    if (changes.onlyBands) void save(false)
    else {
      setSaveError(undefined)
      setSaveOpen(true)
    }
  }

  const openRule = (i: number) => setPanel({ mode: canEdit ? 'edit' : 'view', index: i, before: draft })
  const panelRule = panel ? config.rules[panel.index] : undefined
  const savedRule = panelRule ? page.config.rules.find((r) => r.id === panelRule.id) : undefined

  if (!page.configured && !draft) {
    return (
      <>
        <TiersNote on="scoring" />
        {canEdit ? null : <ReadOnlyNote helpers={helpers} />}
        <EmptyCard
          page={page}
          canEdit={canEdit}
          onSuggested={() => setDraft({ ...page.config, rules: page.suggested })}
          onAdd={onAdd}
          reason={canEdit ? undefined : deniedReason('scoring:edit', helpers, t(m.editingScoring))}
        />
      </>
    )
  }

  return (
    <>
      <TiersNote on="scoring" />
      {canEdit ? null : <ReadOnlyNote helpers={helpers} />}
      {page.recalculation ? <RecalcStatus rc={page.recalculation} canEdit={canEdit} /> : null}
      {conflict ? (
        <div className="note b" role="alert">
          <Icon name="alertCircle" size={16} />
          <div className="flex min-w-0 flex-[1_1_320px] flex-wrap items-center gap-x-3 gap-y-2">
            <span className="min-w-0 flex-[1_1_320px]">{t(m.conflict)}</span>
            <Button size="sm" onClick={onReload}>
              {t(m.reload)}
            </Button>
          </div>
        </div>
      ) : null}
      <BandTiles bands={page.config.bands} afterBands={config.bands} counts={page.bands} after={shown?.bandsAfter} total={page.contacts} />
      <div className="grid items-start gap-4 [grid-template-columns:repeat(auto-fit,minmax(min(460px,100%),1fr))]">
        <SpreadCard page={page} config={config} impact={shown} canEdit={canEdit} onBands={(bands) => setDraft({ ...config, bands })} />
        {page.byTier.length ? <TiersCard page={page} /> : null}
      </div>
      <RulesCard
        rules={config.rules}
        saved={page.config.rules}
        options={page.options}
        canEdit={canEdit}
        setUpBy={page.setUpBy}
        setUpAt={page.setUpAt}
        onToggle={(i, on) => replace(i, { ...config.rules[i]!, enabled: on })}
        onEdit={openRule}
        onDuplicate={(i) => {
          const copy = { ...config.rules[i]!, id: newRuleID() }
          setRules([...config.rules.slice(0, i + 1), copy, ...config.rules.slice(i + 1)])
        }}
        onDelete={(i) => setRules(config.rules.filter((_, j) => j !== i))}
      />
      <div className="grid items-start gap-4 [grid-template-columns:repeat(auto-fit,minmax(min(460px,100%),1fr))]">
        <DecayCard config={config} canEdit={canEdit} onDecay={(decay) => setDraft({ ...config, decay })} />
        <UsedByCard page={page} />
      </div>
      {dirty && !panel ? (
        <SaveBar
          page={page}
          config={config}
          changes={changes}
          changed={impact?.changed}
          previewing={loading}
          saving={saving}
          blocked={Object.values(configProblems(config)).find(Boolean)}
          onDiscard={() => setDraft(null)}
          onSave={startSave}
        />
      ) : null}
      {panel && panelRule ? (
        <RulePanel
          key={panelRule.id}
          mode={panel.mode}
          n={panel.index + 1}
          rule={panelRule}
          saved={savedRule}
          options={page.options}
          impact={impact}
          previewing={loading}
          onChange={(r) => replace(panel.index, r)}
          onApply={() => setPanel(null)}
          onCancel={() => {
            if (panel.mode !== 'view') setDraft(panel.before)
            setPanel(null)
          }}
        />
      ) : null}
      {saveOpen ? (
        <SaveDialog
          changes={changes}
          rules={config.rules}
          saved={page.config.rules}
          options={page.options}
          impact={impact}
          saving={saving}
          error={saveError}
          onCancel={() => setSaveOpen(false)}
          onSave={(start) => void save(start)}
        />
      ) : null}
      <UnsavedChangesGuard dirty={dirty && !saving} description={t(m.leaveDescription)} />
    </>
  )
}

/** Read-only without scoring:edit, with who can change it. */
function ReadOnlyNote({ helpers }: { helpers: string[] }) {
  const t = useT()
  const names = helpers
  return (
    <div className="note n" role="note">
      <span className="sc-note-ic" style={{ color: 'var(--ink-2)' }}>
        <Icon name="info" size={16} strokeWidth={1.8} />
      </span>
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(m.readOnly)}</span>
        <div>
          {t(names.length ? m.readOnlyAsk : m.readOnlyNoAsk, { helpers: orNames(names) })}
        </div>
      </div>
    </div>
  )
}

function orNames(names: string[]): string {
  return formatList(names, 'disjunction')
}

/** Running, or stopped with Try again (finished shows as a toast). */
function RecalcStatus({ rc, canEdit }: { rc: Recalc; canEdit: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const retry = $api.useMutation('post', '/api/v1/workspaces/{workspace}/scoring/recalculate')
  if (rc.status === 'running') {
    const left = rc.remainingSeconds !== undefined && rc.remainingSeconds !== null && rc.remainingSeconds >= 60 ? t(m.recalcLeft, { minutes: Math.round(rc.remainingSeconds / 60) }) : t(m.recalcAlmost)
    const pct = rc.total ? Math.min(100, Math.round((rc.done / rc.total) * 100)) : 0
    return (
      <div className="card flex flex-col gap-2.5 px-4.5 py-4" role="status">
        <div className="flex items-start gap-3">
          <span className="spin mt-[3px]" aria-hidden="true" />
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="font-bold">{t(m.recalcTitle, { total: rc.total, done: rc.done, left })}</span>
            <span className="k" style={{ color: 'var(--ink-nav)' }}>
              {t(m.recalcLead)}
            </span>
          </div>
        </div>
        <div className="sc-progress" role="progressbar" aria-label={t(m.recalcProgress)} aria-valuemin={0} aria-valuemax={100} aria-valuenow={pct} style={{ height: 8 }}>
          <i style={{ width: `${pct}%` }} />
        </div>
        <div className="flex flex-wrap justify-between gap-2">
          <span className="k">{t(m.recalcLine, { done: rc.done, total: rc.total, time: clock(rc.startedAt), name: rc.startedBy, what: savedWhat(rc.changes) })}</span>
          {(rc.summary?.automations ?? []).length === 0
            ? null
            : rc.summary!.automations.map((a) => (
                <span key={a.id} className="k">
                  {t(m.recalcAutomations, { name: a.name, started: String(rc.startAutomations), count: a.crossed })}
                </span>
              ))}
        </div>
      </div>
    )
  }
  if (rc.status !== 'failed') return null
  const rule = rc.changes.rules.length === 1 && !rc.changes.decay && !rc.changes.bands && !rc.changes.setup ? rc.changes.rules[0] : undefined
  return (
    <div className="card flex flex-col gap-3 px-4.5 py-4" style={{ borderColor: 'var(--bad-line)' }}>
      <div className="note b" role="alert">
        <span className="sc-note-ic" style={{ color: 'var(--bad-strong)' }}>
          <Icon name="alertCircle" size={16} strokeWidth={1.8} />
        </span>
        <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
          <span className="font-bold">{t(m.failedTitle, { done: rc.done, total: rc.total, time: clock(rc.failedAt ?? rc.startedAt) })}</span>
          <div>
            {rule ? t(m.failedBodyRule, { n: rule }) : t(m.failedBody)}
            <div className="mt-2.5 flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="primary"
                loading={retry.isPending}
                disabledReason={canEdit ? undefined : deniedReason('scoring:edit', [], t(m.editingScoring))}
                onClick={() =>
                  retry.mutate(
                    { params: { path: { workspace: slug } } },
                    {
                      onSuccess: (next) => {
                        qc.setQueryData(scoringQuery(slug).queryKey, next)
                        if (next.recalculation) followRecalc(qc, slug, next.recalculation)
                        toast({ title: t(m.retried) })
                      },
                      onError: (e) => toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) }),
                    },
                  )
                }
              >
                {t(m.tryAgain)}
              </Button>
              <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }}>
                {t(m.details)}
              </ButtonLink>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

/** The bar for unsaved changes: what changed and who moves. */
function SaveBar({
  page,
  config,
  changes,
  changed,
  previewing,
  saving,
  blocked,
  onDiscard,
  onSave,
}: {
  page: ScoringPageData
  config: Config
  changes: ReturnType<typeof changesOf>
  changed?: number
  previewing: boolean
  saving: boolean
  /** Why it can't be saved yet. */
  blocked?: string
  onDiscard: () => void
  onSave: () => void
}) {
  const t = useT()
  const n = changes.rules[0]
  const rule = n ? config.rules[n - 1] : undefined
  const was = rule ? page.config.rules.find((r) => r.id === rule.id) : undefined
  if (changes.setup) {
    return (
      <SaveBarFrame saving={saving} blocked={blocked} onDiscard={onDiscard} onSave={onSave}>
        {t(m.unsavedSetup, { count: config.rules.length, contacts: page.contacts })}
      </SaveBarFrame>
    )
  }
  const what =
    onlyPoints(changes, rule, was) && rule && was
      ? t(m.unsavedRule, { n: n!, from: signed(was.points), to: signed(rule.points) })
      : t(m.unsavedCount, { count: changes.rules.length + changes.deleted + (changes.decay ? 1 : 0) + (changes.bands ? 1 : 0) })
  const moves = previewing || changed === undefined ? t(m.previewing) : changed ? t(m.unsavedMoves, { count: changed }) : t(m.unsavedNoMoves)
  return (
    <SaveBarFrame saving={saving} blocked={blocked} onDiscard={onDiscard} onSave={onSave}>
      {what} · {moves}
    </SaveBarFrame>
  )
}

function SaveBarFrame({ saving, blocked, onDiscard, onSave, children }: { saving: boolean; blocked?: string; onDiscard: () => void; onSave: () => void; children: ReactNode }) {
  const t = useT()
  return (
    <div className="card sc-savebar" role="region" aria-label={t(m.unsaved)}>
      <Badge tone="warn" dot>
        {t(m.unsaved)}
      </Badge>
      <span className="k" style={{ color: 'var(--ink-nav)' }} aria-live="polite">
        {children}
      </span>
      <span className="ml-auto flex gap-2">
        <Button onClick={onDiscard} disabled={saving}>
          {t(m.discard)}
        </Button>
        <Button variant="primary" loading={saving} disabledReason={blocked} onClick={onSave}>
          {t(m.save)}
        </Button>
      </span>
    </div>
  )
}

/** No rules yet: the suggested rules, or one by hand. */
function EmptyCard({ page, canEdit, onSuggested, onAdd, reason }: { page: ScoringPageData; canEdit: boolean; onSuggested: () => void; onAdd: () => void; reason?: string }) {
  const t = useT()
  return (
    <section className="card sc-empty" aria-labelledby="sc-empty-t">
      <svg width="72" height="52" viewBox="0 0 72 52" aria-hidden="true">
        <rect x="4" y="30" width="10" height="18" rx="2" fill="var(--chart-score-1)" />
        <rect x="18" y="22" width="10" height="26" rx="2" fill="var(--chart-score-2)" />
        <rect x="32" y="14" width="10" height="34" rx="2" fill="var(--chart-score-3)" />
        <path d="M50 26h16M58 18v16" stroke="var(--accent)" strokeWidth="2.4" strokeLinecap="round" />
      </svg>
      <h2 id="sc-empty-t" className="tt">
        {t(m.emptyTitle)}
      </h2>
      <p className="ds">{t(m.emptyBody)}</p>
      <SuggestedRules rules={page.suggested} options={page.options} bands={page.config.bands} weeks={page.config.decay.afterWeeks} />
      <div className="flex flex-wrap justify-center gap-2.5">
        <Button variant="primary" style={{ minHeight: 40 }} disabledReason={reason} onClick={onSuggested}>
          {t(m.useSuggested)}
        </Button>
        <Button style={{ minHeight: 40 }} disabledReason={reason} onClick={onAdd}>
          {t(m.addOne)}
        </Button>
      </div>
      {canEdit ? <span className="k">{t(m.suggestedNote)}</span> : null}
    </section>
  )
}

/** Couldn't load: what failed, that scores keep updating, Retry. */
function LoadError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const ref = problemReference(error)
  return (
    <section className="card sc-loaderr" role="alert">
      <span style={{ color: 'var(--bad-strong)' }}>
        <Icon name="alertCircle" size={28} />
      </span>
      <h2 className="m-0 text-[16px] font-bold">{t(m.loadError)}</h2>
      <span className="k max-w-[440px]" style={{ color: 'var(--ink-2)' }}>
        {t(m.loadErrorBody, { detail: errorMessage(error) })}
      </span>
      <div className="flex flex-wrap justify-center gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(m.retry)}
        </Button>
        <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
          {t(m.overview)}
        </ButtonLink>
      </div>
      {ref ? <span className="k mono">{t(m.requestRef, { ref })}</span> : null}
    </section>
  )
}

function ScoringSkeleton() {
  const t = useT()
  return (
    <div className="card flex flex-col gap-3.5 p-4.5" aria-busy="true" aria-label={t(m.loading)}>
      <div className="grid grid-cols-3 gap-3 max-md:grid-cols-1">
        {[0, 1, 2].map((i) => (
          <div key={i} className="card sc-tile" style={{ borderLeft: '4px solid var(--line)' }}>
            <Skeleton w={90} h={12} r={6} />
            <Skeleton w={70} h={22} r={6} style={{ marginTop: 6 }} />
            <Skeleton w={110} h={10} r={6} style={{ marginTop: 6 }} />
          </div>
        ))}
      </div>
      <div className="flex h-[120px] items-end justify-between gap-3.5 border-b border-line-2 px-2.5">
        {[100, 70, 48, 34, 24, 18, 12, 9, 7, 5].map((h, i) => (
          <Skeleton key={i} w={22} h={h} r={4} />
        ))}
      </div>
      {[0, 1, 2].map((i) => (
        <div key={i} className="flex items-center gap-3">
          <Skeleton w={40} h={22} r={6} />
          <Skeleton w="42%" h={12} r={6} />
          <Skeleton w={80} h={12} r={6} />
          <Skeleton w={40} h={22} r={999} />
        </div>
      ))}
    </div>
  )
}
