import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Checkbox, Field, Input, Select } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { EARN } from '../EarnPage'
import { money } from '../format'
import { giftWhen } from './parts'
import { mg } from './messages'

type View = Schemas['GiftsView']
type Option = Schemas['GiftOption']
type Row = Schemas['GiftRow']

const lengths = [1, 3, 6, 12, 24]

/** A gift option: a tier for a length at a one-time price (earn:edit). */
export function OptionDialog({ v, option, onClose }: { v: View; option?: Option; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const first = v.tiers[0]
  const [tierId, setTierId] = useState(option?.tierId ?? first?.id ?? '')
  const [months, setMonths] = useState(String(option?.months ?? 3))
  const tier = v.tiers.find((x) => x.id === tierId)
  const suggested = (m: number) => (tier?.monthlyCents ? (tier.monthlyCents * m) / 100 : 0)
  const [price, setPrice] = useState(option ? String(option.amountCents / 100) : suggested(3) ? String(suggested(3)) : '')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const create = $api.useMutation('post', `${EARN}/gifts/options`)
  const update = $api.useMutation('put', `${EARN}/gifts/options/{id}`)
  const remove = $api.useMutation('delete', `${EARN}/gifts/options/{id}`)
  const cents = Math.round(Number(price.replace(',', '.')) * 100) || 0
  const m = Number(months)
  const name = t(mg.oName, { tier: tier?.name ?? '', months: m })
  const done = (title: string) => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/gifts`] })
    toast({ kind: 'success', title, detail: t(mg.auditedNote) })
    onClose()
  }
  const onError = (e: unknown) => {
    if (e instanceof ApiError && e.problem.errors?.length) {
      const next: Record<string, string> = {}
      for (const x of e.problem.errors) next[(x.location ?? '').replace(/^body\./, '')] = x.message ?? ''
      setErrors(next)
      return
    }
    toast({ kind: 'error', title: option ? t(mg.oEdit) : t(mg.oNew), detail: errorMessage(e) })
  }
  const save = () => {
    setErrors({})
    const body = { tierId, months: m, amountCents: cents }
    if (option) update.mutate({ params: { path: { workspace: slug, id: option.id } }, body }, { onSuccess: () => done(t(mg.oSaved, { name })), onError })
    else create.mutate({ params: { path: { workspace: slug } }, body }, { onSuccess: () => done(t(mg.oAdded, { name })), onError })
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={option ? t(mg.oEdit) : t(mg.oNew)}
      footer={
        <>
          {option ? (
            <Button
              variant="danger"
              className="mr-auto"
              loading={remove.isPending}
              onClick={() =>
                remove.mutate({ params: { path: { workspace: slug, id: option.id } } }, { onSuccess: () => done(t(mg.oRemoved, { name })), onError })
              }
            >
              {t(mg.oRemove)}
            </Button>
          ) : null}
          <Button onClick={onClose}>{t(mg.cancel)}</Button>
          <Button variant="primary" loading={create.isPending || update.isPending} disabled={!tierId || cents <= 0} onClick={save}>
            {option ? t(mg.oSave) : t(mg.oCreate)}
          </Button>
        </>
      }
    >
      <Field label={t(mg.oTier)} error={errors.tierId}>
        <Select value={tierId} onChange={(e) => setTierId(e.target.value)}>
          {v.tiers.map((x) => (
            <option key={x.id} value={x.id}>
              {x.name}
            </option>
          ))}
        </Select>
      </Field>
      <Field label={t(mg.oLength)} error={errors.months}>
        <Select
          value={months}
          onChange={(e) => {
            const next = Number(e.target.value)
            if (!option && (price === '' || Number(price) === suggested(m))) setPrice(suggested(next) ? String(suggested(next)) : price)
            setMonths(e.target.value)
          }}
        >
          {lengths.map((x) => (
            <option key={x} value={x}>
              {t(mg.length, { months: x })}
            </option>
          ))}
        </Select>
      </Field>
      <Field
        label={t(mg.oPrice)}
        error={errors.amountCents}
        hint={cents > 0 ? t(mg.oPriceHint, { price: money(Math.round(cents / m), v.currency, 'auto') }) : tier?.monthlyCents ? t(mg.oPriceTier, { price: money(tier.monthlyCents, v.currency, 'auto') }) : undefined}
      >
        <Input inputMode="decimal" className="w-[140px]" value={price} onChange={(e) => setPrice(e.target.value)} />
      </Field>
      <span className="k">{t(mg.oAudit)}</span>
    </Dialog>
  )
}

/** "Change the delivery date" of a scheduled gift (earn:manage). */
export function DeliveryDialog({ row, onClose }: { row: Row; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const zones = [...new Set([row.timezone, 'America/Chicago', 'America/New_York', 'America/Los_Angeles', 'Europe/London', 'Europe/Berlin', 'Asia/Tokyo'].filter(Boolean))]
  const ymd = (at: string, tz: string) => {
    try {
      return new Intl.DateTimeFormat('en-CA', { timeZone: tz || undefined, year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(at))
    } catch {
      return ''
    }
  }
  const hm = (at: string, tz: string) => {
    try {
      return new Intl.DateTimeFormat('en-GB', { timeZone: tz || undefined, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(new Date(at))
    } catch {
      return '08:00'
    }
  }
  const [zone, setZone] = useState(row.timezone || zones[0] || '')
  const [date, setDate] = useState(row.deliverAt ? ymd(row.deliverAt, zone) : '')
  const [time, setTime] = useState(row.deliverAt ? hm(row.deliverAt, zone) : '08:00')
  const [notify, setNotify] = useState(false)
  const [error, setError] = useState('')
  const move = $api.useMutation('post', `${EARN}/gifts/{id}/delivery`)
  const buyer = row.buyer.split(' ')[0] ?? row.buyer
  const was = row.deliverAt ? giftWhen(row.deliverAt, row.timezone) : undefined
  const to = row.inPerson ? t(mg.inPerson) : row.recipientName || row.recipientEmail
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/gifts`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/gifts/list`] })
  }
  const moved = (r: Row) => (r.deliverAt ? giftWhen(r.deliverAt, r.timezone) : { date: '', time: '' })
  // Undo puts the date back after the dialog closed: a plain call, not this dialog's mutation.
  const back = row.deliverAt ? { date: ymd(row.deliverAt, row.timezone), time: hm(row.deliverAt, row.timezone), timezone: row.timezone } : undefined
  const restore = async () => {
    if (!back) return
    const { data, error: err } = await http.POST(`${EARN}/gifts/{id}/delivery`, { params: { path: { workspace: slug, id: row.id } }, body: back })
    refresh()
    if (data) toast({ kind: 'success', title: t(mg.moved, moved(data)), detail: t(mg.auditedNote) })
    else toast({ kind: 'error', title: t(mg.dTitle), detail: errorMessage(err) })
  }
  const send = (body: Schemas['GiftDeliveryInput']) =>
    move.mutate(
      { params: { path: { workspace: slug, id: row.id } }, body },
      {
        onSuccess: (r) => {
          refresh()
          toast({ kind: 'success', title: t(mg.moved, moved(r)), detail: t(mg.auditedNote), undo: back ? () => void restore() : undefined })
          onClose()
        },
        onError: (e) => setError(e instanceof ApiError && e.problem.errors?.length ? (e.problem.errors[0]?.message ?? errorMessage(e)) : errorMessage(e)),
      },
    )
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mg.dEyebrow, { to, gift: t(mg.giftLine, { tier: row.short, months: row.months }), amount: money(row.amountCents, row.currency, 'always') })}</span>}
      title={t(mg.dTitle)}
      description={
        was
          ? t(row.inPerson ? mg.dTextPerson : mg.dText, { email: row.inPerson ? row.buyerEmail : row.recipientEmail, buyer: row.buyer, when: `${was.date}, ${was.time}` })
          : undefined
      }
      footer={
        <>
          <span className="k mr-auto">{t(mg.dAudit)}</span>
          <Button onClick={onClose}>{t(mg.cancel)}</Button>
          <Button variant="primary" loading={move.isPending} disabled={!date || !time} onClick={() => send({ date, time, timezone: zone, notifyBuyer: notify })}>
            {t(mg.dSave)}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(150px,100%),1fr))] gap-3">
        <Field label={t(mg.dDate)}>
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label={t(mg.dTime)}>
          <Input type="time" value={time} onChange={(e) => setTime(e.target.value)} />
        </Field>
        <Field label={t(mg.dZone)} hint={zone === row.timezone ? t(mg.dZoneHint, { name: buyer }) : undefined}>
          <Select value={zone} onChange={(e) => setZone(e.target.value)}>
            {zones.map((z) => (
              <option key={z} value={z}>
                {z}
              </option>
            ))}
          </Select>
        </Field>
      </div>
      <Checkbox label={t(mg.dNotify, { name: row.buyer })} checked={notify} onChange={(e) => setNotify(e.target.checked)} />
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
    </Dialog>
  )
}
