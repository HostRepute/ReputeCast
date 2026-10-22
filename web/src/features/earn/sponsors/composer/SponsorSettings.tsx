import type { Component } from 'grapesjs'
import { Link } from '@tanstack/react-router'
import { useEffect, useId, useMemo, type ReactNode } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Segmented } from '@/components/ui/Tabs'
import { formatList, useT } from '@/i18n'
import { shortDate } from '@/lib/format'
import { useCan } from '@/lib/session'
import { useComposer } from '@/features/composer/context'
import { money } from '@/features/earn/format'
import { Initials } from '../parts'
import { mcs } from './messages'
import { applySponsor, ATTR_LABEL, ATTR_LAYOUT, isCurrent, sponsorBlocks, sponsorOf, type SponsorBlock, type SponsorLayout } from './sponsor'
import '../sponsors.css'

/**
 * The Sponsor block of a component as the server fills it. With fill, the
 * block is filled again on the canvas when it doesn't hold what the booking
 * has now (SponsorSync does it for every block, so settings only read).
 */
function useSponsorBlock(comp: Component, fill = false) {
  const c = useComposer()
  const cfg = sponsorOf(comp)
  const campaign = c.kind === 'transactional' ? '' : c.campaign.id
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{id}/sponsor-block',
    { params: { path: { workspace: c.slug, id: campaign }, query: { layout: cfg?.layout ?? 'logo-left', label: cfg?.label ?? 'name' } } },
    { enabled: !!cfg && !!campaign, staleTime: 15_000 },
  )
  const blk = q.data
  // A block that doesn't hold what the booking has now is filled again (an
  // edit like any other: saved, undoable). Readers of the email can't edit.
  const stale = fill && !!blk && !!cfg && c.canEdit && !isCurrent(comp, blk)
  useEffect(() => {
    if (stale && blk) applySponsor(comp, blk)
  }, [stale, blk, comp])
  return q
}

/** Fills one block when the composer opens (Sponsor blocks that aren't selected). */
function Fill({ comp }: { comp: Component }) {
  useSponsorBlock(comp, true)
  return null
}

/** Keeps the email's Sponsor blocks filled from the booking while the composer is open. */
export function SponsorSync() {
  const c = useComposer()
  const body = c.editor?.getWrapper()?.findType('mj-body')[0]
  // eslint-disable-next-line react-hooks/exhaustive-deps -- rev covers canvas changes
  const blocks = useMemo(() => (c.kind === 'transactional' ? [] : sponsorBlocks(body)), [body, c.rev, c.kind])
  if (!c.canEdit) return null
  return (
    <>
      {blocks.map((b) => (
        <Fill key={b.cid} comp={b} />
      ))}
    </>
  )
}

function Prop({ title, children }: { title: ReactNode; children?: ReactNode }) {
  const id = useId()
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {title}
      </span>
      {children}
    </div>
  )
}

function statusBadge(t: ReturnType<typeof useT>, blk: SponsorBlock) {
  switch (blk.bookingStatus) {
    case 'booked':
      return <Badge tone="ok">{t(mcs.statusBooked)}</Badge>
    case 'sent':
      return <Badge tone="info">{t(mcs.statusSent)}</Badge>
    case 'hold':
      return <Badge tone="warn">{t(mcs.statusHold)}</Badge>
  }
  return null
}

/** The booking card: the issue, the sponsor and price, the creative's state and the way to the booking. */
function BookingCard({ blk }: { blk: SponsorBlock }) {
  const t = useT()
  const c = useComposer()
  const canManage = useCan('sponsors:manage')
  const date = blk.issueOn ? shortDate(blk.issueOn) : ''
  const booked = blk.state !== 'none'
  const head = blk.issue
    ? booked || blk.bookingStatus === 'hold'
      ? t(mcs.issueLine, { issue: blk.issue, date, slot: blk.slotName ?? '' })
      : t(mcs.issueOpen, { issue: blk.issue, date })
    : t(mcs.noSlot)
  const link = blk.bookingId ? `/w/${c.slug}/earn/sponsors?booking=${blk.bookingId}` : ''
  return (
    <div className="sp-cblk">
      <div className="sp-cblk-head">
        <span className="font-semibold">{head}</span>
        {statusBadge(t, blk)}
      </div>
      {booked && blk.sponsor ? (
        <div className="sp-cblk-row">
          <Initials initials={blk.initials ?? ''} size="sm" />
          <span className="font-semibold">{blk.sponsor}</span>
          {blk.canSeePrice && blk.priceCents !== undefined ? <span className="k ml-auto">{money(blk.priceCents, blk.currency)}</span> : null}
        </div>
      ) : null}
      {booked ? (
        <div className="sp-two">
          {blk.state === 'waiting' ? <Badge tone="warn">{t(mcs.creativeWaiting)}</Badge> : <Badge tone="ok">{t(mcs.creativeReady)}</Badge>}
          {blk.state === 'waiting' && blk.creativeDue ? <span className="k">{t(mcs.due, { date: shortDate(blk.creativeDue) })}</span> : null}
        </div>
      ) : (
        <span className="k">
          {blk.bookingStatus === 'hold' && blk.issue
            ? t(mcs.noneHold, { issue: blk.issue })
            : blk.issue
              ? t(mcs.none, { issue: blk.issue })
              : blk.slotId
                ? t(mcs.noDate)
                : t(mcs.noneNoIssue)}
        </span>
      )}
      {blk.canSeePrice ? (
        link ? (
          <a className="sp-cblk-link" href={link} target="_blank" rel="noopener noreferrer">
            {t(mcs.openBooking)}
            <Icon name="external" size={12} />
          </a>
        ) : canManage ? (
          <Link className="sp-cblk-link" to="/w/$workspace/earn/sponsors" params={{ workspace: c.slug }} search={blk.issue ? {} : { tab: 'slots' }} target="_blank">
            {blk.issue ? t(mcs.bookIssue, { issue: blk.issue }) : t(mcs.setUpSlot)}
            <Icon name="external" size={12} />
          </Link>
        ) : null
      ) : (
        <span className="k">{blk.helpers.length ? t(mcs.noPrice, { helpers: formatList(blk.helpers, 'disjunction') }) : t(mcs.noPriceAlone)}</span>
      )}
    </div>
  )
}

