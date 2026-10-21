import { useQueryClient } from '@tanstack/react-query'
import type { Component } from 'grapesjs'
import { useId, useState } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useCan } from '@/lib/session'
import { useComposer } from '@/features/composer/context'
import { descendants, setAttr } from '@/features/composer/editor/attrs'
import { markerOf, mjChildren, tagOf } from '@/features/composer/editor/labels'
import { PRODUCT, PRODUCTS } from './api'
import { cardPrice, productCardValues, type CardValues } from './elsewhere'
import { mf } from './elsewhere.messages'

/*
 * The composer's Product card filled from a digital product (publish-sell
 * brief §6: "The existing Product card can be filled from a digital
 * product"): its cover and alt text, the name, the price line and the
 * button's link to the product's page.
 */

/**
 * Whether a component is the Product card block's section: marked
 * <!-- rc:product-card -->, or, without a marker, shaped like the block (an
 * image column beside a column of texts and a button). The editor drops a
 * pasted snippet's leading comment, so a card inserted from the Blocks panel
 * has no marker.
 */
export function isProductCard(c: Component): boolean {
  if (tagOf(c) !== 'mj-section') return false
  const marker = markerOf(c)
  if (marker) return marker === 'product-card'
  const cols = mjChildren(c)
  if (cols.length !== 2 || cols.some((x) => tagOf(x) !== 'mj-column')) return false
  const [image, copy] = cols.map((x) => mjChildren(x).map(tagOf))
  return (
    image!.length === 1 &&
    image![0] === 'mj-image' &&
    copy!.filter((t) => t === 'mj-text').length >= 2 &&
    copy!.includes('mj-button') &&
    copy!.every((t) => t === 'mj-text' || t === 'mj-button')
  )
}

const escapeText = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

/** A text's words, in place of what it had (kept as children or as a content string). */
function setText(x: Component, text: string) {
  x.set('content', '')
  x.components(escapeText(text))
}

/** Writes a product into the card: its first image, its first two texts (the name, the price line) and its first button's link. */
export function fillProductCard(section: Component, v: CardValues) {
  const image = descendants(section, 'mj-image')[0]
  if (image) {
    image.addAttributes({ src: v.src, alt: v.alt })
    image.set('src', v.src)
  }
  const [name, price] = descendants(section, 'mj-text')
  if (name) setText(name, v.name)
  if (price) setText(price, v.price)
  const button = descendants(section, 'mj-button')[0]
  if (button) setAttr(button, 'href', v.href)
}

/**
 * The Product card's "Fill from a product" (Inspector › Settings): pick a
 * published product, and the card takes its cover, name, price and page.
 * Everything stays editable afterwards. Needs products:view.
 */
export function FillFromProduct({ c }: { c: Component }) {
  const t = useT()
  const ctx = useComposer()
  const qc = useQueryClient()
  const id = useId()
  const can = useCan('products:view')
  const card = isProductCard(c)
  const list = $api.useQuery('get', PRODUCTS, { params: { path: { workspace: ctx.slug } } }, { enabled: can && card, staleTime: 60_000 })
  const [picked, setPicked] = useState('')
  const [busy, setBusy] = useState(false)
  if (!card || !can) return null
  const products = (list.data?.products ?? []).filter((p) => p.status === 'published')
  const disabled = !ctx.canEdit
  const fill = async () => {
    setBusy(true)
    try {
      // What's live: a published product's draft may differ from what buyers see.
      const d = await qc.fetchQuery($api.queryOptions('get', PRODUCT, { params: { path: { workspace: ctx.slug, product: picked } } }))
      const v = productCardValues(t, d)
      fillProductCard(c, v)
      toast({ kind: 'success', title: t(mf.filled, { name: v.name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(mf.failed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  let body
  if (list.isPending) body = <span className="k">{t(mf.loading)}</span>
  else if (list.isError) body = <span className="k">{errorMessage(list.error)}</span>
  else if (!products.length) body = <span className="k">{t(mf.none)}</span>
  else
    body = (
      <>
        <select className="in" aria-label={t(mf.product)} disabled={disabled} value={picked} onChange={(e) => setPicked(e.target.value)}>
          <option value="">{t(mf.pick)}</option>
          {products.map((p) => (
            <option key={p.id} value={p.id}>
              {t(mf.option, { name: p.name, price: cardPrice(t, p.price, p.currency) })}
            </option>
          ))}
        </select>
        <Button size="sm" className="self-start" loading={busy} busyLabel={t(mf.filling)} disabled={disabled || !picked} onClick={() => void fill()}>
          {t(mf.fill)}
        </Button>
        <span className="k">{t(mf.hint)}</span>
      </>
    )
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {t(mf.title)}
      </span>
      {body}
    </div>
  )
}
