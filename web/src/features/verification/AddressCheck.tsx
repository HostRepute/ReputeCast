import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, dayMonthDate, orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, money, Notice, ResultBadge, tagValue, useRefreshVerification, type AddressCheck } from './shared'

type Check = 'valid' | 'invalid' | 'risky' | 'unknown' | 'never'

/** Contacts' filter by the latest address check. */
export function AddressCheckFilter({ value, onChange, className }: { value?: Check; onChange: (v: Check | undefined) => void; className?: string }) {
  const t = useT()
  return (
    <select className={className} aria-label={t(m.addressCheck)} value={value ?? ''} onChange={(e) => onChange((e.target.value || undefined) as Check | undefined)}>
      <option value="">{t(m.anyAddressCheck)}</option>
      <option value="valid">{t(m.resultValid)}</option>
      <option value="invalid">{t(m.resultInvalid)}</option>
      <option value="risky">{t(m.resultRisky)}</option>
      <option value="unknown">{t(m.resultUnknown)}</option>
      <option value="never">{t(m.resultNever)}</option>
    </select>
  )
}

const reasonShort: Record<string, keyof typeof m> = { catch_all: 'reasonCatchAllDomain', role: 'reasonRoleAddress', disposable: 'reasonDisposableDomain', abuse: 'shortAbuse' }

/** A contact's Address check row: never checked with Verify now, or the latest result. */
export function AddressCheckRow({ contactId }: { contactId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${base}/contacts/{contact}`, { params: { path: { workspace: slug, contact: contactId } } })
  const [open, setOpen] = useState(false)
  const c = q.data
  if (!c) return null
  const canStart = c.connected && c.canVerify
  return (
    <div className="pkv">
      <span className="k">{t(m.addressCheck)}</span>
      <span className="flex flex-wrap items-center gap-2">
        {!c.checked ? (
          <>
            <span className="text-ink-2">{t(m.neverChecked)}</span>
            {canStart ? (
              <Button size="sm" onClick={() => setOpen(true)}>
                {t(m.verifyNowShort)}
              </Button>
            ) : null}
          </>
        ) : (
          <>
            <ResultBadge result={c.result} reason={c.result === 'risky' && c.reason ? t(m[reasonShort[c.reason] ?? 'shortAbuse']) : undefined} />
            <span className="k text-ink-2!">
              {c.result === 'valid' && c.serviceName && c.checkedAt ? (
                t(m.checkedWith, { name: c.serviceName, date: dayMonthDate(c.checkedAt) })
              ) : (
                <>
                  {c.serviceName && c.code ? t(m.checkedCode, { name: c.serviceName, code: c.code }) : null}
                  {c.suppressed && c.suppressedAt ? ` · ${t(m.suppressedOn, { date: dayMonthDate(c.suppressedAt), time: clock(c.suppressedAt) })}` : null}
                  {c.tag ? (
                    <>
                      {' · '}
                      <T msg={m.taggedWith} values={{ ...tagValue, code: c.tag }} />
                    </>
                  ) : null}
                </>
              )}
            </span>
          </>
        )}
      </span>
      {open ? <VerifyOneDialog c={c} contactId={contactId} onClose={() => setOpen(false)} /> : null}
    </div>
  )
}

/** Verify this address? One check, with the same notice as every use. */
function VerifyOneDialog({ c, contactId, onClose }: { c: AddressCheck; contactId: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const refresh = useRefreshVerification()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const name = c.verifyServiceName ?? c.serviceName ?? ''
  const go = async () => {
    setBusy(true)
    setError(undefined)
    const res = await http.POST(`${base}/contacts/{contact}/verify`, { params: { path: { workspace: slug, contact: contactId } }, body: {} })
    setBusy(false)
    if (!res.data) {
      setError(errorMessage(res.error))
      return
    }
    void refresh()
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/contacts/{contact}'] })
    onClose()
    const r = res.data.result
    toast({ kind: 'success', title: t(m.checkedToast, { email: c.email, result: t(r === 'valid' ? m.resultValid : r === 'invalid' ? m.resultInvalid : r === 'risky' ? m.resultRisky : m.resultUnknown) }) })
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !busy && onClose()}
      eyebrow={<span className="k">{t(m.fromProfile, { contact: c.email })}</span>}
      title={t(m.verifyThisTitle)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={() => void go()} loading={busy} busyLabel={t(m.starting)} disabledReason={busy || c.canVerify ? undefined : c.helpers.length ? t(m.cantVerifyPlain, { helpers: orList(c.helpers) }) : t(m.cantVerifyPlainNobody)}>
            {t(m.verifyNow)}
          </Button>
        </>
      }
    >
      <span className="text-[14px]">
        <T msg={m.verifyOne} values={{ email: c.email, name, price: money(c.price, true), mono: (x) => <span className="mono">{x}</span> }} />
      </span>
      <Notice head={c.regionLabel ? t(m.noticeOne, { email: c.email, name, region: c.regionLabel }) : t(m.noticeOneNoRegion, { email: c.email, name })} name={name} />
      {error ? (
        <div className="note b" role="alert">
          <Icon name="alertCircle" size={16} className="mt-px flex-none" />
          <span>{error}</span>
        </div>
      ) : null}
    </Dialog>
  )
}
