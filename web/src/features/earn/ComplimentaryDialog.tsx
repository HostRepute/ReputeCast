import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Checkbox, Field, Input, OptionCard, Select, Textarea } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { EARN } from './EarnPage'
import { memberEmailName } from './MemberPanel'
import { mm } from './members.messages'

type Reason = Schemas['EarnComplimentaryInput']['reason']
const reasons: Reason[] = ['team', 'contributor', 'press', 'partner', 'giveaway', 'other']

/**
 * Give complimentary access (a contact or a new address, a tier, an end
 * date, a reason) or, with `member`, change it.
 */
export function ComplimentaryDialog({ tiers, member, onClose }: { tiers?: Schemas['EarnListRef'][]; member?: Schemas['EarnMember']; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const plans = $api.useQuery('get', `${EARN}/plans`, { params: { path: { workspace: slug } } }, { enabled: !tiers })
  const options = tiers ?? plans.data?.tiers.map((x) => ({ id: x.id, name: x.name })) ?? []
  const [who, setWho] = useState<'contact' | 'new'>('contact')
  const [search, setSearch] = useState('')
  const [contact, setContact] = useState<{ id: string; email: string }>()
  const [email, setEmail] = useState('')
  const [tier, setTier] = useState(member ? '' : (options[0]?.id ?? ''))
  const [until, setUntil] = useState(member?.grantUntil ? member.grantUntil.slice(0, 10) : '')
  const [dated, setDated] = useState(Boolean(member?.grantUntil))
  const [reason, setReason] = useState<Reason>((member?.source.reason as Reason) || 'contributor')
  const [note, setNote] = useState(member?.grantNote ?? '')
  const [welcome, setWelcome] = useState(false)
  const [error, setError] = useState<Record<string, string>>({})
  const found = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts', { params: { path: { workspace: slug }, query: { q: search, pageSize: 5 } } }, { enabled: !member && who === 'contact' && search.length >= 2 })
  const give = $api.useMutation('post', `${EARN}/members/complimentary`)
  const change = $api.useMutation('put', `${EARN}/members/{id}/complimentary`)
  const tierId = tier || (member ? (options.find((o) => o.name === member.tierName)?.id ?? options[0]?.id ?? '') : '')
  const done = (title: string) => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members/{id}`] })
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
      { params: { path: { workspace: slug } }, body: { contactId: who === 'contact' ? contact?.id : undefined, email: who === 'new' ? email : undefined, tierId, until: untilValue, reason, note, sendWelcome: welcome } },
      { onSuccess: () => done(t(mm.compGiven, { who: who === 'new' ? email : (contact?.email ?? '') })), onError: fail },
    )
  }
  const busy = give.isPending || change.isPending
  const tierName = options.find((o) => o.id === tierId)?.name ?? ''
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={member ? t(mm.compChangeTitle) : t(mm.compTitle)}
      footer={
        <>
          <Button onClick={onClose}>{t(mm.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={submit} disabled={!member && (who === 'contact' ? !contact : !email)}>
            {member ? t(mm.compSave) : t(mm.compSubmit)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {member ? null : (
          <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
            <legend className="mb-1 text-[13px] font-semibold">{t(mm.compWho)}</legend>
            <OptionCard name="who" value="contact" checked={who === 'contact'} onChange={() => setWho('contact')} title={t(mm.compExisting)}>
              {who === 'contact' ? (
                <div className="mt-2 flex flex-col gap-2">
                  <Input aria-label={t(mm.compContact)} placeholder={t(mm.compContactPlaceholder)} value={contact ? contact.email : search} onChange={(e) => {
                    setContact(undefined)
                    setSearch(e.target.value)
                  }} />
                  {!contact && found.data?.items.length ? (
                    <ul className="m-0 flex list-none flex-col gap-1 p-0">
                      {found.data.items.map((c) => (
                        <li key={c.id}>
                          <button type="button" className="btn sm" onClick={() => setContact({ id: c.id, email: c.email })}>
                            {c.email}
                          </button>
                        </li>
                      ))}
                    </ul>
                  ) : null}
                  {error.contactId ? <span className="err">{error.contactId}</span> : null}
                </div>
              ) : null}
            </OptionCard>
            <OptionCard name="who" value="new" checked={who === 'new'} onChange={() => setWho('new')} title={t(mm.compNew)} description={t(mm.compNewHint)}>
              {who === 'new' ? (
                <div className="mt-2">
                  <Field label={t(mm.compEmail)} error={error.email}>
                    <Input type="email" value={email} maxLength={320} onChange={(e) => setEmail(e.target.value)} />
                  </Field>
                </div>
              ) : null}
            </OptionCard>
          </fieldset>
        )}
        <Field label={t(mm.compTier)} error={error.tierId}>
          <Select value={tierId} onChange={(e) => setTier(e.target.value)}>
            {options.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </Select>
        </Field>
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="mb-1 text-[13px] font-semibold">{t(mm.compUntil)}</legend>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="radio" name="until" checked={!dated} onChange={() => setDated(false)} />
            {t(mm.compNoEnd)}
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="radio" name="until" checked={dated} onChange={() => setDated(true)} />
            {t(mm.compOnDate)}
            <Input type="date" aria-label={t(mm.compOnDate)} className="max-w-[180px]" value={until} disabled={!dated} onChange={(e) => setUntil(e.target.value)} />
          </label>
          {error.until ? <span className="err">{error.until}</span> : null}
        </fieldset>
        <Field label={t(mm.compReason)} error={error.reason}>
          <Select value={reason} onChange={(e) => setReason(e.target.value as Reason)}>
            {reasons.map((r) => (
              <option key={r} value={r}>
                {reasonName(t, r)}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t(mm.compNote)} hint={t(mm.compNoteHint)} optional>
          <Textarea rows={2} maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
        {member ? null : <Checkbox label={t(mm.compWelcome, { email: memberEmailName(t, 'welcome', tierName) })} hint={t(mm.compWelcomeHint)} checked={welcome} onChange={(e) => setWelcome(e.target.checked)} />}
      </div>
    </Dialog>
  )
}

function reasonName(t: ReturnType<typeof useT>, r: Reason): string {
  return { team: t(mm.reasonTeamL), contributor: t(mm.reasonContributorL), press: t(mm.reasonPressL), partner: t(mm.reasonPartnerL), giveaway: t(mm.reasonGiveawayL), other: t(mm.reasonOtherL) }[r]
}
