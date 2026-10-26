import { useId, useMemo, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { DisabledReason } from '@/components/ui/DisabledReason'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { NoResults } from '@/components/ui/NoResults'
import { ConnectChannels } from '@/features/channels/ConnectChannels'
import { ReadOnlyNote } from '@/features/senders/ReadOnlyNote'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { deniedReason, useWorkspace } from '@/lib/session'
import { accountNote, priceAt, rank, readVolume, type Answers, type PriceAt } from './catalog-model'
import { mc } from './catalog.messages'
import { m } from './messages'
import { StepBadge, StepHeading, WizardBody, WizardLayout } from './Wizard'

type Entry = Schemas['ConnectCatalogEntry']
type Community = Schemas['ConnectCommunityEntry']

/** 7.2 step 1 (SetupCatalog): every adapter this build can connect. */
export function CatalogPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/connect/catalog', { params: { path: { workspace: slug } } })
  const [search, setSearch] = useState('')
  const [bulk, setBulk] = useState(false)
  const [webhooks, setWebhooks] = useState(false)
  const [dedicated, setDedicated] = useState(false)
  const searchId = useId()
  const data = q.data
  const shown = useMemo(() => {
    const needle = search.trim().toLowerCase()
    return (data?.builtin ?? []).filter(
      (e) =>
        (!needle || e.name.toLowerCase().includes(needle) || e.short.toLowerCase().includes(needle)) &&
        (!bulk || e.bulkAllowed) &&
        (!webhooks || e.webhooks) &&
        (!dedicated || e.dedicatedIps),
    )
  }, [data, search, bulk, webhooks, dedicated])
  // Help me choose (PR-13): the answers price every card and rank them.
  const [volumeText, setVolumeText] = useState('50,000')
  const [account, setAccount] = useState('')
  const [needIP, setNeedIP] = useState(false)
  const volume = readVolume(volumeText)
  const answers: Answers = { volume, account, dedicated: needIP }
  const ranked = useMemo(() => rank(data?.builtin ?? [], { volume, account, dedicated: needIP }), [data, volume, account, needIP])
  const best = ranked[0]
  const second = ranked[1]
  const reason = data && !data.canConnect ? deniedReason('providers:manage', data.helpers) : undefined
  const clear = () => {
    setSearch('')
    setBulk(false)
    setWebhooks(false)
    setDedicated(false)
  }
  return (
    <WizardLayout title={t(m.title)} badge={<StepBadge n={1} />}>
      <WizardBody current={1}>
        {data && !data.canConnect ? (
          <ReadOnlyNote msg={m.readOnlyCatalog} perm="providers:manage" helpers={data.helpers} />
        ) : null}
        <StepHeading title={t(mc.heading)}>{t(mc.lead)}</StepHeading>
        {data ? (
          <HelpMeChoose
            entries={data.builtin}
            volumeText={volumeText}
            onVolume={setVolumeText}
            account={account}
            onAccount={setAccount}
            needIP={needIP}
            onNeedIP={setNeedIP}
            answers={answers}
            best={best}
            second={second}
            reason={reason}
          />
        ) : null}
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Email · SMS · …: the other channels' catalogs (X21). */}
          <ConnectChannels current="email" />
          <label className="k flex items-center gap-2" htmlFor={searchId}>
            {t(common.search)}
            <input
              id={searchId}
              className="in w-[220px] max-md:w-auto max-md:min-w-0 max-md:flex-1"
              type="search"
              placeholder={t(mc.searchPlaceholder)}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={bulk} onChange={(e) => setBulk(e.target.checked)} />
            {t(mc.filterBulk)}
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={webhooks} onChange={(e) => setWebhooks(e.target.checked)} />
            {t(mc.filterWebhooks)}
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={dedicated} onChange={(e) => setDedicated(e.target.checked)} />
            {t(mc.filterDedicated)}
          </label>
          {data ? <span className="k ml-auto">{t(mc.counts, { builtin: count(data.builtin.length), community: count(data.community.length) })}</span> : null}
        </div>
        {q.isError ? (
          <ErrorPanel title={t(mc.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        ) : !data ? (
          <CardsSkeleton />
        ) : shown.length === 0 ? (
          <NoResults title={t(mc.noMatch)} actions={<Button onClick={clear}>{t(mc.clearFilters)}</Button>}>
            {search.trim() ? t(mc.noMatchSearch, { query: search.trim() }) : t(mc.noMatchFilters)}
          </NoResults>
        ) : (
          <div className="cn-grid grid gap-3.5 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(300px, 100%), 1fr))' }}>
            {shown.map((e) => (
              <ProviderCard
                key={e.adapter}
                entry={e}
                reason={reason}
                price={priceAt(e.priceList, answers.volume, answers.dedicated)}
                volume={answers.volume}
                standing={e.adapter === best?.adapter ? 'best' : e.adapter === second?.adapter ? 'second' : undefined}
              />
            ))}
          </div>
        )}
        {data && data.community.length > 0 ? <CommunityCard items={data.community} reason={reason} /> : null}
      </WizardBody>
    </WizardLayout>
  )
}

function ProviderCard({ entry: e, reason, price, volume, standing }: { entry: Entry; reason?: string; price: PriceAt; volume: number; standing?: 'best' | 'second' }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <article className={cn('card cn-prov', standing === 'best' && 'rec')} aria-labelledby={`prov-${e.adapter}`}>
      <div className="t">
        <h3 className="nm m-0" id={`prov-${e.adapter}`}>
          {e.name}
        </h3>
        {standing === 'best' ? (
          <Badge tone="ok" dot>
            {t(mc.recommended)}
          </Badge>
        ) : standing === 'second' ? (
          <Badge dot>{t(mc.runnerUp)}</Badge>
        ) : null}
      </div>
      <dl className="cn-facts m-0 flex flex-col gap-1.5 text-[13px]">
        <div>
          <dt>{t(mc.bulkLabel)}</dt>
          <dd className="bk">
            <BulkMark mark={e.bulkMark} />
            <span>{e.bulk || e.terms}</span>
          </dd>
        </div>
        <div>
          <dt>{volume > 0 ? t(mc.priceAtLabel, { volume: count(volume) }) : t(mc.priceAtNoVolume)}</dt>
          <dd>
            <span className="n">{price.amount}</span>
            {price.note ? ` · ${price.note}` : null}
          </dd>
        </div>
        {e.setup ? (
          <div>
            <dt>{t(mc.setupLabel)}</dt>
            <dd>{e.setup}</dd>
          </div>
        ) : null}
      </dl>
      <details className="cn-det">
        <summary>
          {t(mc.detailsLabel)}
          <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
            <path d="m4 6 4 4 4-4" />
          </svg>
        </summary>
        <div className="mt-1 flex flex-wrap gap-[5px]">
          {e.chips.map((c) => (
            <span key={c.label} className={cn('cn-cap', c.kind === 'yes' && 'y', c.kind === 'no' && 'n', c.kind === 'partial' && 'p')}>
              {c.kind === 'no' ? <T msg={mc.notSupported} values={{ capability: c.label, vh: (chunks) => <span className="vh">{chunks}</span> }} /> : c.label}
            </span>
          ))}
        </div>
        <p className="k m-0 mt-2">{e.pricing}</p>
      </details>
      <DisabledReason reason={reason}>
        <ButtonLink to="/w/$workspace/providers/connect/$adapter" params={{ workspace: slug, adapter: e.adapter }} variant={standing === 'best' ? 'primary' : 'default'}>
          {t(mc.connectProvider, { provider: e.short })}
        </ButtonLink>
      </DisabledReason>
    </article>
  )
}

/** The accounts "I already have an account with" offers, by adapter. */
const accountChoices = ['ses', 'postmark', 'mailgun', 'sendgrid', 'resend', 'sparkpost', 'mailjet', 'brevo', 'azureacs', 'postal']

/**
 * Help me choose (PR-13): three answers price every card at the volume and
 * name a recommendation and a runner-up, ranked by bulk-marketing terms,
 * then the capabilities ReputeCast uses, then price.
 */
function HelpMeChoose({
  entries,
  volumeText,
  onVolume,
  account,
  onAccount,
  needIP,
  onNeedIP,
  answers,
  best,
  second,
  reason,
}: {
  entries: Entry[]
  volumeText: string
  onVolume: (v: string) => void
  account: string
  onAccount: (v: string) => void
  needIP: boolean
  onNeedIP: (v: boolean) => void
  answers: Answers
  best?: Entry
  second?: Entry
  reason?: string
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const ids = { volume: useId(), account: useId(), ip: useId() }
  const accountName = (a: string) => (a === 'ses' ? t(mc.accountAWS) : a === 'azureacs' ? t(mc.accountAzureName) : a === 'postal' ? t(mc.accountPostalName) : (entries.find((e) => e.adapter === a)?.name ?? a))
  const choices = accountChoices.filter((a) => a === 'ses' || entries.some((e) => e.adapter === a))
  const note = accountNote(account, entries)
  const bestPrice = best ? priceAt(best.priceList, answers.volume, answers.dedicated) : undefined
  const secondPrice = second ? priceAt(second.priceList, answers.volume, answers.dedicated) : undefined
  const volume = count(answers.volume)
  return (
    <section className="card flex flex-col" aria-labelledby="choose-h">
      <div className="flex flex-col gap-0.5 border-b border-line-2 px-5 py-4">
        <h3 className="h2" id="choose-h">
          {t(mc.chooseTitle)}
        </h3>
        <span className="k">{t(mc.chooseLine)}</span>
      </div>
      <div className="flex flex-col gap-3.5 px-5 pt-4 pb-[18px]">
        <div className="grid items-end gap-3 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(200px, 100%), 1fr))' }}>
          <label className="lbl" htmlFor={ids.volume}>
            {t(mc.volumeLabel)}
            <input id={ids.volume} className="in" inputMode="numeric" value={volumeText} onChange={(e) => onVolume(e.target.value)} />
          </label>
          <label className="lbl" htmlFor={ids.account}>
            {t(mc.accountLabel)}
            <select id={ids.account} className="in" value={account} onChange={(e) => onAccount(e.target.value)}>
              <option value="">{t(mc.accountNone)}</option>
              {choices.map((a) => (
                <option key={a} value={a}>
                  {accountName(a)}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl" htmlFor={ids.ip}>
            {t(mc.dedicatedLabel)}
            <select id={ids.ip} className="in" value={needIP ? 'yes' : 'no'} onChange={(e) => onNeedIP(e.target.value === 'yes')}>
              <option value="no">{t(mc.no)}</option>
              <option value="yes">{t(mc.yes)}</option>
            </select>
          </label>
        </div>
        <div aria-live="polite">
          {best && bestPrice ? (
            <div className="cn-rec">
              <div className="flex flex-wrap items-center gap-2.5">
                <Badge tone="ok" icon="check">
                  {t(mc.recommended)}
                </Badge>
                <span className="text-[17px] font-bold">{best.name}</span>
              </div>
              <p className="m-0 text-[13px]">
                {answers.volume > 0
                  ? t(mc.recLine, { price: bestPrice.amount, volume, dedicated: answers.dedicated ? 'yes' : 'no', terms: lowerFirst(best.bulk || best.terms), setup: best.setup })
                  : t(mc.recNoVolume)}
              </p>
              {note ? <p className="m-0 text-[13px] text-ink-2">{note}</p> : null}
              <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
                <DisabledReason reason={reason}>
                  <ButtonLink variant="primary" to="/w/$workspace/providers/connect/$adapter" params={{ workspace: slug, adapter: best.adapter }}>
                    {t(mc.connectProvider, { provider: best.short })}
                  </ButtonLink>
                </DisabledReason>
                <span className="k">{t(mc.rankLine)}</span>
              </div>
              {second && secondPrice ? (
                <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t border-[#99f6e4] pt-2.5 text-[13px]">
                  <span className="k font-semibold">{t(mc.runnerUp)}</span>
                  <span className="min-w-0 flex-[1_1_260px]">
                    <span className="font-semibold">{second.name}</span> ·{' '}
                    {answers.volume > 0
                      ? t(mc.runnerUpLine, {
                          price: secondPrice.amount,
                          hasNote: secondPrice.note ? 'yes' : 'no',
                          note: secondPrice.note,
                          all: second.capabilityScore >= second.capabilityMax ? 'yes' : 'no',
                        })
                      : t(mc.runnerUpNoVolume)}
                  </span>
                  <DisabledReason reason={reason}>
                    <ButtonLink size="sm" to="/w/$workspace/providers/connect/$adapter" params={{ workspace: slug, adapter: second.adapter }}>
                      {t(mc.connectProvider, { provider: second.short })}
                    </ButtonLink>
                  </DisabledReason>
                </div>
              ) : null}
            </div>
          ) : (
            <span className="k">{t(mc.noRecommendation)}</span>
          )}
        </div>
      </div>
    </section>
  )
}

/** "Allowed after production access" in the middle of a sentence. */
function lowerFirst(s: string): string {
  return s.charAt(0).toLowerCase() + s.slice(1)
}

/** A card's bulk-marketing mark: a check when allowed, a warning when limited, a cross when not. */
function BulkMark({ mark }: { mark: Entry['bulkMark'] }) {
  if (mark === 'yes')
    return (
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="var(--ok-fg)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="m3.5 8.5 3 3 6-7" />
      </svg>
    )
  if (mark === 'no')
    return (
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="var(--bad-fg)" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
        <path d="m4.5 4.5 7 7M11.5 4.5l-7 7" />
      </svg>
    )
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="var(--warn-fg)" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6" />
      <path d="M8 5v3.5M8 11v.2" />
    </svg>
  )
}

function CommunityCard({ items, reason }: { items: Community[]; reason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-col" aria-labelledby="community-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h3 className="h2" id="community-h">
            {t(mc.communityTitle)}
          </h3>
          <span className="k">{t(mc.communityLine)}</span>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/providers/adapters" params={{ workspace: slug }}>
          {t(mc.manageAdapters)}
        </ButtonLink>
      </div>
      {/* As on the phone board: two 140 px columns (the provider cards above stay one per row there, where the board breaks names letter by letter). */}
      <div className="grid max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(300px, 100%), 1fr))' }}>
        {items.map((c, i) => (
          <div key={c.id} className={cn('flex flex-col gap-1.5 px-4.5 py-3.5', i < items.length - 1 && 'border-r border-line-2')}>
            <div className="flex items-center justify-between gap-2">
              <span className="font-semibold">{c.name}</span>
              <span className="mono k">{c.version}</span>
            </div>
            <span className="k">{c.summary}</span>
            {c.configure ? (
              <ButtonLink size="sm" className="self-start" to="/w/$workspace/providers/adapters" params={{ workspace: slug }}>
                {t(mc.configure)}
              </ButtonLink>
            ) : (
              <DisabledReason reason={reason ?? (c.adapter ? undefined : c.reason)}>
                <ButtonLink size="sm" className="self-start" to="/w/$workspace/providers/connect/$adapter" params={{ workspace: slug, adapter: c.adapter || c.name }}>
                  {t(mc.connect)}
                </ButtonLink>
              </DisabledReason>
            )}
          </div>
        ))}
      </div>
    </section>
  )
}

function CardsSkeleton() {
  const t = useT()
  return (
    <div className="grid gap-3.5 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(300px, 100%), 1fr))' }} aria-busy="true">
      <span className="vh" role="status">
        {t(mc.loading)}
      </span>
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <div key={i} className="card cn-prov" aria-hidden="true">
          <Skeleton w="45%" h={16} />
          <Skeleton w="90%" h={12} />
          <Skeleton w="80%" h={44} />
          <Skeleton w={120} h={36} r={8} />
        </div>
      ))}
    </div>
  )
}
