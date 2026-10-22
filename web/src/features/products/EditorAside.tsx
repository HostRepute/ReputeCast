import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Bar } from '@/components/ui/Data'
import { T, useI18n, useT, useTIn } from '@/i18n'
import { count } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import type { ProductDetail } from './api'
import { pe } from './editor.messages'
import { MINIMUM_CENTS, bareUrl, day, fileSize, isPriced, leadText, money, priceCard, priceShort, seeNoun, shopDescription, shopFiles } from './format'
import { pm } from './messages'
import { FileIcon, bold } from './parts'

type T = ReturnType<typeof useT>
type Price = Schemas['ProductPrice']

/** What the previews show: the form as it stands, with the last valid price. */
export interface Preview {
  name: string
  slug: string
  shortDescription: string
  longDescription: string
  coverUrl: string
  coverAlt: string
  price: Price
  salesPageId: string
  showInShop: boolean
  offerOn: boolean
  offerText: string
}

/**
 * The site's name, mark and navigation for the default product page's header (Website › Home & navigation,
 * and the logo or initial of Forms & public pages › Branding).
 */
export function useSiteFrame(): { title: string; nav: string[]; initial: string; logo?: string } {
  const { slug, me } = useWorkspace()
  const can = useCan('website:view')
  const canBrand = useCan('subscribers:view')
  const path = { params: { path: { workspace: slug } } }
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/website/home-nav', path, { enabled: can, staleTime: 5 * 60_000 })
  const b = $api.useQuery('get', '/api/v1/workspaces/{workspace}/public-pages', path, { enabled: canBrand, staleTime: 5 * 60_000 })
  const title = q.data?.identity.title || me.workspace.name
  const brand = b.data?.branding
  return {
    title,
    nav: (q.data?.header ?? []).map((h) => h.name),
    // Without the branding, the workspace's initial stands in for the publisher's.
    initial: brand?.initial || (me.workspace.name.trim() || title.trim()).charAt(0).toUpperCase(),
    logo: brand?.logo?.url || undefined,
  }
}

/** The product card as the shop shows it (ProductEditor: In the shop). */
export function ShopCard({ detail, p }: { detail: ProductDetail; p: Preview }) {
  const t = useT()
  const tEn = useTIn('en')
  const { locale } = useI18n()
  const cur = detail.payments.currency || 'usd'
  const summary = { count: detail.files.length, kind: detail.files[0]?.kindLabel ?? '', pages: detail.files.length === 1 ? (detail.files[0]?.pages ?? 0) : 0, bytes: detail.filesBytes }
  const files = shopFiles(t, summary)
  // The button names an English noun of the name ("See the guide"): in other languages it reads "See it".
  const noun = locale.startsWith('en') ? seeNoun(p.name) : undefined
  const price = !isPriced(p.price) ? t(pm.priceNone) : p.price.kind === 'pwyw' ? t(pe.shopPwyw, { min: money(p.price.minimumCents, cur) }) : money(p.price.amountCents, cur)
  return (
    <section className="card" aria-labelledby="pr-shop-h">
      <CardSection id="pr-shop-h" title={t(pe.inShop)} caption={t(pe.inShopCaption, { url: bareUrl(detail.shopUrl) })} />
      <div className="pad flex flex-col gap-2.5">
        <div className="pr-st">
          <article className="pr-shopc">
            <div className="s-art">
              <PreviewCover p={p} size="card" />
            </div>
            <div className="s-b">
              <p className="s-nm">{p.name}</p>
              <p className="s-ds">{shopDescription(p.shortDescription, files, shopFiles(tEn, summary))}</p>
              <div className="s-f">
                <span className="s-pr">{price}</span>
                <span className="s-btn1 s-full">{noun ? t(pe.shopSeeThe, { noun }) : t(pe.shopSeeIt)}</span>
              </div>
            </div>
          </article>
        </div>
        {p.showInShop ? null : <span className="k">{t(pe.hiddenFromShop)}</span>}
      </div>
    </section>
  )
}

function PreviewCover({ p, size }: { p: Preview; size: 'card' | 'lg' }) {
  const t = useT()
  if (p.coverUrl) return <img className={`s-cv s-${size}`} src={p.coverUrl} alt={p.coverAlt || t(pm.coverOf, { name: p.name })} />
  return (
    <span className={`s-cv s-${size} s-blank`} role="img" aria-label={t(pe.noCover)}>
      <FileIcon size={size === 'card' ? 40 : 46} />
    </span>
  )
}

