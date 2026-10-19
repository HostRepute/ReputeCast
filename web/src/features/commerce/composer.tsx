import { useQuery } from '@tanstack/react-query'
import type { Component } from 'grapesjs'
import { useEffect, useId, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { useComposer } from '@/features/composer/context'
import { markerOf, tagOf } from '@/features/composer/editor/labels'
import { AttentionIcon } from '@/features/integrations/shared'
import { T, defineMessages, useT } from '@/i18n'
import { clock, count, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { commerceBase, money, storesBase } from './api'
import './commerce.css'

type Block = Schemas['CommerceBlock']
type Render = Schemas['CommerceBlockRender']
type Product = Schemas['CommerceProduct']
type CatalogStore = Schemas['CommerceCatalogStore']

/**
 * Store blocks in the composer (design expansion phase 7, CommerceEmails):
 * the product picker, the Product grid and its settings in the Style tab,
 * the Product card filled from the store's catalog, a product removed from
 * the store, and the Checkout items block of automation emails.
 *
 * A store block is a section with its settings in `rc-store` and its
 * drawing in a raw block between `<!--rc:store-items {…}-->` markers;
 * sends draw it again with the store's current prices.
 */
export const ce = defineMessages('commerce.composer', {
  blockGrid: 'Product grid',
  blockCard: 'Product card',
  pickerEyebrow: 'Composer · product grid',
  pickerEyebrowCard: 'Composer · product card',
  pickerTitle: 'Insert products',
  pickerTitleCard: 'Insert a product',
  source: 'Source',
  sourceStore: '{name} · {count}',
  search: 'Search {name}',
  collection: 'Collection',
  all: 'All',
  inStock: 'In stock',
  soldOut: 'Sold out',
  left: '{count} left',
  removed: 'Removed',
  pickerNote: 'Prices and availability are read when each email is sent. A sold-out product is hidden or labelled, as the block says.',
  selected: '{count} selected',
  insert: '{count, plural, one {Insert # product} other {Insert # products}}',
  insertOne: 'Insert the product',
  cancel: 'Cancel',
  noProducts: 'No product matches.',
  more: 'Show more',
  gridFrom: 'From {store}',
  products: 'Products',
  chosen: 'Chosen products',
  newest: 'Newest from a collection',
  collectionOption: '{title} · {count}',
  howMany: 'How many',
  soldOutLabel: 'Sold-out products',
  hide: 'Hide them',
  label: 'Label “Sold out”',
  compareAt: 'Show the compare-at price',
  compareAtSub: 'When a product is on sale',
  button: 'Button text',
  links: "Links get <mono>utm_source={source}</mono>, the campaign's slug and the click token, so orders can be credited to this email.",
  choose: 'Choose products',
  change: 'Change products',
  chosenCount: '{count, plural, one {# product chosen} other {# products chosen}}',
  removedTitle: 'Deleted in {platform} on {date}, {time}',
  removedBody: '{product} is no longer in {store}. Remove it or pick another product.',
  pickAnother: 'Pick another product',
  removeIt: 'Remove it',
  disconnected: "{store} is disconnected: the block keeps its last prices until it's reconnected.",
  fillFromStore: 'Fill from {store}',
  fillHint: "The card takes the product's name, image, price and link from the catalog, and each send reads them again.",
  drawError: "The block couldn't be drawn: {error}",
  emptyGrid: 'Choose products for this grid in the Style tab.',
  checkoutInsert: 'Insert Checkout items',
  checkoutHint: "Checkout items only appear in emails of flows started by a store trigger: the checkout's products, its total and “Return to checkout”.",
  checkoutInserted: 'Checkout items inserted. Save to keep them.',
})

const ATTR = 'rc-store'
const platformName = (p: string) => (p === 'woocommerce' ? 'WooCommerce' : 'Shopify')

/** The store section a component is in (itself included), if any. */
export function storeSectionOf(c: Component | null | undefined): Component | null {
  let cur: Component | undefined = c ?? undefined
  while (cur) {
    if (tagOf(cur) === 'mj-section' && typeof (cur.getAttributes() as Record<string, unknown>)[ATTR] === 'string') return cur
    cur = cur.parent()
  }
  return null
}

function readBlock(section: Component): Block | null {
  const raw = (section.getAttributes() as Record<string, unknown>)[ATTR]
  if (typeof raw !== 'string' || !raw) return null
  try {
    const b = JSON.parse(raw) as Partial<Block>
    if (!b.store || !b.mode) return null
    return { products: [], ...b } as Block
  } catch {
    return null
  }
}

/** Writes a block's settings and drawing on its section, in one undo step. */
function applyDrawing(section: Component, block: Block, html: string) {
  section.addAttributes({ [ATTR]: JSON.stringify(block), 'rc-name': block.mode === 'card' ? 'Product card' : 'Product grid' })
  section.components(`<mj-column><mj-raw>${html}</mj-raw></mj-column>`)
}

async function drawBlock(slug: string, block: Block): Promise<Render> {
  const { data } = await http.POST(`${commerceBase}/blocks/render`, { params: { path: { workspace: slug } }, body: { block } })
  return data as Render
}

/** The workspace's stores with their collections (the picker and the blocks). */
export function useCatalog() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${commerceBase}/catalog`, { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
}

/** The Blocks panel's store blocks: a Product grid, when a store is connected. */
export function useStoreBlockDefs(enabled: boolean) {
  const t = useT()
  const q = useCatalog()
  return useMemo(() => {
    const st = (q.data?.stores ?? []).find((s) => s.status !== 'disconnected')
    if (!enabled || !st) return []
    const block: Block = { store: st.id, mode: 'chosen', products: [], soldOut: 'hide', button: 'Shop now' }
    const cfg = JSON.stringify(block)
    const placeholder = `<!--rc:store-items ${cfg}--><div style="padding:22px 28px;font-family:'IBM Plex Sans',Helvetica,Arial,sans-serif;font-size:13px;color:#5B6472">${t(ce.emptyGrid)}</div><!--/rc:store-items-->`
    return [
      {
        id: 'rc-product-grid',
        label: t(ce.blockGrid),
        badge: 'New',
        group: 'reputecast' as const,
        icon: 'rcGrid',
        kind: 'section' as const,
        content: `<mj-section background-color="#ffffff" padding="0px" ${ATTR}='${cfg.replace(/'/g, '&#39;')}' rc-name="Product grid"><mj-column><mj-raw>${placeholder}</mj-raw></mj-column></mj-section>`,
        keywords: 'shop store shopify woocommerce products merch grid ecommerce',
      },
    ]
  }, [enabled, q.data, t])
}

