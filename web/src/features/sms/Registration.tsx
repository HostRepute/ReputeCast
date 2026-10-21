import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading } from '@/components/ui/Feedback'
import { Field, Input, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { CopyRow, CountryName, dayYear, regionLabel, registrationBadge, useKindName, useRefreshSms } from './common'
import { m } from './dialogs.messages'

type Panel = Schemas['SmsRegistrationPanel']
type Item = Schemas['SmsRegistrationView']

/**
 * A sender's registration (SmsStates › Registration): a 10DLC number's
 * brand and campaign with what the provider's form asks for, a toll-free
 * number's verification, a sender ID country by country with the status
 * set by hand, and a rejection with the pages carriers check.
 */
export function RegistrationDialog({ senderId, open, onOpenChange }: { senderId: string; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const kindName = useKindName()
  const path = { workspace: slug, senderId }
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/senders/{senderId}/registration', { params: { path } })
  const p = q.data
  const s = p?.sender
  const countryPanel = s?.kind === 'sender_id'
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      size="lg"
      eyebrow={<span className="k">{countryPanel ? t(m.regEyebrowCountry) : t(m.regEyebrow)}</span>}
      title={
        s ? (
          <>
            <span className={s.kind === 'sender_id' ? undefined : 'mono'}>{s.label || s.value}</span>
            {s.kind === 'sender_id' ? ` · ${kindName(s.kind)} · ${s.provider}` : ` · ${kindName(s.kind)} · ${s.provider}`}
          </>
        ) : (
          t(m.regLoading)
        )
      }
      footer={<Button onClick={() => onOpenChange(false)}>{t(m.close)}</Button>}
    >
      {q.isError ? (
        <ErrorPanel title={errorMessage(q.error)} />
      ) : !p || !s ? (
        <Loading label={t(m.regLoading)}>
          <span className="k">{t(m.regLoading)}</span>
        </Loading>
      ) : s.kind === 'sender_id' ? (
        <ByCountry panel={p} />
      ) : s.brand || s.campaign ? (
        <TenDLC panel={p} />
      ) : (
        <Verification panel={p} />
      )}
    </Dialog>
  )
}

function readLine(p: Panel, t: ReturnType<typeof useT>): string {
  const name = p.sender.provider
  return p.readAt ? t(m.readFrom, { name, time: clock(p.readAt) }) : t(m.readFromNever, { name })
}

/** A 10DLC number: brand and campaign, the helper, the provider's console. */
function TenDLC({ panel }: { panel: Panel }) {
  const t = useT()
  const brand = panel.items.find((i) => i.kind === 'brand')
  const campaign = panel.items.find((i) => i.kind === 'campaign')
  const name = panel.sender.provider
  const rejected = campaign?.status === 'rejected' ? campaign : brand?.status === 'rejected' ? brand : undefined
  return (
    <div className="flex flex-col gap-3.5">
      {rejected ? <Rejected panel={panel} item={rejected} brand={brand} /> : null}
      <div className="flex flex-col rounded-lg border border-line">
        {brand ? (
          <RegRow
            first
            title={t(m.brandItem, { name: brand.name ?? '' })}
            sub={panel.readFrom ? readLine(panel, t) : brand.setBy ? t(m.setByHandBy, { name: brand.setBy }) : t(m.setByHand, { name })}
            item={brand}
          />
        ) : null}
        {campaign ? (
          <RegRow
            first={!brand}
            title={t(m.campaignItem, { name: campaign.name ?? '' })}
            sub={campaign.status === 'pending' ? t(m.carriersTake) : campaign.status === 'rejected' && campaign.reason ? t(m.rejectedItem, { reason: campaign.reason }) : readLine(panel, t)}
            item={campaign}
          />
        ) : null}
      </div>
      {campaign && campaign.status !== 'approved' ? <Helper panel={panel} /> : null}
      <div className="flex flex-wrap items-center justify-between gap-2.5">
        <span className="k" style={{ flex: '1 1 300px' }}>
          {t(m.consoleNote, { name })}
        </span>
        <ReadNow panel={panel} />
        {panel.console ? (
          <ButtonAnchor variant="primary" href={panel.console} newTab>
            {t(m.openIn, { name })}
          </ButtonAnchor>
        ) : null}
      </div>
    </div>
  )
}

function RegRow({ title, sub, item, first }: { title: string; sub: string; item: Item; first?: boolean }) {
  const t = useT()
  const b = registrationBadge(item, t)
  return (
    <div className="sms-srow" style={first ? { borderTop: 0 } : undefined}>
      <div className="tx">
        <span className="font-semibold">{title}</span>
        <span className="k">{sub}</span>
      </div>
      <div className="ct">
        <Badge tone={b.tone}>{b.text}</Badge>
      </div>
    </div>
  )
}

/** What the provider's campaign form asks for, ready to copy. */
function Helper({ panel }: { panel: Panel }) {
  const t = useT()
  const h = panel.helper
  const name = panel.sender.provider
  return (
    <div className="flex flex-col gap-2">
      <span className="font-semibold">{t(m.helperTitle, { name })}</span>
      <span className="k">{t(m.helperLead, { name })}</span>
      {h.optIn ? <CopyRow label={t(m.helperOptIn)} value={h.optIn} /> : null}
      {h.samples.map((s, i) => (
        <CopyRow key={i} label={t(m.helperSample, { n: i + 1 })} value={s} />
      ))}
      {h.helpReply ? <CopyRow label={t(m.helperHelp)} value={h.helpReply} /> : null}
      {h.optOutReply ? <CopyRow label={t(m.helperOptOut)} value={h.optOutReply} /> : null}
      {h.termsUrl ? <CopyRow label={t(m.helperTerms)} value={h.termsUrl} mono /> : null}
    </div>
  )
}

/** Read the status from the provider now (when its API reports it). */
function ReadNow({ panel }: { panel: Panel }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const reason = usePermissionReason('providers:manage')
  const read = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/senders/{senderId}/registration/read', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.readDone, { name: panel.sender.provider }) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  if (!panel.readFrom) return null
  return (
    <Button disabledReason={reason} loading={read.isPending} onClick={() => read.mutate({ params: { path: { workspace: slug, senderId: panel.sender.id } } })}>
      {t(m.readNow)}
    </Button>
  )
}

/** A toll-free number's verification: what was submitted, when it was verified. */
function Verification({ panel }: { panel: Panel }) {
  const t = useT()
  const item = panel.items.find((i) => i.kind === 'toll_free') ?? panel.sender.registration
  const d = panel.details
  const name = panel.sender.provider
  const b = registrationBadge(item, t)
  if (item.status === 'rejected') return <Rejected panel={panel} item={item} />
  return (
    <div className="flex flex-col gap-3.5">
      <div>
        <Badge tone={b.tone}>{b.text}</Badge>
      </div>
      <dl className="kv">
        {d.useCase ? (
          <>
            <dt>{t(m.detailUseCase)}</dt>
            <dd>{d.useCase}</dd>
          </>
        ) : null}
        {d.volume ? (
          <>
            <dt>{t(m.detailVolume)}</dt>
            <dd>{d.volume}</dd>
          </>
        ) : null}
        {d.optIn ? (
          <>
            <dt>{t(m.detailOptIn)}</dt>
            <dd>{d.optIn}</dd>
          </>
        ) : null}
        {d.terms ? (
          <>
            <dt>{t(m.detailTerms)}</dt>
            <dd>{d.terms}</dd>
          </>
        ) : null}
        {d.submitted ? (
          <>
            <dt>{t(m.detailSubmitted)}</dt>
            <dd>{d.submittedBy ? t(m.submittedBy, { date: dayYear(d.submitted), name: d.submittedBy }) : dayYear(d.submitted)}</dd>
          </>
        ) : null}
        {item.since && (item.status === 'verified' || item.status === 'approved') ? (
          <>
            <dt>{item.status === 'verified' ? t(m.detailVerified) : t(m.detailApproved)}</dt>
            <dd>{dayYear(item.since)}</dd>
          </>
        ) : null}
      </dl>
      <div className="flex flex-wrap items-center justify-between gap-2.5 border-t border-line-2 pt-3">
        <span className="k">{panel.readFrom ? t(m.tollFreeFoot, { name, region: regionLabel(panel.sender.countries) }) : t(m.setByHand, { name })}</span>
        <span className="flex flex-wrap gap-2">
          <ReadNow panel={panel} />
          {panel.console ? (
            <ButtonAnchor href={panel.console} newTab>
              {t(m.openIn, { name })}
            </ButtonAnchor>
          ) : null}
        </span>
      </div>
    </div>
  )
}

/** A rejection: the carrier's reason, the pages ReputeCast checked, resubmit. */
function Rejected({ panel, item, brand }: { panel: Panel; item: Item; brand?: Item }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const name = panel.sender.provider
  const [checks, setChecks] = useState(panel.checks)
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/senders/{senderId}/registration/check-pages', {
    onSuccess: (d) => {
      setChecks(d.checks)
      void refresh()
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  return (
    <div className="flex flex-col gap-3 rounded-lg border p-3.5" style={{ borderColor: '#FCA5A5' }}>
      <Note tone="bad">
        <span className="flex flex-col gap-1">
          <span className="font-bold">{t(m.rejectedTitle)}</span>
          <span>{t(m.rejectedText, { provider: name, name: item.name ?? panel.sender.label, reason: (item.reason ?? '').toLowerCase() })}</span>
        </span>
      </Note>
      <dl className="kv">
        {item.reason ? (
          <>
            <dt>{t(m.carrierReason)}</dt>
            <dd>{item.reason}</dd>
          </>
        ) : null}
        <dt>{t(m.checkedBy)}</dt>
        <dd>
          {checks.length
            ? checks
                .map((c) => {
                  const url = c.url.replace(/^https?:\/\//, '')
                  return c.status ? t(m.pageAnswered, { url, status: c.status }) : t(m.pageUnreachable, { url })
                })
                .join(' · ')
            : t(m.notCheckedYet)}
        </dd>
        {brand && brand.status === 'approved' ? (
          <>
            <dt>{t(m.brand)}</dt>
            <dd>{t(m.brandStill, { name: brand.name ?? '' })}</dd>
          </>
        ) : null}
      </dl>
      <div className="flex flex-wrap gap-2">
        <Button size="sm" loading={check.isPending} onClick={() => check.mutate({ params: { path: { workspace: slug, senderId: panel.sender.id } } })}>
          {t(m.checkPages)}
        </Button>
        {panel.console ? (
          <ButtonAnchor size="sm" variant="primary" href={panel.console} newTab>
            {t(m.resubmitIn, { name })}
          </ButtonAnchor>
        ) : null}
      </div>
    </div>
  )
}

/** A sender ID country by country; statuses the provider doesn't report are set here. */
function ByCountry({ panel }: { panel: Panel }) {
  const t = useT()
  const name = panel.sender.provider
  const [setting, setSetting] = useState<string | null>(null)
  const reason = usePermissionReason('providers:manage')
  const pending = panel.items.filter((i) => i.status === 'pending' || i.status === 'not_registered' || i.status === 'rejected')
  return (
    <div className="-mx-1 flex flex-col">
      {panel.items.map((i, n) => {
        const b = registrationBadge(i, t)
        const hold = i.status === 'pending' || i.status === 'rejected'
        const sub =
          i.status === 'not_supported'
            ? t(m.notSupportedUS)
            : i.status === 'allowed'
              ? t(m.noRegistrationNeeded)
              : i.source === 'manual' || !panel.readFrom
                ? t(m.setByHand, { name })
                : readLine(panel, t)
        return (
          <div key={i.country} className="sms-srow items-center" style={{ borderTop: n === 0 ? 0 : undefined, background: hold ? '#FFFBEB' : undefined }}>
            <div className="tx" style={{ flexDirection: 'row', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
              <CountryName code={i.country} />
              <span className="k">{sub}</span>
            </div>
            <div className="ct">
              <Badge tone={b.tone}>{b.text}</Badge>
            </div>
          </div>
        )
      })}
      {pending.length ? (
        <div className="border-t border-line-2 px-[18px] pt-3.5">
          <Note tone="warn" role="note">
            {pending.map((i) => (
              <div key={i.country} className="flex flex-col gap-2">
                <span>
                  {t(m.heldNote, { count: panel.held, country: regionLabel([i.country]), sender: panel.sender.value, provider: name })}
                </span>
                <div>
                  <Button size="sm" disabledReason={reason} onClick={() => setSetting(i.country)}>
                    {t(m.setStatusFor, { country: regionLabel([i.country]) })}
                  </Button>
                </div>
              </div>
            ))}
          </Note>
        </div>
      ) : null}
      {setting ? <SetStatus panel={panel} country={setting} onDone={() => setSetting(null)} /> : null}
    </div>
  )
}

/** Set a country's status by hand, with its date (audited). */
function SetStatus({ panel, country, onDone }: { panel: Panel; country: string; onDone: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const today = new Date().toISOString().slice(0, 10)
  const [status, setStatus] = useState<'approved' | 'pending' | 'rejected' | 'allowed' | 'not_registered'>('approved')
  const [on, setOn] = useState(today)
  const [why, setWhy] = useState('')
  const [error, setError] = useState<string>()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/sms/senders/{senderId}/registration', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.saved, { country: regionLabel([country]) }) })
      onDone()
    },
    onError: (e) => setError(errorMessage(e)),
  })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onDone()}
      title={t(m.setStatusTitle, { country: regionLabel([country]) })}
      description={t(m.setStatusText, { provider: panel.sender.provider })}
      footer={
        <>
          <Button onClick={onDone}>{t(m.cancel)}</Button>
          <Button
            variant="primary"
            loading={save.isPending}
            onClick={() =>
              save.mutate({
                params: { path: { workspace: slug, senderId: panel.sender.id } },
                body: { kind: 'sender_id', country, status, on, reason: status === 'rejected' ? why.trim() : undefined },
              })
            }
          >
            {t(m.save)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        {error ? <Note tone="bad">{error}</Note> : null}
        <Field label={t(m.status)}>
          <Select value={status} onChange={(e) => setStatus(e.target.value as typeof status)}>
            <option value="approved">{t(m.statusApproved)}</option>
            <option value="pending">{t(m.statusPending)}</option>
            <option value="rejected">{t(m.statusRejected)}</option>
            <option value="allowed">{t(m.statusAllowed)}</option>
            <option value="not_registered">{t(m.statusNotRegistered)}</option>
          </Select>
        </Field>
        <Field label={t(m.statusDate)}>
          <Input type="date" value={on} max={today} onChange={(e) => setOn(e.target.value)} />
        </Field>
        {status === 'rejected' ? (
          <Field label={t(m.statusReason)}>
            <Input value={why} onChange={(e) => setWhy(e.target.value)} />
          </Field>
        ) : null}
      </div>
    </Dialog>
  )
}
