import { useQueryClient } from '@tanstack/react-query'
import { Fragment, useEffect, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, OptionCard, Select, Toggle } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { andList, clock, count, dayMonthDate, dayMonthYear, monthDayTime } from '@/lib/format'
import { trackJob, updateJob } from '@/lib/jobs'
import { useWorkspace } from '@/lib/session'
import { EARN, useEarnStatus } from './EarnPage'
import { day, pricePer } from './format'
import { mi } from './import.messages'
import { memberEmailName } from './MemberPanel'

type Import = Schemas['EarnImport']
type Source = Import['source']
const sources: Source[] = ['substack', 'ghost', 'beehiiv', 'kit', 'csv']

/** Import paid members: source, upload, match with Stripe, prices and lists, review and import (EarnImport). */
export function ImportDialog({ id, onClose, onChange }: { id?: string; onClose: () => void; onChange: (id: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = useEarnStatus()
  const q = $api.useQuery('get', `${EARN}/imports/{id}`, { params: { path: { workspace: slug, id: id ?? '' } } }, { enabled: Boolean(id), refetchInterval: (s) => (s.state.data?.status === 'importing' ? 1500 : false) })
  const [source, setSource] = useState<Source>('substack')
  const [picked, setStep] = useState<number>(id ? 0 : 1)
  const imp = q.data
  // An import opened again starts at the step its state is at.
  const step = picked === 0 && imp ? (imp.status === 'uploaded' ? 2 : imp.status === 'matched' ? 3 : 5) : picked
  const connected = Boolean(status.data?.mode)
  const platform = imp?.platform ?? platformLabel(t, source)
  const title =
    step <= 1 ? t(mi.title) : step === 2 ? t(mi.uploadTitle) : imp?.status === 'done' ? t(mi.doneTitle, { rows: imp.rows, platform: imp.platform }) : t(mi.titleFrom, { platform })
  // Matching, prices and the review lay out wide, as EarnImport draws them.
  const wide = step >= 3 && imp?.status !== 'done' && imp?.status !== 'importing'
  const stepName = [t(mi.stepSource), t(mi.stepUpload), t(mi.stepMatch), t(mi.stepPrices), t(mi.stepReview)][Math.max(0, Math.min(4, step - 1))]
  return (
    <Dialog
      open
      size={wide ? 'lg' : 'md'}
      hideClose
      onOpenChange={(o) => !o && onClose()}
      className={wide ? 'ea-import ea-wide' : 'ea-import'}
      eyebrow={step > 0 && imp?.status !== 'done' ? <span className="k">{t(mi.stepOf, { n: step, step: (stepName ?? '').charAt(0).toLowerCase() + (stepName ?? '').slice(1) })}</span> : undefined}
      title={title}
      description={wide ? t(mi.matchLead, { account: status.data?.account ?? '' }) : undefined}
      dismissible={imp?.status !== 'importing'}
    >
      {!connected && status.data ? (
        <NotConnected onClose={onClose} />
      ) : id && !imp ? (
        q.error ? <ErrorPanel title={t(mi.errorTitle)}>{errorMessage(q.error)}</ErrorPanel> : <Skeleton w="100%" h={300} />
      ) : step <= 1 ? (
        <SourceStep source={source} onSource={setSource} account={status.data?.account ?? ''} onCancel={onClose} onNext={() => setStep(2)} />
      ) : step === 2 ? (
        <UploadStep imp={imp} source={imp?.source ?? source} onBack={() => setStep(1)} onUploaded={(x) => onChange(x.id)} onNext={() => setStep(3)} />
      ) : imp && step === 3 ? (
        <MatchStep imp={imp} onBack={() => setStep(2)} onCancel={onClose} onNext={() => setStep(4)} />
      ) : imp && step === 4 ? (
        <PricesStep imp={imp} onBack={() => setStep(3)} onCancel={onClose} onNext={() => setStep(5)} />
      ) : imp ? (
        <ReviewStep imp={imp} onBack={() => setStep(4)} onClose={onClose} />
      ) : null}
    </Dialog>
  )
}

function platformLabel(t: ReturnType<typeof useT>, s: Source): string {
  return { substack: t(mi.substack), ghost: t(mi.ghost), beehiiv: t(mi.beehiiv), kit: t(mi.kit), csv: t(mi.csv) }[s]
}

function Steps({ at, platform }: { at: number; platform: string }) {
  const t = useT()
  const names = [at > 1 ? t(mi.stepSourceDone, { platform }) : t(mi.stepSource), t(mi.stepUpload), t(mi.stepMatch), t(mi.stepPrices), t(mi.stepReview)]
  return (
    <ol className="ea-steps m-0 list-none p-0" aria-label={t(mi.stepsLabel)}>
      {names.map((n, i) => (
        <Fragment key={i}>
          {i > 0 ? (
            <li className="sep" aria-hidden="true">
              ›
            </li>
          ) : null}
          <li className={`s ${i + 1 < at ? 'done' : i + 1 === at ? 'cur' : ''}`} aria-current={i + 1 === at ? 'step' : undefined}>
            <span className="n">{i + 1 < at ? <Icon name="check" size={12} /> : i + 1}</span>
            {n}
          </li>
        </Fragment>
      ))}
    </ol>
  )
}

function Footer({ children }: { children: ReactNode }) {
  return <div className="mt-4 flex flex-wrap items-center justify-between gap-2 border-t border-line-2 pt-4">{children}</div>
}

function NotConnected({ onClose }: { onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className="flex flex-col gap-3">
      <Note tone="warn" label={t(mi.notConnectedTitle)}>
        {t(mi.notConnectedText)}
      </Note>
      <span className="k">{t(mi.notConnectedTest)}</span>
      <Footer>
        <Button onClick={onClose}>{t(mi.close)}</Button>
        <ButtonLink to="/w/$workspace/earn/connect" params={{ workspace: slug }} variant="primary">
          {t(mi.connectStripe)}
        </ButtonLink>
      </Footer>
    </div>
  )
}

function SourceStep({ source, onSource, account, onCancel, onNext }: { source: Source; onSource: (s: Source) => void; account: string; onCancel: () => void; onNext: () => void }) {
  const t = useT()
  const hints: Record<Source, string> = { substack: t(mi.substackHint), ghost: t(mi.ghostHint), beehiiv: t(mi.beehiivHint), kit: t(mi.kitHint), csv: t(mi.csvHint) }
  return (
    <div className="flex flex-col gap-3">
      <Steps at={1} platform="" />
      <p className="m-0 text-ink-2">{t(mi.sourceLead, { account })}</p>
      <div className="ea-srcopts" role="group" aria-label={t(mi.stepSource)}>
        {sources.map((s) => (
          <button key={s} type="button" className="ea-srcopt" aria-pressed={source === s} onClick={() => onSource(s)}>
            <b>{platformLabel(t, s)}</b>
            <span className="k">{hints[s]}</span>
          </button>
        ))}
      </div>
      <Footer>
        <span />
        <div className="flex gap-2">
          <Button onClick={onCancel}>{t(mi.cancel)}</Button>
          <Button variant="primary" onClick={onNext}>
            {t(mi.continue)}
          </Button>
        </div>
      </Footer>
    </div>
  )
}

function useRefresh() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: ['get', `${EARN}/imports/{id}`] })
}