// ── The product picker ──────────────────────────────────────────────────

function availability(t: ReturnType<typeof useT>, p: Product): { tone: 'ok' | 'mut' | 'warn' | 'bad'; label: string } {
  if (p.removed) return { tone: 'bad', label: t(ce.removed) }
  if (!p.available) return { tone: 'mut', label: t(ce.soldOut) }
  if (p.inventory !== undefined && p.inventory > 0 && p.inventory <= 5) return { tone: 'warn', label: t(ce.left, { count: p.inventory }) }
  return { tone: 'ok', label: t(ce.inStock) }
}

function Merch({ p, h }: { p: Product; h: number }) {
  if (p.imageUrl) return <img src={p.imageUrl} alt={p.imageAlt || p.title} className="cm-merch" style={{ width: '100%', height: h, objectFit: 'cover' }} />
  return (
    <span className="cm-merch" style={{ width: '100%', height: h }} role="img" aria-label={p.imageAlt || p.title}>
      <svg width="34" height="24" viewBox="0 0 40 28" fill="none" stroke="#9AA3B2" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M13 4l-8 5 3 5 4-2v14h16V12l4 2 3-5-8-5c-1 2-3 3-7 3s-6-1-7-3z" />
      </svg>
    </span>
  )
}

/**
 * "Insert products": a store's catalog by collection, searchable, products
 * with their price and availability; Insert chooses them for a grid (or the
 * one product of a card).
 */
