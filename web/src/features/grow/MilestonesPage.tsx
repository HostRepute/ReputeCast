import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Input } from '@/components/ui/Field'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { count, dayMonthDate } from '@/lib/format'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { RewardEditor } from './RewardEditor'
import { m } from './messages'
import { mm } from './milestones.messages'
import { mr } from './referrals.messages'
import { GROW, GrowPage, LoadError, ProgramNav, ReadOnlyNote, fileKind, fileSize, mono, orNames, useGrowRefresh, useReferralPrograms } from './shared'

export type MilestoneList = Schemas['ReferralMilestoneList']
export type Milestone = Schemas['ReferralMilestoneView']

export interface MilestonesSearch {
  program?: string
  edit?: string
}

/**
 * Referrals › Milestones (ReferralMilestones, ReferralStates): the
 * milestones with their rewards, health and emails, the files, the "To
 * fulfill" queue, and the reward editor (?edit=<id> or ?edit=new).
 */
export function MilestonesPage({ search, onSearch }: { search: MilestonesSearch; onSearch: (s: MilestonesSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const { list, program, q: pq } = useReferralPrograms(search.program)
  const q = $api.useQuery('get', `${GROW}/referrals/{program}/milestones`, { params: { path: { workspace: slug, program: program?.id ?? '' } } }, { enabled: Boolean(program) })
  const d = q.data
  const canEdit = useCan('grow:edit')
  const err = pq.error ?? q.error
  return (
    <GrowPage tab="referrals" documentTitle={t(m.docReferrals)}>
      <div className="gw">
        <ProgramNav current="milestones" programId={search.program} />
        {!list || (program && !d) ? (
          err ? (
            <LoadError error={err} title={t(mm.loadError)} onRetry={() => void (pq.isError ? pq.refetch() : q.refetch())} />
          ) : (
            <Loading label={t(mm.loadingLabel)}>
              <Skeleton w="100%" h={260} r={10} />
            </Loading>
          )
        ) : !program || !d ? (
          <section className="card gw-pad k">{t(mr.noneText)}</section>
        ) : (
          <>
            {canEdit ? null : (
              <ReadOnlyNote>
                <T msg={d.helpers.edit.length ? mm.readOnlyText : mm.readOnlyTextNoHelpers} values={{ helpers: orNames(d.helpers.edit), mono }} />
              </ReadOnlyNote>
            )}
            <MilestonesTable d={d} onEdit={(id) => onSearch({ ...search, edit: id })} />
            <div className="gw-g2 gw-st">
              <FilesCard d={d} />
              <FulfillCard d={d} />
            </div>
            {search.edit ? (
              <RewardEditor
                d={d}
                milestone={search.edit === 'new' ? undefined : d.items.find((x) => x.id === search.edit)}
                onClose={() => onSearch({ ...search, edit: undefined })}
              />
            ) : null}
          </>
        )}
      </div>
    </GrowPage>
  )
}

const typeIcon: Record<Milestone['type'], IconName> = { download: 'download', codes: 'tag', manual: 'send', coupon: 'star', webhook: 'webhook' }

export function typeLabel(t: ReturnType<typeof useT>, type: Milestone['type']): string {
  switch (type) {
    case 'download':
      return t(mm.typeDownload)
    case 'codes':
      return t(mm.typeCodes)
    case 'manual':
      return t(mm.typeManual)
    case 'coupon':
      return t(mm.typeCoupon)
    case 'webhook':
      return t(mm.typeWebhook)
  }
  return type
}

/** "PDF · 2.4 MB · sticker-pack.pdf", "Offer “Referral reward · 3 months of Insider free”". */
export function rewardLine(t: ReturnType<typeof useT>, x: Milestone): string {
  switch (x.type) {
    case 'download':
      if (x.file) return t(mm.lineFile, { type: fileKind(x.file.contentType), size: fileSize(x.file.size), name: x.file.name })
      return x.downloadUrl ? t(mm.lineUrl, { url: x.downloadUrl.replace(/^https?:\/\//, '') }) : t(mm.lineNoFile)
    case 'codes':
      return x.codesUploaded ? t(mm.lineCodes, { uploaded: x.codesUploaded, left: count(x.codesLeft) }) : t(mm.lineCodesNone)
    case 'manual':
      return x.askAddress ? t(mm.lineManualAddress) : t(mm.lineManual)
    case 'coupon':
      return x.couponOfferName ? t(mm.lineCoupon, { offer: x.couponOfferName }) : t(mm.lineCouponNone)
    case 'webhook':
      return t(mm.lineWebhook, { url: x.webhookUrl.replace(/^https?:\/\//, '') })
  }
  return ''
}

function usedText(t: ReturnType<typeof useT>, x: Milestone): string {
  const n = { count: count(x.stats.used) }
  if (x.type === 'download') return t(mm.usedDownload, n)
  if (x.type === 'coupon') return t(mm.usedCoupon, n)
  if (x.type === 'manual') return t(mm.usedManual, n)
  return t(mm.usedOther, n)
}

/** The health badge and the line under it. */
function Health({ x }: { x: Milestone }) {
  const t = useT()
  const h = x.health
  switch (h.kind) {
    case 'codes_low':
      return (
        <>
          <Badge tone="warn">{t(mm.codesLow, { count: h.codesLeft })}</Badge>
          {h.runOutDays ? <div className="k mt-[3px]">{t(mm.runOut, { days: Math.round(h.runOutDays) })}</div> : null}
        </>
      )
    case 'codes_out':
      return (
        <>
          <Badge tone="bad">{t(mm.codesOut)}</Badge>
          <div className="k mt-[3px]">{t(mm.codesOutWaiting, { count: h.waiting })}</div>
        </>
      )
    case 'webhook_failing':
      return (
        <>
          <Badge tone="bad">{t(mm.webhookFailing, { count: count(x.stats.failed || h.waiting) })}</Badge>
          {h.lastStatus ? <div className="k mt-[3px]">{t(mm.webhookLast, { status: h.lastStatus })}</div> : null}
        </>
      )
    case 'stripe_missing':
      return <Badge tone="warn">{t(mm.stripeMissing)}</Badge>
    case 'file_missing':
      return <Badge tone="bad">{t(mm.fileMissing)}</Badge>
  }
  return (
    <>
      <Badge tone="ok">{t(mm.healthy)}</Badge>
      {x.type === 'coupon' ? <div className="k mt-[3px]">{t(mm.stripeConnected)}</div> : null}
    </>
  )
}

function RewardTile({ x, size = 40 }: { x: Milestone; size?: number }) {
  if (x.image) {
    return (
      <span className="gw-mtile" style={{ width: size, height: size }} aria-hidden="true">
        <img src={x.image.url} alt="" />
      </span>
    )
  }
  return (
    <span className="gw-mtile" style={{ width: size, height: size }} aria-hidden="true">
      <Icon name={typeIcon[x.type]} size={16} />
    </span>
  )
}

/** The milestones with their reward, type, numbers, health and email. */
function MilestonesTable({ d, onEdit }: { d: MilestoneList; onEdit: (id: string) => void }) {
  const t = useT()
  const canEdit = useCan('grow:edit')
  return (
    <section className="card">
      <CardSection
        title={t(mm.title, { count: d.items.length })}
        caption={d.sender ? t(mm.caption, { list: d.listName, sender: d.sender }) : t(mm.captionNoSender, { list: d.listName })}
      />
      {d.items.length ? (
        <>
          <div className="max-md:hidden">
            <TableScroll label={t(mm.tableLabel)}>
              <table className="tbl gw-tbl">
                <thead>
                  <tr>
                    <th scope="col">{t(mm.colMilestone)}</th>
                    <th scope="col">{t(mm.colReward)}</th>
                    <th scope="col">{t(mm.colType)}</th>
                    <th scope="col" className="r">
                      {t(mm.colReached)}
                    </th>
                    <th scope="col" className="r">
                      {t(mm.colSent)}
                    </th>
                    <th scope="col">{t(mm.colUsed)}</th>
                    <th scope="col">{t(mm.colHealth)}</th>
                    <th scope="col">{t(mm.colEmail)}</th>
                    <th scope="col">
                      <span className="vh">{t(mm.edit)}</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {d.items.map((x) => (
                    <tr key={x.id}>
                      <td>
                        <span className="flex flex-col">
                          <span className="gw-num text-[18px] leading-[1.1] font-bold">{x.count}</span>
                          <span className="k">{t(m.referralUnit, { count: x.count })}</span>
                        </span>
                      </td>
                      <td>
                        <span className="flex min-w-0 items-center gap-2.5">
                          <RewardTile x={x} />
                          <span className="flex min-w-0 flex-col gap-px">
                            <span className="gw-tt2">{x.rewardName}</span>
                            <span className="k">{rewardLine(t, x)}</span>
                          </span>
                        </span>
                      </td>
                      <td>
                        <Badge tone="neutral">{typeLabel(t, x.type)}</Badge>
                      </td>
                      <td className="r gw-num">{count(x.stats.reached)}</td>
                      <td className="r gw-num">{count(x.stats.sent)}</td>
                      <td className="nw">{usedText(t, x)}</td>
                      <td>
                        <Health x={x} />
                      </td>
                      <td>
                        <span className="flex flex-col items-start gap-0.5">
                          <Badge tone={x.email.on ? 'ok' : 'neutral'}>{x.email.on ? t(mm.emailOn) : t(mm.emailOff)}</Badge>
                          <span className="k">{x.email.on ? x.email.subject : t(mm.emailOffText)}</span>
                        </span>
                      </td>
                      <td className="r">
                        <Button size="sm" onClick={() => onEdit(x.id)}>
                          {canEdit ? t(mm.edit) : t(mm.view)}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </TableScroll>
          </div>
          <div className="md:hidden">
            {d.items.map((x) => (
              <div key={x.id} className="gw-prow" style={{ alignItems: 'flex-start' }}>
                <RewardTile x={x} size={36} />
                <span className="tx">
                  <span className="gw-tt2">
                    {x.rewardName} <span className="k font-medium">· {t(m.referralsN, { count: x.count })}</span>
                  </span>
                  <span className="k">{rewardLine(t, x)}</span>
                  <span className="k">{t(mm.phoneLine, { reached: count(x.stats.reached), used: usedText(t, x) })}</span>
                  <span className="mt-1">
                    <Health x={x} />
                  </span>
                </span>
                <Button size="sm" onClick={() => onEdit(x.id)}>
                  {canEdit ? t(mm.edit) : t(mm.view)}
                </Button>
              </div>
            ))}
          </div>
        </>
      ) : (
        <div className="gw-pad k">{t(mm.empty)}</div>
      )}
      <div className="gw-foot">
        <span className="k">{t(mm.foot)}</span>
        {canEdit ? (
          <Button size="sm" variant="primary" icon="plus" onClick={() => onEdit('new')}>
            {t(mm.add)}
          </Button>
        ) : null}
      </div>
    </section>
  )
}

/** The program's files for download rewards. */
function FilesCard({ d }: { d: MilestoneList }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const refresh = useGrowRefresh()
  const used = new Set(d.items.map((x) => x.file?.id).filter(Boolean))
  const remove = async (f: Schemas['ReferralFile']) => {
    try {
      await http.DELETE('/api/v1/workspaces/{workspace}/grow/referrals/{program}/files/{file}', { params: { path: { workspace: slug, program: d.programId, file: f.id } } })
      toast({ title: t(mm.fileRemoved, { name: f.name }) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  return (
    <section className="card">
      <CardSection title={t(mm.filesTitle)} caption={t(mm.filesCaption)} />
      <div className="gw-pad" style={{ paddingTop: 6, paddingBottom: 6 }}>
        {d.files.length ? (
          d.files.map((f) => (
            <div className="gw-lrow" key={f.id}>
              <span className="gw-mtile" style={{ width: 40, height: 40 }} aria-hidden="true">
                <Icon name="download" size={16} />
              </span>
              <span className="tx">
                <span className="gw-tt2 mono text-[12.5px]">{f.name}</span>
                <span className="k">
                  {f.uploadedBy
                    ? t(mm.fileLine, { size: fileSize(f.size), by: f.uploadedBy, date: dayMonthDate(f.uploadedAt) })
                    : t(mm.fileLineNoBy, { size: fileSize(f.size), date: dayMonthDate(f.uploadedAt) })}
                </span>
              </span>
              <span className="k gw-num text-right">{t(mm.downloads, { count: f.downloads })}</span>
              {canEdit && !used.has(f.id) ? (
                <Button size="sm" variant="ghost" onClick={() => void remove(f)}>
                  {t(mm.removeFile)}
                </Button>
              ) : null}
            </div>
          ))
        ) : (
          <span className="k">{t(mm.filesEmpty)}</span>
        )}
      </div>
      <div className="gw-foot">
        <span className="k">{t(mm.filesFoot)}</span>
      </div>
    </section>
  )
}

/** Manual rewards waiting to be sent by hand, with their addresses. */
function FulfillCard({ d }: { d: MilestoneList }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canReview = useCan('grow:review')
  const canExport = useCan('subscribers:export')
  const refresh = useGrowRefresh()
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState<string>()
  const manualUsed = d.items.some((x) => x.type === 'manual')
  const items = d.toFulfill
  const fulfill = async (it: Schemas['ReferralFulfillItem']) => {
    setBusy(it.rewardId)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/grow/rewards/{reward}/fulfill', {
        params: { path: { workspace: slug, reward: it.rewardId } },
        body: { note: notes[it.rewardId] ?? '' },
      })
      toast({ kind: 'success', title: t(mm.fulfilled, { reward: it.rewardName, name: it.referrer }), detail: t(mm.fulfilledDetail) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  const exportCsv = async () => {
    try {
      const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/grow/referrals/${d.programId}/fulfill.csv`, { credentials: 'same-origin' })
      if (!r.ok) throw new Error(r.statusText)
      const blob = await r.blob()
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = 'to-fulfill.csv'
      a.click()
      URL.revokeObjectURL(a.href)
      toast({ kind: 'success', title: t(mm.exported) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  return (
    <section className="card">
      <CardSection title={t(mm.fulfillTitle, { count: items.length })} caption={t(mm.fulfillCaption)} />
      {items.length ? (
        <>
          <TableScroll label={t(mm.fulfillLabel)}>
            <table className="tbl gw-tbl">
              <thead>
                <tr>
                  <th scope="col">{t(mm.colReferrer)}</th>
                  <th scope="col">{t(mm.colMilestone)}</th>
                  <th scope="col">{t(mm.colReachedOn)}</th>
                  <th scope="col">{t(mm.colAddress)}</th>
                  <th scope="col">{t(mm.colStatus)}</th>
                </tr>
              </thead>
              <tbody>
                {items.map((it) => (
                  <tr key={it.rewardId}>
                    <td>
                      <span className="flex items-center gap-2.5">
                        <Avatar name={it.referrer} size={28} color="muted" />
                        <span className="flex flex-col leading-[1.3]">
                          <span className="font-semibold">{it.referrer}</span>
                          {it.email ? <span className="k mono">{it.email}</span> : null}
                        </span>
                      </span>
                    </td>
                    <td>
                      <span className="flex flex-col">
                        <span>{t(m.referralsN, { count: it.milestone })}</span>
                        <span className="k">{it.rewardName}</span>
                      </span>
                    </td>
                    <td className="nw">{dayMonthDate(it.reachedAt)}</td>
                    <td>
                      {it.address ? (
                        <address className="text-[13px] not-italic">
                          {it.address.name}
                          <br />
                          {it.address.street}
                          <br />
                          {it.address.city}, {it.address.country}
                        </address>
                      ) : (
                        <span className="k">{d.showAddresses ? t(mm.noAddress) : t(mm.addressHidden)}</span>
                      )}
                    </td>
                    <td>
                      <span className="flex flex-col items-start gap-1.5">
                        <Badge tone="warn">{t(mm.statusToFulfill)}</Badge>
                        {canReview ? (
                          <>
                            <Input
                              aria-label={t(mm.noteFor, { name: it.referrer })}
                              placeholder={t(mm.notePlaceholder)}
                              value={notes[it.rewardId] ?? ''}
                              maxLength={500}
                              onChange={(e) => setNotes({ ...notes, [it.rewardId]: e.target.value })}
                            />
                            <Button size="sm" loading={busy === it.rewardId} onClick={() => void fulfill(it)}>
                              {t(mm.markFulfilled)}
                            </Button>
                          </>
                        ) : null}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
          <div className="gw-foot">
            <span className="k">{t(mm.fulfillFoot)}</span>
            <span className="flex items-center gap-2.5">
              <span className="k">
                <T msg={mm.exportNeeds} values={{ mono }} />
              </span>
              <Button size="sm" icon="download" disabledReason={canExport ? undefined : deniedReason('subscribers:export')} onClick={() => void exportCsv()}>
                {t(mm.exportCsv)}
              </Button>
            </span>
          </div>
        </>
      ) : (
        <div className="gw-pad flex items-start gap-3">
          <span className="gw-ti" style={{ background: 'var(--neutral-bg)', color: 'var(--ink-2)' }} aria-hidden="true">
            <Icon name="send" size={13} />
          </span>
          <span className="flex flex-col gap-0.5">
            <span className="font-semibold">{t(mm.fulfillNoneTitle)}</span>
            <span className="k">{manualUsed ? t(mm.fulfillNoneText) : t(mm.fulfillNoneTextUnused)}</span>
          </span>
        </div>
      )}
    </section>
  )
}
