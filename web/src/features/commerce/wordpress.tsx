import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { ConnectStoreDialog } from './ConnectStoreDialog'
import { cw } from './elsewhere.messages'
import './commerce.css'

type Woo = Schemas['IntegrationSiteWoo']

/*
 * Settings › Integrations › WordPress: a site running WooCommerce shows
 * "WooCommerce detected", the plugin's checkout consent and abandoned
 * checkout reporting, linked to the store's page (design expansion phase 7).
 */

function Line({ ok, children }: { ok: boolean; children: string }) {
  return (
    <li>
      {ok ? (
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#15803D" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="m3 8 3.5 3.5L13 5" />
        </svg>
      ) : (
        <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M8 3v6M8 12v.5" />
        </svg>
      )}
      <span>{children}</span>
    </li>
  )
}

/** The badge in the site's header. */
export function WooBadge({ woo }: { woo?: Woo }) {
  const t = useT()
  if (!woo) return null
  return <Badge tone="purple">{t(cw.detected)}</Badge>
}

/** The site's WooCommerce section. */
export function WooSection({ woo, canManage }: { woo?: Woo; canManage: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [connecting, setConnecting] = useState(false)
  if (!woo) return null
  return (
    <div className="ih-sect">
      <h3>{t(cw.detected)}</h3>
      <span className="k">{t(cw.version, { version: woo.version })}</span>
      <ul className="cm-wlist">
        <Line ok={woo.consent}>{woo.consent ? t(cw.consentOn) : t(cw.consentOff)}</Line>
        <Line ok={woo.checkouts}>{woo.checkouts ? t(cw.checkoutsOn) : t(cw.checkoutsOff)}</Line>
      </ul>
      {woo.storeId ? (
        <div className="row gap-2.5">
          <ButtonLink size="sm" to="/w/$workspace/settings/integrations/stores/$storeId" params={{ workspace: slug, storeId: woo.storeId }}>
            {t(cw.openStore, { name: woo.storeName ?? '' })}
          </ButtonLink>
          {woo.storeStatus === 'disconnected' ? <span className="k">{t(cw.storeOff, { name: woo.storeName ?? '' })}</span> : null}
        </div>
      ) : (
        <>
          <span className="k">{t(cw.notConnected)}</span>
          {canManage ? (
            <Button size="sm" className="self-start" onClick={() => setConnecting(true)}>
              {t(cw.connect)}
            </Button>
          ) : null}
        </>
      )}
      {connecting ? <ConnectStoreDialog platform="woocommerce" onOpenChange={setConnecting} /> : null}
    </div>
  )
}
