import { useState } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { storesBase, useRefreshStores, type StorePage } from './api'
import { cc, cm } from './messages'
import './commerce.css'

/**
 * "Disconnect Acme Merch?": what pauses, stops and stays, then keep or
 * delete the store's products and orders (deleting asks for the store's
 * name). Recorded in the audit log.
 */
export function DisconnectStoreDialog({ page: p, onOpenChange }: { page: StorePage; onOpenChange: (open: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshStores()
  const [del, setDel] = useState(false)
  const [confirm, setConfirm] = useState('')
  const [err, setErr] = useState<string>()
  const m = $api.useMutation('post', `${storesBase}/{id}/disconnect`)
  const name = p.store.name
  const shopify = p.store.platform === 'shopify'
  const live = p.flows.filter((f) => f.status === 'live')
  const hooks = p.webhooks.filter((w) => w.status === 'registered').length || p.webhooks.length
  const ready = !del || confirm.trim() === name
  const submit = () => {
    setErr(undefined)
    m.mutate(
      { params: { path: { workspace: slug, id: p.store.id } }, body: { deleteData: del, confirm: del ? confirm.trim() : undefined } },
      {
        onSuccess: () => {
          toast({ kind: del ? 'deleted' : 'success', title: t(del ? cm.deleted : cm.disconnectedToast, { name }) })
          void refresh()
          onOpenChange(false)
        },
        onError: (e) => setErr(errorMessage(e)),
      },
    )
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(cc.dcTitle, { name })}
      eyebrow={<span className="k">{shopify ? t(cc.eyebrowShopify) : t(cc.eyebrowWoo)}</span>}
      size="md"
      className="max-w-[720px]"
      hideClose
      dismissible={false}
      footer={
        <>
          <span className="k mr-auto">{t(cc.dcAudit)}</span>
          <Button onClick={() => onOpenChange(false)}>{t(cc.cancel)}</Button>
          <Button variant="danger-primary" onClick={submit} disabled={!ready} loading={m.isPending}>
            {t(cc.dcButton, { name })}
          </Button>
        </>
      }
    >
      <span className="text-ink-nav">{t(cc.dcIntro, { count: hooks })}</span>
      <dl className="kv cm-kv">
        {live.length ? (
          <>
            <dt>{t(cc.dcPauses)}</dt>
            <dd>{t(cc.dcPausesValue, { names: formatList(live.map((f) => t(cc.dcFlow, { name: f.name }))), count: live.length })}</dd>
          </>
        ) : null}
        <dt>{t(cc.dcStops)}</dt>
        <dd>{t(cc.dcStopsValue)}</dd>
        <dt>{t(cc.dcStays)}</dt>
        <dd>{p.consent.listName ? t(cc.dcStaysValue, { count: count(p.consent.listMembers), list: p.consent.listName }) : t(cc.dcStaysNoList)}</dd>
      </dl>
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="lbl mb-1.5">{t(cc.dcData)}</legend>
        <div className={`cm-choice${del ? '' : ' on'}`}>
          <input id="cm-keep" type="radio" name="cm-store-data" checked={!del} onChange={() => setDel(false)} aria-describedby="cm-keep-sub" />
          <span className="flex flex-col gap-0.5">
            <label htmlFor="cm-keep" className="cursor-pointer font-semibold">
              {t(cc.dcKeep)}
            </label>
            <span className="k" id="cm-keep-sub">
              {t(cc.dcKeepSub)}
            </span>
          </span>
        </div>
        <div className={`cm-choice${del ? ' on' : ''}`}>
          <input id="cm-delete" type="radio" name="cm-store-data" checked={del} onChange={() => setDel(true)} aria-describedby="cm-delete-sub" />
          <span className="flex min-w-0 flex-1 flex-col gap-1.5">
            <label htmlFor="cm-delete" className="cursor-pointer font-semibold text-[#991B1B]">
              {t(cc.dcDelete)}
            </label>
            <span className="k" id="cm-delete-sub">
              {t(cc.dcDeleteSub, { products: p.store.products, orders: count(p.sync.ordersImported) })}
            </span>
            <input
              className="in"
              type="text"
              value={confirm}
              placeholder={name}
              aria-label={t(cc.dcConfirm, { name })}
              aria-invalid={err ? true : undefined}
              onChange={(e) => setConfirm(e.target.value)}
              onFocus={() => setDel(true)}
              autoComplete="off"
              spellCheck={false}
            />
            {err ? (
              <span className="k font-medium" role="alert">
                <span className="font-semibold text-bad-fg">{err}</span>
              </span>
            ) : null}
          </span>
        </div>
      </fieldset>
    </Dialog>
  )
}