export function ProductPicker({
  single,
  storeId,
  initial,
  onInsert,
  onClose,
}: {
  single?: boolean
  storeId?: string
  initial?: string[]
  onInsert: (store: CatalogStore, products: string[]) => void
  onClose: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const catalog = useCatalog()
  const stores = (catalog.data?.stores ?? []).filter((s) => s.status !== 'disconnected' || s.id === storeId)
  const [picked, setPicked] = useState<string | undefined>(storeId)
  const store = stores.find((s) => s.id === picked) ?? stores[0]
  const [collection, setCollection] = useState('')
  const [search, setSearch] = useState('')
  const [chosen, setChosen] = useState<string[]>(initial ?? [])
  const products = $api.useQuery(
    'get',
    `${storesBase}/{id}/products`,
    { params: { path: { workspace: slug, id: store?.id ?? '' }, query: { collection: collection || undefined, search: search.trim() || undefined, limit: 48 } } },
    { enabled: !!store, placeholderData: (prev) => prev },
  )
  const toggle = (id: string) => setChosen((cur) => (single ? [id] : cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id]))
  const total = store?.products ?? 0
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="md"
      className="max-w-[720px]"
      hideClose
      eyebrow={<span className="k">{t(single ? ce.pickerEyebrowCard : ce.pickerEyebrow)}</span>}
      title={t(single ? ce.pickerTitleCard : ce.pickerTitle)}
      footer={
        <>
          <span className="k mr-auto" role="status">
            {t(ce.selected, { count: chosen.length })}
          </span>
          <Button onClick={onClose}>{t(ce.cancel)}</Button>
          <Button variant="primary" disabled={!store || chosen.length === 0} onClick={() => store && onInsert(store, chosen)}>
            {single ? t(ce.insertOne) : t(ce.insert, { count: chosen.length })}
          </Button>
        </>
      }
    >
      {catalog.isPending ? (
        <Skeleton w="100%" h={220} />
      ) : (
        <>
          <div className="row justify-between gap-2.5">
            <div className="seg" role="group" aria-label={t(ce.source)}>
              {stores.map((s) => (
                <button key={s.id} type="button" aria-pressed={s.id === store?.id} className={s.id === store?.id ? 'on' : undefined} onClick={() => setPicked(s.id)}>
                  {t(ce.sourceStore, { name: s.name, count: count(s.products) })}
                </button>
              ))}
            </div>
            <label className="cm-search">
              <Icon name="search" size={15} className="flex-none text-ink-3" />
              <input aria-label={t(ce.search, { name: store?.name ?? '' })} placeholder={t(ce.search, { name: store?.name ?? '' })} value={search} onChange={(e) => setSearch(e.target.value)} type="search" />
            </label>
          </div>
          <div className="cm-ichips" role="group" aria-label={t(ce.collection)}>
            <button type="button" className={`cm-fchip${collection === '' ? ' on' : ''}`} aria-pressed={collection === ''} onClick={() => setCollection('')}>
              {t(ce.all)} <b>{count(total)}</b>
            </button>
            {(store?.collections ?? []).map((c) => (
              <button key={c.id} type="button" className={`cm-fchip${collection === c.id ? ' on' : ''}`} aria-pressed={collection === c.id} onClick={() => setCollection(c.id)}>
                {c.title} <b>{count(c.products)}</b>
              </button>
            ))}
          </div>
          {products.isPending ? (
            <Skeleton w="100%" h={200} />
          ) : (products.data?.items ?? []).length === 0 ? (
            <span className="k">{t(ce.noProducts)}</span>
          ) : (
            <div className="cm-pp-grid">
              {(products.data?.items ?? []).map((p) => {
                const on = chosen.includes(p.id)
                const a = availability(t, p)
                return (
                  <button key={p.id} type="button" className={`cm-ppc${on ? ' on' : ''}`} aria-pressed={on} disabled={p.removed && !on} onClick={() => toggle(p.id)}>
                    <Merch p={p} h={84} />
                    <span className="nm">{p.title}</span>
                    <span className="pr">
                      <b>{money(p.price, p.currency, 'auto')}</b>
                      <Badge tone={a.tone === 'mut' ? 'neutral' : a.tone} className="!h-5 !text-[11px]">
                        {a.label}
                      </Badge>
                    </span>
                  </button>
                )
              })}
            </div>
          )}
          <span className="k">{t(ce.pickerNote)}</span>
        </>
      )}
    </Dialog>
  )
}

// ── The Style tab ───────────────────────────────────────────────────────

function Prop({ title, children, id }: { title?: ReactNode; children: ReactNode; id?: string }) {
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      {title ? (
        <span className="t" id={id}>
          {title}
        </span>
      ) : null}
      {children}
    </div>
  )
}

function Seg<V extends string>({ value, options, onChange, label, disabled }: { value: V; options: { value: V; label: string }[]; onChange: (v: V) => void; label: string; disabled?: boolean }) {
  return (
    <div className="seg self-start" role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o.value} type="button" aria-pressed={value === o.value} className={value === o.value ? 'on' : undefined} disabled={disabled} onClick={() => onChange(o.value)}>
          {o.label}
        </button>
      ))}
    </div>
  )
}

/** A text input that commits on Enter or blur. */
function Draft({ value, onCommit, label, disabled }: { value: string; onCommit: (v: string) => void; label: string; disabled?: boolean }) {
  const [draft, setDraft] = useState(value)
  return (
    <input
      className="in w-[140px]"
      aria-label={label}
      disabled={disabled}
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={() => draft !== value && onCommit(draft)}
      onKeyDown={(e) => e.key === 'Enter' && draft !== value && onCommit(draft)}
    />
  )
}