/** The default product page, small (ProductEditor: Default product page). */
export function DefaultPage({ detail, p }: { detail: ProductDetail; p: Preview }) {
  const t = useT()
  const site = useSiteFrame()
  const cur = detail.payments.currency || 'usd'
  const page = detail.pages.find((x) => x.id === p.salesPageId)
  const url = `${bareUrl(detail.shopUrl)}/${p.slug}`
  const lead = leadText(p.longDescription) || p.shortDescription.trim()
  return (
    <section className="card" aria-labelledby="pr-page-h">
      <CardSection id="pr-page-h" title={t(pe.defaultPage)} />
      <div className="pad flex flex-col gap-2.5">
        <div className="pr-pagebox" role="img" aria-label={t(pe.defaultPageLabel)}>
          <div className="s-pg pr-st">
            <div className="s-hd">
              <div className="s-hdi">
                <span className="s-brand">
                  {site.logo ? (
                    <span className="s-mark s-logo">
                      <img src={site.logo} alt="" width={36} height={36} />
                    </span>
                  ) : (
                    <span className="s-mark">{site.initial}</span>
                  )}
                  {site.title}
                </span>
                <span className="s-nav">
                  {site.nav.slice(0, 6).map((n, i) => (
                    <span key={`${n}-${i}`}>{n}</span>
                  ))}
                </span>
                <span className="s-act">
                  <span className="s-btn2">{t(pe.siteSignIn)}</span>
                  <span className="s-btn1">{t(pe.siteSubscribe)}</span>
                </span>
              </div>
            </div>
            <div className="s-lav">
              <div className="s-spin">
                <div className="s-hero">
                  <div className="s-art2">
                    <PreviewCover p={p} size="lg" />
                  </div>
                  <div className="s-copy">
                    <span className="s-h1">{p.name}</span>
                    {lead ? <p className="s-lead">{lead}</p> : null}
                    <BuyBox p={p} currency={cur} connected={detail.payments.connected} />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <span className="k pr-ink">{page ? t(pe.defaultNotInUse, { path: page.path, url }) : t(pe.defaultInUse, { url })}</span>
      </div>
    </section>
  )
}

/** The Buy box, as the public pages draw it (internal/publicpages buyBox). */
function BuyBox({ p, currency, connected }: { p: Preview; currency: string; connected: boolean }) {
  const t = useT()
  const pwyw = p.price.kind === 'pwyw'
  const button = pwyw ? `${money(p.price.minimumCents, currency)}+` : money(p.price.amountCents, currency)
  if (!isPriced(p.price)) {
    // A draft without a price yet: nothing to buy.
    return (
      <div className="s-buybox">
        <span className="s-btn1 s-lg s-full s-off">{t(pm.priceNone)}</span>
      </div>
    )
  }
  return (
    <div className="s-buybox">
      {pwyw ? (
        <>
          <div className="s-bb-price">
            <span className="s-amt">{t(pe.pwyw)}</span>
          </div>
          <p className="s-bb-sub">
            {p.price.suggestedCents > 0
              ? t(pe.pwywFromSuggested, { min: money(p.price.minimumCents, currency), suggested: money(p.price.suggestedCents, currency) })
              : t(pe.pwywFrom, { min: money(p.price.minimumCents, currency) })}
          </p>
        </>
      ) : (
        <div className="s-bb-price">
          <span className="s-amt">{money(p.price.amountCents, currency)}</span>
          <span className="s-mut">{t(pe.oneTimePayment)}</span>
        </div>
      )}
      <span className={connected ? 's-btn1 s-lg s-full' : 's-btn1 s-lg s-full s-off'}>{connected ? t(pe.buyFor, { price: button }) : t(pe.buyUnavailable)}</span>
      {p.offerOn && p.offerText.trim() ? (
        <span className="s-offer">
          <span className="s-box" />
          <span>
            <span className="s-ot">{p.offerText}</span>
            <span className="s-os">{t(pe.offerNote)}</span>
          </span>
        </span>
      ) : null}
    </div>
  )
}

export interface CheckRow {
  key: string
  ok: boolean
  title: string
  detail?: string
  fix?: 'stripe' | 'template' | 'route' | 'pages'
  template?: string
  templateId?: string
}

/**
 * The publish checks as the card lists them: the route folds into the
 * template's line while both pass ("product-delivery v1 · the
 * Transactional route exists"), and failing checks come first.
 */
