import { Fragment, useState } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Bar } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Grid, Row } from '@/components/ui/Layout'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { calendarDate, count, pct, trimPct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './builder.messages'
import { useInvalidateForms, type SignupForm } from './common'
import { abSentence, confidences, splitText, type ABDraft, type ABResult, type VariantKey } from './variants'

/**
 * "Steps" under "Last 30 days" for a form with several steps
 * (FormsStates): how many views each step had and how many signed up,
 * as bars against step 1 or as a table.
 */
export function StepFunnel({ stats }: { stats: SignupForm['stats'] }) {
  const t = useT()
  const [table, setTable] = useState(false)
  const first = stats.steps[0]?.views ?? 0
  const share = (n: number) => (first ? (n / first) * 100 : 0)
  const rows = [
    ...stats.steps.map((s) => ({ key: `s${s.step}`, name: t(m.stepN, { n: s.step }), views: s.views, value: t(m.stepViews, { n: s.views, count: count(s.views) }) })),
    { key: 'signups', name: t(m.signups), views: stats.stepSignups, value: count(stats.stepSignups) },
  ]
  return (
    <>
      <span className="k">{t(m.steps)}</span>
      {table ? (
        <Table caption={t(m.funnel)}>
          <thead>
            <tr>
              <Th>{t(m.colStep)}</Th>
              <Th align="right">{t(m.views)}</Th>
              <Th align="right">{t(m.colOfFirst)}</Th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.key}>
                <Td>{r.name}</Td>
                <Td align="right" num>
                  {count(r.views)}
                </Td>
                <Td align="right" num>
                  {trimPct(Number(share(r.views).toFixed(1)))}
                </Td>
              </tr>
            ))}
          </tbody>
        </Table>
      ) : (
        <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-x-2.5 gap-y-2 text-[13px]">
          {rows.map((r) => (
            <Fragment key={r.key}>
              <span>{r.name}</span>
              <Bar value={share(r.views)} />
              <span className="n">{r.value}</span>
            </Fragment>
          ))}
        </div>
      )}
      <Button size="sm" className="self-start" onClick={() => setTable(!table)}>
        {table ? t(m.showBars) : t(m.showTable)}
      </Button>
    </>
  )
}

/**
 * "A/B test" (FormBuilderABTest, FormsStates "A/B test results"): each
 * variant's submission rate since the test started, where it stands, its
 * rule (saved with Save & publish) and ending it.
 */
