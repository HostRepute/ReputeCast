import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { EARN } from './EarnPage'
import { memberEmailName } from './MemberPanel'
import { mm } from './members.messages'

type Reason = Schemas['EarnComplimentaryInput']['reason']
const reasons: Reason[] = ['contributor', 'team', 'press', 'partner', 'giveaway', 'other']
const emailLike = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/** Who gets the access: a contact (preset or found), or a new address. */
export type Person = { id: string; name: string; email: string; lists?: string[] }

/**
 * Give complimentary access (a contact or a new address, a tier, a reason,
 * an end date) or, with `member`, change it (EarnMemberPanelStates).
 */
export function ComplimentaryDialog({ tiers, member, person, onClose }: { tiers?: Schemas['EarnListRef'][]; member?: Schemas['EarnMember']; person?: Person; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const plans = $api.useQuery('get', `${EARN}/plans`, { params: { path: { workspace: slug } } }, { enabled: !tiers })
  const emails = $api.useQuery('get', `${EARN}/settings/member-emails`, { params: { path: { workspace: slug } } }, { enabled: !member })
  const options =
    tiers?.map((x) => ({ id: x.id, name: x.name, short: x.name })) ??
    plans.data?.tiers.filter((x) => x.visibility !== 'archived').map((x) => ({ id: x.id, name: x.name, short: x.shortName })) ??
    []
  const [chosen, setChosen] = useState<Person | undefined>(person)
  const [search, setSearch] = useState('')
  const [tier, setTier] = useState(member ? '' : (options[0]?.id ?? ''))
  const [until, setUntil] = useState(member?.grantUntil ? member.grantUntil.slice(0, 10) : '')
  const [dated, setDated] = useState(Boolean(member?.grantUntil))
  const [reason, setReason] = useState<Reason>((member?.source.reason as Reason) || 'contributor')
  const [note, setNote] = useState(member?.grantNote ?? '')
  const [welcome, setWelcome] = useState(true)
  const [error, setError] = useState<Record<string, string>>({})
  const q = search.trim()
  const found = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts', { params: { path: { workspace: slug }, query: { q, pageSize: 5 } } }, { enabled: !member && !chosen && q.length >= 2 })
  const give = $api.useMutation('post', `${EARN}/members/complimentary`)
  const change = $api.useMutation('put', `${EARN}/members/{id}/complimentary`)
  const tierId = tier || (member ? (options.find((o) => o.name === member.tierName)?.id ?? options[0]?.id ?? '') : (options[0]?.id ?? ''))
  const tierShort = options.find((o) => o.id === tierId)?.short ?? ''
  const exact = found.data?.items.find((c) => c.email.toLowerCase() === q.toLowerCase())
  const newAddress = !member && !chosen && emailLike.test(q) && !exact && !found.isFetching ? q : ''
  const welcomeEmail = emails.data?.emails.find((e) => e.email === 'welcome')
  const welcomeLive = welcomeEmail?.status === 'live'
  const done = (title: string) => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members/{id}`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/contacts/{contactId}`] })
    toast({ kind: 'success', title })
    onClose()
  }
  const fail = (e: unknown) => {
    if (e instanceof ApiError && e.problem.errors?.length) {
      const next: Record<string, string> = {}
      for (const x of e.problem.errors) next[(x.location ?? '').replace(/^body\./, '')] = x.message ?? ''
      setError(next)
      return
    }
    toast({ kind: 'error', title: errorMessage(e) })
  }
  const submit = () => {
    const untilValue = dated ? until : ''
    if (member) {
      change.mutate({ params: { path: { workspace: slug, id: member.id } }, body: { tierId, until: untilValue, reason, note } }, { onSuccess: () => done(t(mm.compChanged)), onError: fail })
      return
    }
    give.mutate(
      {
        params: { path: { workspace: slug } },
        body: { contactId: chosen?.id, email: chosen ? undefined : newAddress, tierId, until: untilValue, reason, note, sendWelcome: welcome && welcomeLive },
      },
      { onSuccess: () => done(t(mm.compGiven, { who: chosen ? chosen.name || chosen.email : newAddress })), onError: fail },
    )
  }
  const busy = give.isPending || change.isPending
  const who = member ? member.name || member.email : chosen ? chosen.name || chosen.email : ''
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      hideClose
      eyebrow={<span className="k">{who ? t(mm.compEyebrow, { name: who }) : t(mm.compEyebrowNone)}</span>}
      title={member ? t(mm.compChangeTitle) : t(mm.compTitle)}
      footer={
        <>
          <Button onClick={onClose}>{t(mm.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={submit} disabled={!member && !chosen && !newAddress}>
            {member ? t(mm.compSave) : t(mm.compSubmit)}
          </Button>
        </>
      }
    >
      {member ? null : (
        <div className="lbl">
          {t(mm.compWho)}
          {chosen ? (
            <div className="row gap-2.5">
              <Avatar name={chosen.name || chosen.email} size={32} />
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="font-semibold text-ink">{chosen.name || chosen.email}</span>
                <span className="k">{chosen.lists?.length ? t(mm.compOnLists, { email: chosen.email, lists: orList(chosen.lists) }) : chosen.email}</span>
              </span>
              <Button size="sm" aria-label={t(mm.compChange)} onClick={() => setChosen(undefined)}>
                {t(mm.compChangeShort)}
              </Button>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              <input className="in" type="search" aria-label={t(mm.compWho)} placeholder={t(mm.compContactPlaceholder)} value={search} onChange={(e) => setSearch(e.target.value)} />
              {q.length >= 2 && found.data?.items.length ? (
                <ul className="ea-picks" aria-label={t(mm.compMatches)}>
                  {found.data.items.map((c) => (
                    <li key={c.id}>
                      <button type="button" onClick={() => setChosen({ id: c.id, name: c.name, email: c.email, lists: c.lists })}>
                        <Avatar name={c.name || c.email} size={24} />
                        <span className="flex min-w-0 flex-col items-start">
                          <span className="font-semibold">{c.name || c.email}</span>
                          <span className="k">{c.email}</span>
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              ) : null}
              {newAddress ? <span className="k">{t(mm.compNewAddress, { email: newAddress })}</span> : null}
              {error.contactId || error.email ? <span className="err">{error.contactId || error.email}</span> : null}
            </div>
          )}
          <span className="k font-medium">{t(mm.compWhoHint)}</span>
        </div>
      )}
      <div className="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
        <label className="lbl">
          {t(mm.compTier)}
          <select className="in" aria-label={t(mm.compTier)} value={tierId} onChange={(e) => setTier(e.target.value)}>
            {options.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
          {error.tierId ? <span className="err">{error.tierId}</span> : null}
        </label>
        <label className="lbl">
          {t(mm.compReason)}
          <select className="in" aria-label={t(mm.compReason)} value={reason} onChange={(e) => setReason(e.target.value as Reason)}>
            {reasons.map((r) => (
              <option key={r} value={r}>
                {reasonName(t, r)}
              </option>
            ))}
          </select>
        </label>
      </div>
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="lbl mb-1">{t(mm.compUntil)}</legend>
        <label className={`ea-radio${!dated ? ' on' : ''}`}>
          <input type="radio" name="comp-until" checked={!dated} onChange={() => setDated(false)} />
          <span className="flex min-w-0 flex-col gap-0.5 font-semibold">
            {t(mm.compNoEnd)}
            <span className="k">{t(mm.compNoEndHint)}</span>
          </span>
        </label>
        <label className={`ea-radio${dated ? ' on' : ''}`}>
          <input type="radio" name="comp-until" checked={dated} onChange={() => setDated(true)} />
          <span className="flex min-w-0 flex-1 flex-col gap-2 font-semibold">
            {t(mm.compOnDate)}
            <input className="in max-w-[180px] font-normal" type="date" aria-label={t(mm.compEndDate)} value={until} disabled={!dated} onChange={(e) => setUntil(e.target.value)} />
          </span>
        </label>
        {error.until ? <span className="err">{error.until}</span> : null}
      </fieldset>
      <label className="lbl">
        {t(mm.compNote)}
        <textarea className="in ea-ta" style={{ height: 64 }} maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />
        <span className="k font-medium">{t(mm.compNoteHint)}</span>
      </label>
      {member ? null : (
        <label className="flex items-start gap-2.5">
          <input type="checkbox" className="mt-0.5" checked={welcome && welcomeLive} disabled={!welcomeLive} onChange={(e) => setWelcome(e.target.checked)} />
          <span className="flex min-w-0 flex-col gap-0.5 font-semibold">
            {t(mm.compWelcome)}
            <span className="k font-medium">
              {welcomeEmail
                ? welcomeLive
                  ? t(mm.compWelcomeTemplate, { name: memberEmailName(t, 'welcome', tierShort), template: welcomeEmail.template, version: welcomeEmail.liveVersion })
                  : t(mm.compWelcomeDraft, { name: memberEmailName(t, 'welcome', tierShort) })
                : memberEmailName(t, 'welcome', tierShort)}
            </span>
          </span>
        </label>
      )}
      <Note tone="neutral">{t(mm.compLivesHere)}</Note>
    </Dialog>
  )
}

function reasonName(t: ReturnType<typeof useT>, r: Reason): string {
  return { team: t(mm.reasonTeamL), contributor: t(mm.reasonContributorL), press: t(mm.reasonPressL), partner: t(mm.reasonPartnerL), giveaway: t(mm.reasonGiveawayL), other: t(mm.reasonOtherL) }[r]
}