export function checkRows(t: T, detail: ProductDetail): CheckRow[] {
  const cur = detail.payments.currency || 'usd'
  const by = Object.fromEntries(detail.checks.map((c) => [c.key, c]))
  const routeOK = by.route?.ok ?? true
  const rows: CheckRow[] = []
  for (const c of detail.checks) {
    switch (c.key) {
      case 'stripe':
        rows.push(
          c.ok
            ? { key: c.key, ok: true, title: t(pe.ckStripe), detail: c.account ? t(pe.ckStripeDetail, { account: c.account, mode: c.mode ?? 'live' }) : undefined }
            : { key: c.key, ok: false, title: t(pe.ckStripeBad), detail: t(pe.ckStripeBadDetail), fix: 'stripe' },
        )
        break
      case 'file':
        rows.push(
          c.ok
            ? { key: c.key, ok: true, title: t(pe.ckFile), detail: c.file ? t(pe.ckFileDetail, { file: c.file, size: fileSize(t, c.bytes ?? 0) }) : undefined }
            : { key: c.key, ok: false, title: t(pe.ckFileBad), detail: t(pe.ckFileBadDetail) },
        )
        break
      case 'price':
        rows.push(
          c.ok
            ? { key: c.key, ok: true, title: t(pe.ckPrice), detail: priceCheck(t, detail.draft.price, cur) }
            : { key: c.key, ok: false, title: t(pe.ckPriceBad), detail: t(pe.ckPriceBadDetail, { min: money(MINIMUM_CENTS, cur) }) },
        )
        break
      case 'template':
        if (c.ok) {
          rows.push({
            key: c.key,
            ok: true,
            title: t(pe.ckTemplate),
            detail: c.template ? t(routeOK ? pe.ckTemplateDetailRoute : pe.ckTemplateDetail, { template: c.template, version: c.version ?? 1 }) : undefined,
          })
        } else if (c.template) {
          rows.push({
            key: c.key,
            ok: false,
            title: t(pe.ckTemplateDraft, { template: c.template }),
            detail: t(pe.ckTemplateDraftDetail),
            fix: 'template',
            template: c.template,
            templateId: c.templateId,
          })
        } else {
          rows.push({ key: c.key, ok: false, title: t(pe.ckTemplateNone), detail: t(pe.ckTemplateNoneDetail) })
        }
        break
      case 'route':
        if (!c.ok) rows.push({ key: c.key, ok: false, title: t(pe.ckRouteBad), detail: t(pe.ckRouteBadDetail), fix: 'route' })
        else if (!by.template?.ok) rows.push({ key: c.key, ok: true, title: t(pe.ckRoute), detail: c.route })
        break
      case 'cover_alt':
        rows.push(
          c.ok
            ? { key: c.key, ok: true, title: t(pe.ckAlt), detail: detail.draft.coverUrl ? undefined : t(pe.ckAltNoCover) }
            : { key: c.key, ok: false, title: t(pe.ckAltBad), detail: t(pe.ckAltBadDetail) },
        )
        break
      case 'sales_page':
        rows.push(
          c.ok
            ? { key: c.key, ok: true, title: t(pe.ckPage), detail: c.default ? t(pe.ckPageDefault) : c.path ? (c.at ? t(pe.ckPageDetail, { path: c.path, date: day(c.at) }) : c.path) : undefined }
            : { key: c.key, ok: false, title: t(pe.ckPageBad), detail: t(pe.ckPageBadDetail, { path: c.path ?? '' }), fix: 'pages' },
        )
        break
    }
  }
  return [...rows.filter((r) => !r.ok), ...rows.filter((r) => r.ok)]
}

/** "$29, one-time", "Pay what you want, from $5". */
function priceCheck(t: T, p: Price, currency: string): string {
  return p.kind === 'pwyw' ? priceCard(t, p, currency) : priceShort(t, p, currency)
}