export function ABTestCard({
  result,
  draft,
  formId,
  heading,
  reason,
  errors,
  onDraft,
}: {
  result: ABResult
  draft: ABDraft
  formId: string
  /** Variant A's heading: the form's saved one. */
  heading: string
  reason?: string
  errors: Record<string, string>
  onDraft: (patch: Partial<ABDraft>) => void
}) {
  const t = useT()
  const [table, setTable] = useState(false)
  const [ending, setEnding] = useState<VariantKey>()
  const readOnly = Boolean(reason)
  const top = Math.max(0, ...result.variants.map((v) => v.rate))
  const leader = result.outcome === 'winner' ? result.leader : undefined
  const err = (k: string) =>
    errors[k] ? (
      <span className="err" role="alert">
        {errors[k]}
      </span>
    ) : null
  return (
    <div className="fb-sec last">
      <div className="flex items-center gap-2.5">
        <span className="font-semibold">{t(m.abTitle)}</span>
        <Badge tone="ok" dot>
          {t(m.abRunning)}
        </Badge>
      </div>
      {result.startedOn ? <span className="k">{t(m.abSince, { date: calendarDate(result.startedOn, 'shortDate'), split: splitText(result.variants) })}</span> : null}
      {table ? (
        <div className="rounded-lg border border-line">
          <Table caption={t(m.abResults)}>
            <thead>
              <tr>
                <Th>{t(m.colVariant)}</Th>
                <Th align="right">{t(m.views)}</Th>
                <Th align="right">{t(m.submissions)}</Th>
                <Th align="right">{t(m.colRate)}</Th>
                <Th align="right">{t(m.colConfirmed)}</Th>
              </tr>
            </thead>
            <tbody>
              {result.variants.map((v) => (
                <tr key={v.key}>
                  <Td>{v.key}</Td>
                  <Td align="right" num>
                    {count(v.views)}
                  </Td>
                  <Td align="right" num>
                    {count(v.submissions)}
                  </Td>
                  <Td align="right" num>
                    {pct(v.rate)}
                  </Td>
                  <Td align="right" num>
                    {count(v.confirmed)}
                  </Td>
                </tr>
              ))}
            </tbody>
          </Table>
        </div>
      ) : (
        result.variants.map((v) => (
          <div key={v.key} className="flex flex-col gap-1">
            <div className="flex items-center gap-2.5">
              <Badge tone="neutral">{v.key}</Badge>
              <Bar className="flex-1" value={top ? (v.rate / top) * 100 : 0} />
              <span className="n text-[13px]">{pct(v.rate)}</span>
            </div>
            <span className="k">
              {result.doubleOptIn
                ? t(m.abOfViewsConfirmed, { submissions: count(v.submissions), views: count(v.views), confirmed: count(v.confirmed) })
                : t(m.abOfViews, { submissions: count(v.submissions), views: count(v.views) })}
            </span>
          </div>
        ))
      )}
      <Button size="sm" className="self-start" onClick={() => setTable(!table)}>
        {table ? t(m.showBars) : t(m.showTable)}
      </Button>
      <p className="m-0 text-[13px] font-semibold">{abSentence(result)}</p>
      <label className="fb-check">
        <input type="checkbox" checked={draft.autoEnd} disabled={readOnly} onChange={(e) => onDraft({ autoEnd: e.target.checked })} />
        {t(m.abAutoEnd)}
      </label>
      <Grid cols={2} gap={8} className="items-end">
        <label className="lbl">
          {t(m.abConfidence)}
          <select className="fb-in" value={draft.confidence} disabled={readOnly} aria-invalid={errors['ab.confidence'] ? true : undefined} onChange={(e) => onDraft({ confidence: Number(e.target.value) })}>
            {confidences.map((c) => (
              <option key={c} value={c}>
                {pct(c * 100, 0)}
              </option>
            ))}
          </select>
        </label>
        <label className="lbl">
          {t(m.abMinViews)}
          <input
            className="fb-in"
            type="number"
            min={100}
            max={100000}
            value={draft.minViews}
            disabled={readOnly}
            aria-invalid={errors['ab.minViews'] ? true : undefined}
            onChange={(e) => onDraft({ minViews: Number(e.target.value) })}
          />
        </label>
      </Grid>
      {err('ab.confidence')}
      {err('ab.minViews')}
      <Row gap={6} wrap>
        {leader ? (
          <Button size="sm" variant="primary" disabledReason={reason} onClick={() => setEnding(leader as VariantKey)}>
            {t(m.abKeepEnd, { key: leader })}
          </Button>
        ) : null}
        <Button size="sm" disabledReason={reason} onClick={() => setEnding((result.leader as VariantKey | undefined) ?? 'A')}>
          {t(m.abEnd)}
        </Button>
      </Row>
      {ending ? <EndTestDialog formId={formId} result={result} heading={heading} keep={ending} onClose={() => setEnding(undefined)} /> : null}
    </div>
  )
}

/**
 * "End the A/B test" (FormBuilderABTest): what ending it does, then the
 * variant to keep (the leader preselected).
 */
function EndTestDialog({ formId, result, heading, keep: first, onClose }: { formId: string; result: ABResult; heading: string; keep: VariantKey; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateForms()
  const end = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms/{form}/ab/end')
  const [keep, setKeep] = useState(first)
  const submit = async () => {
    try {
      await end.mutateAsync({ params: { path: { workspace: slug, form: formId } }, body: { keep } })
      toast({ kind: 'success', title: tt(m.ended, { key: keep }), detail: tt(m.endedBody) })
      onClose()
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: tt(m.endFailed), detail: errorMessage(e), retry: () => void submit() })
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.endTitle)}
      size="sm"
      dismissible={false}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={end.isPending} busyLabel={t(m.ending)} onClick={() => void submit()}>
            {t(m.endConfirm, { key: keep })}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13.5px] text-ink-2">
        <li>{t(m.endTakesOver)}</li>
        <li>{t(m.endDeletes)}</li>
        <li>{t(m.endNextLoad)}</li>
      </ul>
      <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
        <legend className="vh">{t(m.endKeepLegend)}</legend>
        {result.variants.map((v) => (
          <label key={v.key} className={cn('opt', keep === v.key && 'on')}>
            <input type="radio" name="ab-keep" checked={keep === v.key} onChange={() => setKeep(v.key)} />
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="font-semibold">{t(m.endKeep, { key: v.key, heading: v.key === 'A' ? heading : (v.content?.heading ?? '') })}</span>
            </span>
          </label>
        ))}
      </fieldset>
    </Dialog>
  )
}