/**
 * The Sponsor block's settings (ComposerEarnBlocks): the booking it fills
 * itself from, the layout, the label (always shown, optionally naming the
 * sponsor) and the link with its UTM parameters. Visibility follows, as
 * for any block.
 */
export function SponsorSettings({ c: comp }: { c: Component }) {
  const t = useT()
  const c = useComposer()
  const cfg = sponsorOf(comp)
  const q = useSponsorBlock(comp)
  const ids = useId()
  if (!cfg || c.kind === 'transactional') return null
  const blk = q.data
  const disabled = !c.canEdit
  const layouts: { value: SponsorLayout; label: string }[] = [
    { value: 'logo-left', label: t(mcs.layoutLogo) },
    { value: 'image-top', label: t(mcs.layoutImage) },
    { value: 'text', label: t(mcs.layoutText) },
  ]
  const utm = blk ? Object.entries(blk.creative.utm ?? {}).filter(([, v]) => v) : []
  const booked = !!blk && blk.state !== 'none'
  return (
    <>
      <div className="flex flex-col gap-0.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
        <span className="font-semibold">{t(mcs.title)}</span>
        <span className="k">{t(mcs.sub)}</span>
      </div>
      <Prop title={t(mcs.booking)}>
        {blk ? <BookingCard blk={blk} /> : <span className="k">{q.error ? t(mcs.failed, { error: errorMessage(q.error) }) : t(mcs.loading)}</span>}
      </Prop>
      {blk && !blk.canSeePrice && booked ? (
        <Prop title={t(mcs.creative)}>
          <span className="k">{t(mcs.creativeNote)}</span>
        </Prop>
      ) : null}
      <Prop title={t(mcs.layout)}>
        <div>
          <Segmented
            label={t(mcs.layout)}
            value={cfg.layout}
            items={disabled ? layouts.filter((l) => l.value === cfg.layout) : layouts}
            onChange={(v) => comp.addAttributes({ [ATTR_LAYOUT]: v })}
          />
        </div>
      </Prop>
      <Prop title={t(mcs.label)}>
        <div className="sp-clabel">
          <input type="checkbox" checked disabled aria-label={t(mcs.labelValue)} aria-describedby={`${ids}-always`} />
          <div className="flex flex-col gap-0.5">
            <span className="font-semibold">{t(mcs.labelValue)}</span>
            <span className="k" id={`${ids}-always`}>
              {t(mcs.labelNote)}
            </span>
          </div>
        </div>
        <div className="sp-cname">
          <div className="flex flex-col gap-0.5">
            <span className="font-semibold">{t(mcs.nameSponsor)}</span>
            <span className="k" id={`${ids}-reads`}>
              {blk?.sponsor ? t(mcs.nameReads, { sponsor: blk.sponsor }) : t(mcs.nameReadsNone)}
            </span>
          </div>
          <input
            type="checkbox"
            className="tg"
            aria-label={t(mcs.nameSponsor)}
            aria-describedby={`${ids}-reads`}
            checked={cfg.label === 'name'}
            disabled={disabled}
            onChange={(e) => comp.addAttributes({ [ATTR_LABEL]: e.target.checked ? 'name' : 'plain' })}
          />
        </div>
      </Prop>
      <Prop title={t(mcs.link)}>
        <span className="k text-ink-2">{!blk || !booked ? t(mcs.linkNone) : blk.state === 'waiting' ? t(mcs.linkWaiting) : t(mcs.linkReady)}</span>
        {utm.length ? (
          <div className="flex flex-wrap gap-1">
            {utm.map(([k, v]) => (
              <span key={k} className="sp-utm">
                {'utm_'}{k}={v}
              </span>
            ))}
          </div>
        ) : null}
        {blk?.bookingId && blk.canSeePrice ? (
          <a className="btn sm self-start" href={`/w/${c.slug}/earn/sponsors?booking=${blk.bookingId}`} target="_blank" rel="noopener noreferrer">
            {t(mcs.editUtm)}
          </a>
        ) : null}
        <span className="k">{t(mcs.tracking)}</span>
      </Prop>
    </>
  )
}

/** Under Visibility: who the sponsor report counts, and what holds the issue. */
export function SponsorFoot({ c: comp }: { c: Component }) {
  const t = useT()
  const c = useComposer()
  const cfg = sponsorOf(comp)
  const q = useSponsorBlock(comp)
  if (!cfg || c.kind === 'transactional') return null
  const blk = q.data
  return (
    <>
      <div className="prop">
        <span className="k text-ink-2">{t(mcs.visibilityNote)}</span>
      </div>
      {blk?.state === 'waiting' && blk.issue ? (
        <div style={{ padding: '10px 12px' }}>
          <div className="note w" role="note">
            <Icon name="alertCircle" size={16} />
            <div className="min-w-0 flex-1">{t(mcs.holds, { issue: blk.issue })}</div>
          </div>
        </div>
      ) : null}
    </>
  )
}