/** The publish checks (ProductEditor, ProductEditorStates). */
export function ChecksCard({ detail, rows }: { detail: ProductDetail; rows: CheckRow[] }) {
  const t = useT()
  const canPublish = useCan('products:publish')
  const passed = rows.filter((r) => r.ok).length
  const caption =
    passed === rows.length ? (canPublish || detail.status !== 'draft' ? t(pe.checksAllPass, { count: rows.length }) : t(pe.checksAllPassNoRight, { count: rows.length })) : t(pe.checksSome, { passed, count: rows.length })
  return (
    <section className="card" aria-labelledby="pr-checks-h">
      <CardSection id="pr-checks-h" title={t(pe.checks)} caption={caption} />
      <ul className="pad m-0 list-none pt-1.5 pb-2">
        {rows.map((r) => (
          <li key={r.key} className={r.ok ? 'pr-ckl is-ok' : 'pr-ckl is-bad'}>
            {r.ok ? <Icon name="check" size={14} strokeWidth={2.2} /> : <Icon name="alertCircle" size={15} />}
            <span className="tx">
              <b>{r.title}</b>
              {r.detail ? <span className="k">{r.detail}</span> : null}
              {r.ok ? null : <Fix row={r} />}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}

/** A failing check's fix: where to go and what to do there. */
export function Fix({ row }: { row: CheckRow }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canPayments = useCan('payments:manage')
  const params = { workspace: slug }
  let fix: ReactNode = null
  if (row.fix === 'stripe')
    fix = (
      <>
        {canPayments ? (
          <ButtonLink to="/w/$workspace/earn/settings" params={params} search={{ tab: 'payments' }} size="sm" variant="primary">
            {t(pm.connectStripe)}
          </ButtonLink>
        ) : null}
        <span className="k">{t(pm.paymentsPath)}</span>
      </>
    )
  if (row.fix === 'template' && row.templateId)
    fix = (
      <ButtonLink to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: row.templateId }} size="sm" variant="primary">
        {t(pe.ckTemplateFix, { template: row.template ?? '' })}
      </ButtonLink>
    )
  if (row.fix === 'route')
    fix = (
      <>
        <ButtonLink to="/w/$workspace/providers/routing" params={params} size="sm" variant="primary">
          {t(pe.ckRouteFix)}
        </ButtonLink>
        <span className="k">{t(pe.ckRoutePath)}</span>
      </>
    )
  if (row.fix === 'pages')
    fix = (
      <ButtonLink to="/w/$workspace/forms/pages" params={params} size="sm">
        {t(pe.ckPagesFix)}
      </ButtonLink>
    )
  return fix ? <span className="fix">{fix}</span> : null
}

/** "Since 1 Oct": a published product's numbers. */
export function NumbersCard({ detail }: { detail: ProductDetail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const n = detail.numbers
  if (!n) return null
  const cur = detail.payments.currency || 'usd'
  const sources = [
    { label: t(pe.fromPage), value: n.fromPage, show: true },
    { label: t(pe.fromShop), value: n.fromShop, show: true },
    { label: t(pe.fromDefault), value: n.fromDefault, show: n.fromDefault > 0 },
  ].filter((s) => s.show)
  const max = Math.max(1, ...sources.map((s) => s.value))
  const tile = (label: string, value: string, sub?: string) => (
    <div className="card pr-stat">
      <span className="k">{label}</span>
      <span className="v">{value}</span>
      {sub ? <span className="s">{sub}</span> : null}
    </div>
  )
  return (
    <section className="card" aria-labelledby="pr-numbers-h">
      <CardSection
        id="pr-numbers-h"
        title={t(pe.numbersSince, { date: day(n.since) })}
        actions={
          <Link to="/w/$workspace/earn/products" params={{ workspace: slug }} search={{ product: detail.id }} className="pr-lnk">
            {t(pe.ordersLink)}
          </Link>
        }
      />
      <div className="pad flex flex-col gap-3.5">
        <div className="pr-nums">
          {tile(t(pe.numbersOrders), count(n.orders), n.test ? t(pe.numbersTest) : undefined)}
          {tile(t(pe.numbersGross), money(n.grossCents, cur, 'never'), n.test ? t(pe.numbersTest) : undefined)}
          {tile(t(pe.numbersRefunded), money(n.refundedCents, cur, 'never'), n.refunds ? t(pe.numbersRefundedSub, { count: n.refunds }) : undefined)}
        </div>
        <div className="pr-bars">
          {sources.map((s) => (
            <div key={s.label} className="pr-srcrow">
              <div className="lt">
                <span>{s.label}</span>
                <span className="pr-num font-semibold">{count(s.value)}</span>
              </div>
              <Bar value={(s.value / max) * 100} />
            </div>
          ))}
        </div>
        <div className="pr-lines">
          <span>
            <T
              msg={n.offerList ? pe.buyersLine : pe.buyersLineNoList}
              values={{ b: bold, contacts: count(n.contacts), joined: count(n.joined), none: count(n.none), list: n.offerList }}
            />
          </span>
          {n.automation ? (
            <span>
              <T
                msg={pe.automationLine}
                values={{ b: bold, entered: count(n.automation.entered), name: n.automation.name, contacts: n.automation.contacts, refunded: count(n.automation.refunded) }}
              />
            </span>
          ) : null}
        </div>
      </div>
    </section>
  )
}