function UploadStep({ imp, source, onBack, onUploaded, onNext }: { imp?: Import; source: Source; onBack: () => void; onUploaded: (i: Import) => void; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const file = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string>()
  const [email, setEmail] = useState('')
  const [customer, setCustomer] = useState('')
  const columns = $api.useMutation('put', `${EARN}/imports/{id}/columns`)
  const match = $api.useMutation('post', `${EARN}/imports/{id}/match`)
  const platform = imp?.platform ?? platformLabel(t, source)
  const upload = async (f: File) => {
    setBusy(true)
    setErr(undefined)
    try {
      const { data } = await http.POST(`${EARN}/imports`, {
        params: { path: { workspace: slug }, query: { source, fileName: f.name } },
        body: (await f.arrayBuffer()) as unknown as string,
        headers: { 'Content-Type': 'application/octet-stream' },
        bodySerializer: (b: unknown) => b as BodyInit,
      })
      if (data) onUploaded(data)
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const used = (u: string) => ({ email: t(mi.useEmail), customer: t(mi.useCustomer), plan: t(mi.usePlan), paid_through: t(mi.usePaidThrough), name: t(mi.useName) })[u] ?? t(mi.useNone)
  return (
    <div className="flex flex-col gap-3">
      <Steps at={2} platform={platform} />
      <p className="m-0 text-ink-2">{t(mi.uploadLead, { platform })}</p>
      <input
        ref={file}
        type="file"
        accept=".csv,.zip,text/csv,application/zip"
        className="vh"
        aria-label={t(mi.chooseFile)}
        onChange={(e) => {
          const f = e.target.files?.[0]
          if (f) void upload(f)
          e.target.value = ''
        }}
      />
      {imp ? (
        <div className="card flex flex-wrap items-center justify-between gap-3 px-4 py-3">
          <div className="flex items-center gap-3">
            <Icon name="table" size={18} />
            <div className="flex flex-col">
              <span className="font-semibold">{imp.fileName}</span>
              <span className="k">{t(mi.fileLine, { rows: imp.rows, size: sizeOf(imp.fileSize), when: monthDayTime(imp.createdAt) })}</span>
            </div>
          </div>
          <Button size="sm" onClick={() => file.current?.click()} loading={busy}>
            {t(mi.replace)}
          </Button>
        </div>
      ) : (
        <div>
          <Button icon="upload" onClick={() => file.current?.click()} loading={busy} busyLabel={t(mi.uploading)}>
            {t(mi.chooseFile)}
          </Button>
        </div>
      )}
      {err ? <ErrorPanel title={err} /> : null}
      {imp ? (
        <>
          <TableScroll label={t(mi.columnsLabel)}>
            <table className="tbl" aria-label={t(mi.columnsLabel)}>
              <thead>
                <tr>
                  <th scope="col">{t(mi.colFound)}</th>
                  <th scope="col">{t(mi.colUsedAs)}</th>
                  <th scope="col">{t(mi.colExample)}</th>
                </tr>
              </thead>
              <tbody>
                {imp.columns.map((c) => (
                  <tr key={c.name}>
                    <td className="ea-mono">{c.name}</td>
                    <td>
                      {c.usedAs ? (
                        <span className="inline-flex items-center gap-1.5">
                          <Icon name="check" size={13} /> {used(c.usedAs)}
                        </span>
                      ) : (
                        <span className="k">{t(mi.useNone)}</span>
                      )}
                    </td>
                    <td className="ea-mono">{c.example}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
          {imp.problem === 'no_email' ? (
            <div className="flex flex-col gap-3">
              <Note tone="warn">{t(mi.noEmail)}</Note>
              <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))' }}>
                <Field label={t(mi.emailColumn)} hint={t(mi.required)}>
                  <Select value={email} onChange={(e) => setEmail(e.target.value)}>
                    <option value="">{t(mi.pickColumn)}</option>
                    {imp.columns.map((c) => (
                      <option key={c.name} value={c.name}>
                        {c.name}
                      </option>
                    ))}
                  </Select>
                </Field>
                <Field label={t(mi.customerColumn)} hint={t(mi.customerHint)}>
                  <Select value={customer} onChange={(e) => setCustomer(e.target.value)}>
                    <option value="">{t(mi.noneOption)}</option>
                    {imp.columns.map((c) => (
                      <option key={c.name} value={c.name}>
                        {c.name}
                      </option>
                    ))}
                  </Select>
                </Field>
              </div>
            </div>
          ) : (
            <Note tone="ok">
              {imp.withCustomer > 0 ? t(mi.noProblems, { with: imp.withCustomer, rows: imp.rows, rest: imp.rows - imp.withCustomer }) : t(mi.noProblemsAllEmail)}
            </Note>
          )}
        </>
      ) : null}
      <Footer>
        <Button onClick={onBack}>{t(mi.back)}</Button>
        <Button
          variant="primary"
          disabled={!imp || (imp.problem === 'no_email' && !email)}
          loading={match.isPending || columns.isPending}
          busyLabel={t(mi.matching)}
          onClick={async () => {
            if (!imp) return
            try {
              if (imp.problem === 'no_email') await columns.mutateAsync({ params: { path: { workspace: slug, id: imp.id } }, body: { email, customer } })
              await match.mutateAsync({ params: { path: { workspace: slug, id: imp.id } } })
              await refresh()
              onNext()
            } catch (e) {
              setErr(errorMessage(e))
            }
          }}
        >
          {t(mi.matchBtn)}
        </Button>
      </Footer>
    </div>
  )
}

function sizeOf(bytes: number): string {
  if (bytes >= 1 << 20) return `${(bytes / (1 << 20)).toFixed(1)} MB`
  return `${Math.max(1, Math.round(bytes / 1024))} KB`
}

function MatchStep({ imp, onBack, onCancel, onNext }: { imp: Import; onBack: () => void; onCancel: () => void; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const save = $api.useMutation('put', `${EARN}/imports/{id}/decisions`)
  const [dec, setDec] = useState(imp.decisions)
  const lists = imp.lists.map((l) => l.name).join(', ') || '—'
  const persist = async (next: Schemas['EarnImportDecisions']) => {
    setDec(next)
    await save.mutateAsync({ params: { path: { workspace: slug, id: imp.id } }, body: next })
    await refresh()
  }
  const total = (imp.active?.count ?? 0) + (imp.ended?.count ?? 0) + (imp.none?.count ?? 0) + imp.conflicts.length
  return (
    <div className="flex flex-col gap-3">
      <Steps at={3} platform={imp.platform} />
      <div className="flex flex-wrap items-center justify-between gap-2 text-[13px]">
        <span>
          <b>{imp.fileName}</b> <span className="k">· {t(mi.fileMatched, { rows: imp.rows, account: imp.account })}</span>
        </span>
        <Badge tone="ok" dot>
          {t(mi.matched, { done: total, rows: imp.rows })}
        </Badge>
      </div>
      <div className="ea-meter" aria-hidden="true">
        <span style={{ width: `${imp.rows ? Math.min(100, (total / imp.rows) * 100) : 0}%` }} />
      </div>
      <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))' }}>
        <Group title={t(mi.groupActive)} text={t(mi.groupActiveText, { byCustomer: imp.active?.byCustomer ?? 0, byEmail: imp.active?.byEmail ?? 0 })} n={imp.active?.count ?? 0} tone="ok">
          {imp.active?.examples.map((x) => (
            <Row key={x.row} left={x.email} right={x.tier ? `${x.tier} · ${pricePer(t, x.amountCents ?? 0, x.currency ?? 'usd', x.period)}` : pricePer(t, x.amountCents ?? 0, x.currency ?? 'usd', x.period)} />
          ))}
        </Group>
        <Group title={t(mi.groupEnded)} text={t(mi.groupEndedText, { lists })} n={imp.ended?.count ?? 0} tone="neutral">
          {imp.ended?.examples.map((x) => (
            <Row key={x.row} left={x.email} right={[x.tier, x.endedAt ? t(mi.endedOn, { date: dayMonthDate(x.endedAt) }) : ''].filter(Boolean).join(' · ')} />
          ))}
        </Group>
        <Group title={t(mi.groupNone)} text={t(mi.groupNoneText, { platform: imp.platform })} n={imp.none?.count ?? 0} tone="info">
          {imp.none?.examples.map((x) => (
            <Row key={x.row} left={x.email} right={x.paidThrough ? t(mi.paidThrough, { date: dayMonthYear(x.paidThrough) }) : ''} />
          ))}
          {imp.none?.count ? (
            <fieldset className="m-0 mt-2 flex flex-col gap-2 border-0 p-0">
              <legend className="mb-1 text-[13px] font-semibold">{t(mi.forThese, { count: imp.none.count })}</legend>
              <OptionCard
                name="none"
                value="complimentary"
                checked={dec.noSubscription === 'complimentary'}
                onChange={() => void persist({ ...dec, noSubscription: 'complimentary' })}
                title={t(mi.noneComp)}
                description={imp.paidFirst && imp.paidLast ? t(mi.noneCompHint, { first: dayMonthYear(imp.paidFirst), last: dayMonthYear(imp.paidLast) }) : t(mi.noneCompNoDates)}
              />
              <OptionCard name="none" value="free" checked={dec.noSubscription === 'free'} onChange={() => void persist({ ...dec, noSubscription: 'free' })} title={t(mi.noneFree)} />
              <OptionCard name="none" value="skip" checked={dec.noSubscription === 'skip'} onChange={() => void persist({ ...dec, noSubscription: 'skip' })} title={t(mi.noneSkip)} />
            </fieldset>
          ) : null}
        </Group>
        <Group title={t(mi.conflicts)} text={imp.conflicts.length ? '' : t(mi.conflictsNone)} n={imp.conflicts.length} tone="warn">
          {imp.conflicts.map((c) => (
            <div key={c.row} className="flex flex-col gap-2 border-t border-line-2 pt-2">
              <span className="text-[13px]">{t(mi.conflictText, { email: c.email, count: c.candidates.length, account: imp.account })}</span>
              {c.candidates.map((x) => (
                <OptionCard
                  key={x.customer}
                  name={`conflict-${c.row}`}
                  value={x.customer}
                  checked={(dec.conflicts ?? {})[String(c.row)] === x.customer}
                  onChange={() => void persist({ ...dec, conflicts: { ...(dec.conflicts ?? {}), [String(c.row)]: x.customer } })}
                  title={<span className="ea-mono">{x.customer}</span>}
                  description={x.status ? t(mi.conflictSub, { tier: x.tier || '—', status: x.status, date: day(x.since), price: pricePer(t, x.amountCents ?? 0, x.currency ?? 'usd', x.period) }) : t(mi.conflictNoSub, { date: dayMonthYear(x.created) })}
                />
              ))}
              <Button size="sm" onClick={() => void persist({ ...dec, conflicts: { ...(dec.conflicts ?? {}), [String(c.row)]: '' } })}>
                {t(mi.skipRow)}
              </Button>
            </div>
          ))}
        </Group>
      </div>
      <span className="k">{t(mi.nothingChanges)}</span>
      <Footer>
        <Button onClick={onBack}>{t(mi.back)}</Button>
        <div className="flex gap-2">
          <Button onClick={onCancel}>{t(mi.cancel)}</Button>
          <Button variant="primary" onClick={onNext}>
            {t(mi.toPrices)}
          </Button>
        </div>
      </Footer>
    </div>
  )
}

function Group({ title, text, n, tone, children }: { title: string; text: string; n: number; tone: 'ok' | 'neutral' | 'info' | 'warn'; children?: ReactNode }) {
  return (
    <section className="card flex flex-col gap-2 p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h3 className="m-0 text-[14px] font-semibold">{title}</h3>
          {text ? <span className="k">{text}</span> : null}
        </div>
        <Badge tone={tone} className="ea-num">
          {count(n)}
        </Badge>
      </div>
      {children}
    </section>
  )
}

function Row({ left, right }: { left: string; right: string }) {
  return (
    <div className="flex items-baseline justify-between gap-3 border-t border-line-2 pt-2 text-[13px]">
      <span className="ea-mono">{left}</span>
      <span className="k text-right">{right}</span>
    </div>
  )
}

function PricesStep({ imp, onBack, onCancel, onNext }: { imp: Import; onBack: () => void; onCancel: () => void; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const save = $api.useMutation('put', `${EARN}/imports/{id}/decisions`)
  const [dec, setDec] = useState(imp.decisions)
  const persist = async (next: Schemas['EarnImportDecisions']) => {
    setDec(next)
    await save.mutateAsync({ params: { path: { workspace: slug, id: imp.id } }, body: next })
    await refresh()
  }
  const firstTier = imp.tiers[0]?.name ?? ''
  const unmapped = imp.prices.find((p) => !p.tierId)
  return (
    <div className="flex flex-col gap-4">
      <Steps at={4} platform={imp.platform} />
      <TableScroll label={t(mi.pricesLabel)}>
        <table className="tbl" aria-label={t(mi.pricesLabel)}>
          <thead>
            <tr>
              <th scope="col">{t(mi.colPrice)}</th>
              <th scope="col" className="r">
                {t(mi.colSubs)}
              </th>
              <th scope="col">{t(mi.colMapsTo)}</th>
            </tr>
          </thead>
          <tbody>
            {imp.prices.map((p) => {
              const choice = (dec.prices ?? {})[p.priceId]
              const value = choice ? `${choice.tierId}|${choice.legacy ? 'legacy' : 'current'}` : p.tierId ? `${p.tierId}|${p.legacy ? 'legacy' : 'current'}` : ''
              return (
                <tr key={p.priceId}>
                  <td>
                    <div className="font-semibold">{pricePer(t, p.amountCents, p.currency, p.period)}</div>
                    <div className="ea-mono k">{p.priceId}</div>
                  </td>
                  <td className="r ea-num">{count(p.subscriptions)}</td>
                  <td>
                    {p.known ? (
                      t(mi.tierPrice, { tier: imp.tiers.find((x) => x.id === p.tierId)?.name ?? '', period: p.period })
                    ) : (
                      <Select
                        aria-label={t(mi.chooseTier)}
                        value={value}
                        onChange={(e) => {
                          const [tierId, kind] = e.target.value.split('|')
                          void persist({ ...dec, prices: { ...(dec.prices ?? {}), [p.priceId]: { tierId: tierId ?? '', legacy: kind === 'legacy' } } })
                        }}
                      >
                        <option value="">{t(mi.chooseTier)}</option>
                        {imp.tiers.map((x) => (
                          <Fragment key={x.id}>
                            <option value={`${x.id}|current`}>{t(mi.tierNew, { tier: x.name, period: p.period })}</option>
                            <option value={`${x.id}|legacy`}>{t(mi.tierLegacy, { tier: x.name, period: p.period })}</option>
                          </Fragment>
                        ))}
                      </Select>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </TableScroll>
      {unmapped ? <Note tone="warn">{t(mi.unmapped, { price: pricePer(t, unmapped.amountCents, unmapped.currency, unmapped.period), id: unmapped.priceId, tier: firstTier })}</Note> : null}
      <section className="flex flex-col gap-1">
        <h3 className="m-0 text-[14px] font-semibold">{t(mi.listsTitle)}</h3>
        <span>{imp.lists.map((l) => l.name).join(', ') || '—'}</span>
        <span className="k">{t(mi.listsHint, { platform: imp.platform })}</span>
      </section>
      <section className="flex flex-col gap-1">
        <h3 className="m-0 text-[14px] font-semibold">{t(mi.emailsTitle)}</h3>
        <Toggle
          label={t(mi.sendWelcome, { email: memberEmailName(t, 'welcome', firstTier) })}
          description={dec.sendWelcome ? t(mi.sendWelcomeOn) : t(mi.sendWelcomeOff)}
          checked={dec.sendWelcome}
          onChange={(e) => void persist({ ...dec, sendWelcome: e.target.checked })}
        />
      </section>
      <Footer>
        <Button onClick={onBack}>{t(mi.back)}</Button>
        <div className="flex gap-2">
          <Button onClick={onCancel}>{t(mi.cancel)}</Button>
          <Button variant="primary" onClick={onNext} disabled={imp.prices.some((p) => !p.tierId && !(dec.prices ?? {})[p.priceId])}>
            {t(mi.toReview)}
          </Button>
        </div>
      </Footer>
    </div>
  )
}

function ReviewStep({ imp, onBack, onClose }: { imp: Import; onBack: () => void; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const refresh = useRefresh()
  const start = $api.useMutation('post', `${EARN}/imports/{id}/start`)
  const jobId = `earn-import-${imp.id}`
  // The jobs tray follows the import while it runs.
  useEffect(() => {
    if (imp.status === 'importing') updateJob(slug, jobId, { status: 'running', detail: t(mi.importingOf, { done: imp.done, rows: imp.rows }) })
    if (imp.status === 'done') {
      updateJob(slug, jobId, { status: 'done', detail: t(mi.doneTitle, { rows: imp.rows, platform: imp.platform }), finishedAt: imp.finishedAt ?? new Date().toISOString() })
      void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
    }
    if (imp.status === 'failed') updateJob(slug, jobId, { status: 'failed', detail: imp.error ?? '', finishedAt: new Date().toISOString() })
  }, [imp.status, imp.done, imp.rows, imp.platform, imp.error, imp.finishedAt, slug, jobId, t, qc])
  const active = imp.active?.count ?? 0
  const ended = imp.ended?.count ?? 0
  const none = imp.none?.count ?? 0
  const st = imp.stats
  if (imp.status === 'importing') {
    return (
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-2">
          <span className="font-semibold">{t(mi.importing)}</span>
          <span className="ea-num">{t(mi.importingOf, { done: imp.done, rows: imp.rows })}</span>
        </div>
        <div className="ea-meter" role="progressbar" aria-valuemin={0} aria-valuemax={imp.rows} aria-valuenow={imp.done} aria-label={t(mi.importing)}>
          <span style={{ width: `${imp.rows ? (imp.done / imp.rows) * 100 : 0}%` }} />
        </div>
        <span className="k">{t(mi.importingNote, { platform: imp.platform })}</span>
        <h3 className="m-0 mt-2 text-[14px] font-semibold">{t(mi.meanwhile)}</h3>
        <ul className="m-0 flex flex-col gap-1 pl-5 text-[13px]">
          <li>{t(mi.meanwhile1)}</li>
          <li>{t(mi.meanwhile2)}</li>
          <li>{t(mi.meanwhile3)}</li>
        </ul>
        <Footer>
          <span />
          <Button onClick={onClose}>{t(mi.close)}</Button>
        </Footer>
      </div>
    )
  }
  if (imp.status === 'done' && st) {
    return (
      <div className="flex flex-col gap-3">
        <span className="k">{t(mi.doneLine, { when: imp.finishedAt ? `${dayMonthDate(imp.finishedAt)}, ${clock(imp.finishedAt)}` : '', name: imp.createdBy ?? '' })}</span>
        <dl className="ea-kv">
          <dt>{t(mi.reviewPaid)}</dt>
          <dd>{t(mi.donePaid, { count: st.paid })}</dd>
          <dt>{t(mi.reviewFree)}</dt>
          <dd>{t(mi.doneFree, { count: st.free })}</dd>
          <dt>{t(mi.reviewComp)}</dt>
          <dd>{st.compFirst && st.compLast ? t(mi.doneCompRange, { count: st.complimentary, first: dayMonthYear(st.compFirst), last: dayMonthYear(st.compLast) }) : t(mi.doneComp, { count: st.complimentary })}</dd>
          <dt>{t(mi.doneContacts)}</dt>
          <dd>{t(mi.doneContactsValue, { merged: st.merged, new: st.new, lists: imp.lists.map((l) => l.name).join(', ') || '—' })}</dd>
          <dt>{t(mi.reviewConsent)}</dt>
          <dd>{t(mi.consentValue, { platform: imp.platform })}</dd>
          <dt>{t(mi.donePrices)}</dt>
          <dd>{pricesMapped(t, imp, st.pricesMapped)}</dd>
          <dt>{t(mi.reviewEmails)}</dt>
          <dd>{t(mi.doneEmails, { count: st.emailsSent })}</dd>
          <dt>{t(mi.reviewStripe)}</dt>
          <dd>{t(mi.doneStripe)}</dd>
        </dl>
        <div className="ea-oldbox">
          <b>{t(mi.oldTitle)}</b>
          <span>{t(mi.oldText)}</span>
          {[t(mi.old1, { platform: imp.platform }), t(mi.old2, { platform: imp.platform }), t(mi.old3)].map((x) => (
            <label key={x} className="flex items-start gap-2">
              <input type="checkbox" className="mt-0.5" />
              <span>{x}</span>
            </label>
          ))}
        </div>
        <Footer>
          <span />
          <span className="flex gap-2">
            <ButtonLink to="/w/$workspace/earn" params={{ workspace: slug }}>
              {t(mi.overview)}
            </ButtonLink>
            <Button variant="primary" onClick={onClose}>
              {t(mi.seeMembers)}
            </Button>
          </span>
        </Footer>
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-3">
      <Steps at={5} platform={imp.platform} />
      {imp.status === 'failed' ? <ErrorPanel title={t(mi.failedTitle)}>{t(mi.failedText, { error: imp.error ?? '' })}</ErrorPanel> : null}
      <dl className="ea-kv">
        <dt>{t(mi.reviewPaid)}</dt>
        <dd>{t(mi.reviewPaidValue, { count: active })}</dd>
        <dt>{t(mi.reviewFree)}</dt>
        <dd>{t(mi.reviewFreeValue, { count: ended })}</dd>
        <dt>{t(mi.reviewComp)}</dt>
        <dd>{imp.decisions.noSubscription === 'complimentary' ? t(mi.reviewCompValue, { count: none }) : imp.decisions.noSubscription === 'free' ? t(mi.reviewFreeValue, { count: none }) : '—'}</dd>
        <dt>{t(mi.reviewConsent)}</dt>
        <dd>{t(mi.consentValue, { platform: imp.platform })}</dd>
        <dt>{t(mi.reviewEmails)}</dt>
        <dd>{imp.decisions.sendWelcome ? t(mi.reviewEmailsSome, { email: memberEmailName(t, 'welcome', imp.tiers[0]?.name ?? '') }) : t(mi.reviewEmailsNone)}</dd>
        <dt>{t(mi.reviewStripe)}</dt>
        <dd>{t(mi.reviewStripeValue)}</dd>
      </dl>
      <Footer>
        <Button onClick={onBack}>{t(mi.back)}</Button>
        <Button
          variant="primary"
          loading={start.isPending}
          onClick={() =>
            start.mutate(
              { params: { path: { workspace: slug, id: imp.id } } },
              {
                onSuccess: () => {
                  trackJob(slug, { id: jobId, kind: 'import', title: t(mi.importing), status: 'running', startedAt: new Date().toISOString(), detail: t(mi.importingNote, { platform: imp.platform }) })
                  void refresh()
                },
                onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
              },
            )
          }
        >
          {t(mi.start, { count: imp.rows })}
        </Button>
      </Footer>
    </div>
  )
}

/** "3 mapped: $8/month and $80/year to Insider, $200/year to Founding member". */
function pricesMapped(t: ReturnType<typeof useT>, imp: Import, n: number): string {
  const byTier = new Map<string, string[]>()
  for (const p of imp.prices) {
    if (!p.tierId || !p.period) continue
    const name = imp.tiers.find((x) => x.id === p.tierId)?.name ?? ''
    byTier.set(name, [...(byTier.get(name) ?? []), pricePer(t, p.amountCents, p.currency, p.period)])
  }
  if (!byTier.size) return t(mi.donePricesValue, { count: n })
  const parts = [...byTier.entries()].map(([tier, prices]) => t(mi.pricesTo, { prices: andList(prices), tier }))
  return t(mi.donePricesDetail, { count: n, detail: parts.join(', ') })
}