/**
 * A store block's settings (the Product grid board: products or a
 * collection, how many, sold-out products, the compare-at price, the button
 * text), the products the store removed with what to do about them, and
 * the drawing kept up to date with the store's products.
 */
export function StoreBlockStyle({ section }: { section: Component }) {
  const t = useT()
  const c = useComposer()
  const block = readBlock(section)
  const catalog = useCatalog()
  const store = catalog.data?.stores.find((s) => s.id === block?.store)
  const [picking, setPicking] = useState<null | { replace?: string }>(null)
  const key = block ? JSON.stringify(block) : ''
  const render = useQuery({
    queryKey: ['commerce-render', c.slug, key],
    queryFn: () => drawBlock(c.slug, block!),
    enabled: !!block && (block.products.length > 0 || block.mode === 'collection'),
    staleTime: 30_000,
  })
  const applied = useRef('')
  // A product the store removed since the block was drawn: the canvas flags it.
  useEffect(() => {
    const r = render.data
    if (!r || !block || !c.canEdit || applied.current === r.html) return
    applied.current = r.html
    const raw = section.findType('mj-raw')[0]
    const flagged = raw ? String(raw.toHTML()).includes('No longer in the store') : false
    if (flagged !== r.removed.length > 0) applyDrawing(section, block, r.html)
  }, [render.data, block, c.canEdit, section])
  if (!block) return null
  const disabled = !c.canEdit
  const update = async (patch: Partial<Block>) => {
    const next: Block = { ...block, ...patch }
    try {
      const r = await drawBlock(c.slug, next)
      applied.current = r.html
      applyDrawing(section, next, r.html)
    } catch (e) {
      toast({ kind: 'error', title: t(ce.drawError, { error: errorMessage(e) }) })
    }
  }
  const card = block.mode === 'card'
  const removed = render.data?.removed ?? []
  const first = removed[0]
  const collections = store?.collections ?? []
  return (
    <>
      <div className="flex flex-col gap-0.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
        <span className="font-semibold">{t(card ? ce.blockCard : ce.blockGrid)}</span>
        {store ? <span className="k">{t(ce.gridFrom, { store: store.name })}</span> : null}
      </div>
      {render.data?.disconnected && store ? (
        <Prop>
          <span className="k">{t(ce.disconnected, { store: store.name })}</span>
        </Prop>
      ) : null}
      {first && store ? (
        <Prop>
          <div className="note b" role="alert">
            <AttentionIcon size={16} color="var(--bad-fg)" />
            <div className="flex min-w-0 flex-[1_1_200px] flex-col gap-1">
              <span className="font-bold">
                {first.removedAt ? t(ce.removedTitle, { platform: platformName(first.platform), date: dayMonthDate(first.removedAt), time: clock(first.removedAt) }) : t(ce.removed)}
              </span>
              <div>
                {t(ce.removedBody, { product: first.title, store: store.name })}
                {c.canEdit ? (
                  <div className="mt-2 flex flex-wrap gap-2">
                    <Button size="sm" variant="primary" onClick={() => setPicking({ replace: first.id })}>
                      {t(ce.pickAnother)}
                    </Button>
                    <Button size="sm" onClick={() => void update({ products: block.products.filter((id) => id !== first.id) })}>
                      {t(ce.removeIt)}
                    </Button>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        </Prop>
      ) : null}
      {card ? null : (
        <Prop title={t(ce.products)}>
          <Seg
            label={t(ce.products)}
            value={block.mode === 'collection' ? 'collection' : 'chosen'}
            disabled={disabled}
            onChange={(mode) => void update(mode === 'collection' ? { mode, collection: block.collection || collections[0]?.id || '', count: block.count || 3 } : { mode })}
            options={[
              { value: 'chosen', label: t(ce.chosen) },
              { value: 'collection', label: t(ce.newest) },
            ]}
          />
        </Prop>
      )}
      {block.mode === 'collection' ? (
        <Prop>
          <div className="row2">
            <label className="k">
              {t(ce.collection)}
              <select className="in" disabled={disabled} value={block.collection ?? ''} onChange={(e) => void update({ collection: e.target.value })}>
                {collections.map((col) => (
                  <option key={col.id} value={col.id}>
                    {t(ce.collectionOption, { title: col.title, count: count(col.products) })}
                  </option>
                ))}
              </select>
            </label>
            <label className="k">
              {t(ce.howMany)}
              <select className="in" disabled={disabled} value={String(block.count || 3)} onChange={(e) => void update({ count: Number(e.target.value) })}>
                {[2, 3, 4, 6].map((n) => (
                  <option key={n} value={n}>
                    {n}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </Prop>
      ) : (
        <Prop>
          <span className="k">{t(ce.chosenCount, { count: card ? Math.min(1, block.products.length) : block.products.length })}</span>
          <Button size="sm" className="self-start" disabled={disabled} onClick={() => setPicking({})}>
            {block.products.length ? t(ce.change) : t(ce.choose)}
          </Button>
        </Prop>
      )}
      <Prop title={t(ce.soldOutLabel)}>
        <Seg
          label={t(ce.soldOutLabel)}
          value={block.soldOut === 'label' ? 'label' : 'hide'}
          disabled={disabled}
          onChange={(soldOut) => void update({ soldOut })}
          options={[
            { value: 'hide', label: t(ce.hide) },
            { value: 'label', label: t(ce.label) },
          ]}
        />
      </Prop>
      <Prop>
        <div className="flex items-center justify-between gap-3">
          <span className="flex flex-col gap-0.5">
            <b className="text-[13px] font-semibold">{t(ce.compareAt)}</b>
            <span className="k">{t(ce.compareAtSub)}</span>
          </span>
          <input type="checkbox" role="switch" className="tg" aria-label={t(ce.compareAt)} disabled={disabled} checked={!!block.compareAt} onChange={(e) => void update({ compareAt: e.target.checked })} />
        </div>
        <div className="flex items-center justify-between gap-3">
          <b className="text-[13px] font-semibold">{t(ce.button)}</b>
          <Draft label={t(ce.button)} value={block.button || 'Shop now'} disabled={disabled} onCommit={(button) => void update({ button })} />
        </div>
        <span className="k text-ink-nav">
          <T msg={ce.links} values={{ source: c.slug, mono: (x) => <span className="mono">{x}</span> }} />
        </span>
      </Prop>
      {picking ? (
        <ProductPicker
          single={card}
          storeId={block.store}
          initial={picking.replace ? block.products.filter((id) => id !== picking.replace) : block.products}
          onClose={() => setPicking(null)}
          onInsert={(st, products) => {
            setPicking(null)
            void update({ store: st.id, products: card ? products.slice(0, 1) : products })
          }}
        />
      ) : null}
    </>
  )
}

/** The Product card's "Fill from Acme Merch": the card becomes the store's product. */
export function FillFromStore({ section }: { section: Component }) {
  const t = useT()
  const c = useComposer()
  const catalog = useCatalog()
  const [open, setOpen] = useState(false)
  const id = useId()
  const st = (catalog.data?.stores ?? []).find((s) => s.status !== 'disconnected')
  if (!st || !c.canEdit || markerOf(section) !== 'product-card') return null
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {t(ce.blockCard)}
      </span>
      <Button size="sm" className="self-start" onClick={() => setOpen(true)}>
        {t(ce.fillFromStore, { store: st.name })}
      </Button>
      <span className="k">{t(ce.fillHint)}</span>
      {open ? (
        <ProductPicker
          single
          storeId={st.id}
          onClose={() => setOpen(false)}
          onInsert={(store, products) => {
            setOpen(false)
            const block: Block = { store: store.id, mode: 'card', products: products.slice(0, 1), soldOut: 'label', button: 'View product' }
            drawBlock(c.slug, block)
              .then((r) => applyDrawing(section, block, r.html))
              .catch((e) => toast({ kind: 'error', title: t(ce.drawError, { error: errorMessage(e) }) }))
          }}
        />
      ) : null}
    </div>
  )
}

/**
 * The automation email editor's Checkout items block, for flows a store
 * trigger starts: sample items now, the checkout's own items when it sends.
 */
export function CheckoutItemsButton({ store, onInsert }: { store: string; onInsert: (html: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [busy, setBusy] = useState(false)
  return (
    <div className="flex flex-col gap-1 rounded-lg border border-line px-3 py-2.5">
      <span className="k">{t(ce.checkoutHint)}</span>
      <Button
        size="sm"
        className="self-start"
        loading={busy}
        onClick={() => {
          setBusy(true)
          http
            .GET(`${storesBase}/{id}/checkout-block`, { params: { path: { workspace: slug, id: store } } })
            .then(({ data }) => {
              if (data) {
                onInsert(data.html)
                toast({ kind: 'success', title: t(ce.checkoutInserted) })
              }
            })
            .catch((e) => toast({ kind: 'error', title: errorMessage(e) }))
            .finally(() => setBusy(false))
        }}
      >
        {t(ce.checkoutInsert)}
      </Button>
    </div>
  )
}
