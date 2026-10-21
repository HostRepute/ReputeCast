import { Link } from '@tanstack/react-router'
import type { Component } from 'grapesjs'
import type { ReactNode } from 'react'
import { useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useBuilder, type PageCheck } from './context'
import { useForms } from './data'
import { attrOf, isTop, liveTagOf, tagOf } from './editor/nodes'
import { useResumeForm } from './hooks'
import { UiIcon } from './icons'
import { m } from './messages'
import type { T } from './wording'

/** "acme.io/beta-terms" from "https://acme.io/beta-terms". */
export function bareUrl(u: string): string {
  return u.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

/** A check's title, in words (the server words none: it sends the key and the values). */
export function checkTitle(t: T, c: PageCheck): string {
  switch (c.key) {
    case 'address':
      return c.status === 'block' ? t(m.chkAddressTaken, { path: c.path ?? '' }) : c.detail === 'yours' ? t(m.chkAddressYours, { path: c.path ?? '' }) : t(m.chkAddressFree, { path: c.path ?? '' })
    case 'heading':
      return c.status === 'pass' ? t(m.chkHeading) : t(m.chkHeadingMissing)
    case 'moments':
      if (c.status === 'pass') return t(m.chkMoments)
      return c.detail === 'event' ? t(m.chkEventPast, { name: c.name ?? '' }) : t(m.chkCountdownPast, { name: c.name ?? '', date: c.at ? dayMonthDate(c.at) : '' })
    case 'product':
      if (c.status === 'pass') return t(m.chkProduct)
      if (c.problem === 'missing') return t(m.chkProductMissing, { count: c.count ?? 1 })
      if (c.problem === 'draft') return t(m.chkProductDraft, { name: c.name ?? '' })
      if (c.problem === 'archived') return t(m.chkProductArchived, { name: c.name ?? '' })
      return t(m.chkProductDeleted)
    case 'stripe':
      return c.status === 'pass' ? t(m.chkStripe, { account: c.account ?? '', mode: c.mode === 'test' ? t(m.modeTest) : t(m.modeLive) }) : t(m.chkStripeMissing)
    case 'form':
      if (c.status === 'pass') return t(m.chkForm, { name: c.name ?? '' })
      if (c.problem === 'missing') return t(m.chkFormMissing, { count: c.count ?? 1 })
      if (c.problem === 'paused') return t(m.chkFormPaused, { name: c.name ?? '' })
      return t(m.chkFormDeleted)
    case 'images':
      return c.status === 'pass' ? t(m.chkImages, { count: c.count ?? 0 }) : t(m.chkImageNoAlt, { name: c.name || c.ref || '' })
    case 'links':
      return c.status === 'pass' ? t(m.chkLinks, { count: c.count ?? 0 }) : t(m.chkLinkBroken)
    case 'visibility':
      return t(m.chkVisibility)
    case 'store_blocks':
      return t(m.chkStores, { count: c.count ?? 1 })
  }
  return c.key
}

/** What a check that isn't passing explains. */
export function checkDetail(t: T, c: PageCheck): string | undefined {
  if (c.status === 'pass') return c.key === 'product' && c.name ? t(c.pwyw ? m.chkProductDetailPwyw : m.chkProductDetail, { name: c.name, price: c.detail ?? '' }) : undefined
  switch (c.key) {
    case 'address':
      return t(m.chkAddressTakenDetail)
    case 'heading':
      return t(m.chkHeadingDetail)
    case 'moments':
      return t(m.chkMomentsDetail)
    case 'product':
      return c.problem === 'draft' ? t(m.chkProductDraftDetail) : undefined
    case 'stripe':
      return t(m.chkStripeMissingDetail, { count: c.count ?? 1 })
    case 'form':
      return c.problem === 'paused' ? (c.by && c.at ? t(m.chkFormPausedDetail, { by: c.by, date: dayMonthDate(c.at) }) : t(m.chkFormPausedDetailPlain)) : undefined
    case 'images':
      return t(m.chkImageNoAltDetail)
    case 'links':
      return c.httpStatus ? t(m.chkLinkBrokenDetail, { name: c.name ?? '', url: bareUrl(c.ref ?? ''), status: c.httpStatus }) : t(m.chkLinkSilentDetail, { name: c.name ?? '', url: bareUrl(c.ref ?? '') })
    case 'visibility':
      return t(m.chkVisibilityDetail)
    case 'store_blocks':
      return t(m.chkStoresDetail)
  }
  return undefined
}

function Mark({ status }: { status: PageCheck['status'] }) {
  if (status === 'pass') return <UiIcon name="check" size={14} color="#15803D" />
  return <UiIcon name="alert" size={15} color={status === 'block' ? '#B91C1C' : '#B45309'} />
}

/** Finds the first block of the page that matches. */
function find(root: Component | undefined, match: (c: Component) => boolean): Component | undefined {
  let hit: Component | undefined
  root?.onAll((c: Component) => {
    if (!hit && match(c)) hit = c
  })
  return hit
}

function Row({ c, actions }: { c: PageCheck; actions?: ReactNode }) {
  const t = useT()
  const detail = checkDetail(t, c)
  return (
    <div className={`pb-chkr${c.status === 'block' ? ' blk7' : ''}`}>
      <Mark status={c.status} />
      <span className="tx">
        <span style={{ fontWeight: 600 }}>{checkTitle(t, c)}</span>
        {detail ? <span className="k">{detail}</span> : null}
        {actions ? <span className="acts">{actions}</span> : null}
      </span>
    </div>
  )
}

/**
 * The Checks tab (PageBuilderStates · a blocking check, Stripe not
 * connected): blocking checks first, then warnings, then what passed, each
 * with the fix one click away.
 */
export function ChecksTab() {
  const t = useT()
  const b = useBuilder()
  const forms = useForms()
  const { resume, busy } = useResumeForm()
  const wrapper = b.editor?.getWrapper() ?? undefined
  const checks = b.page.checks
  const blocking = checks.filter((c) => c.status === 'block')
  const warnings = checks.filter((c) => c.status === 'warn')
  const passed = checks.filter((c) => c.status === 'pass')
  const show = (c: Component | undefined, field?: string) => {
    if (!c) return
    b.select(c, { scroll: true })
    b.setRightTab('block')
    if (field) b.setFocusField(field)
  }
  const actions = (c: PageCheck): ReactNode => {
    if (c.status === 'pass') return null
    switch (c.key) {
      case 'address':
        return (
          <button
            type="button"
            className="btn sm"
            onClick={() => {
              b.setRightTab('page')
              b.setFocusField('pb-field-slug')
            }}
          >
            {t(m.changeAddress)}
          </button>
        )
      case 'heading': {
        const first = wrapper?.components().find((x: Component) => isTop(x))
        return first ? (
          <button type="button" className="btn sm" onClick={() => show(first)}>
            {t(m.showFirstBlock)}
          </button>
        ) : null
      }
      case 'moments': {
        const tag = c.detail === 'event' ? 'rc-event' : 'rc-countdown'
        const key = tag === 'rc-event' ? 'title' : 'label'
        const block = find(wrapper, (x) => liveTagOf(x) === tag && attrOf(x, key) === (c.name ?? '')) ?? find(wrapper, (x) => liveTagOf(x) === tag)
        return block ? (
          <button type="button" className="btn sm" onClick={() => show(block)}>
            {t(m.showBlock)}
          </button>
        ) : null
      }
      case 'product': {
        const block = find(wrapper, (x) => liveTagOf(x) === 'rc-product' && (c.problem === 'missing' ? !attrOf(x, 'product') : attrOf(x, 'product') === c.ref))
        return (
          <>
            {c.problem === 'draft' && c.ref ? (
              <Link to="/w/$workspace/earn/products/$productId" params={{ workspace: b.slug, productId: c.ref }} className="btn sm pri">
                {t(m.openProduct)}
              </Link>
            ) : null}
            {block && b.canEdit ? (
              <button type="button" className="btn sm" onClick={() => show(block, 'pb-field-product')}>
                {c.problem === 'missing' ? t(m.pickProduct) : t(m.pickAnotherProduct)}
              </button>
            ) : null}
          </>
        )
      }
      case 'stripe':
        return (
          <>
            <Link to="/w/$workspace/earn/settings" params={{ workspace: b.slug }} className="btn sm pri">
              {t(m.connectStripe)}
            </Link>
            <span className="k self-center">{t(m.stripeWhere)}</span>
          </>
        )
      case 'form': {
        const block = find(wrapper, (x) => liveTagOf(x) === 'rc-form' && (c.problem === 'missing' ? !attrOf(x, 'form') : attrOf(x, 'form') === c.ref))
        const name = c.name || forms.items.find((f) => f.id === c.ref)?.name || ''
        return (
          <>
            {c.problem === 'paused' && c.ref && b.canEdit ? (
              <button type="button" className="btn sm pri" aria-busy={busy || undefined} disabled={busy} onClick={() => void resume(c.ref!, name)}>
                {t(m.resumeForm, { name })}
              </button>
            ) : null}
            {block && b.canEdit ? (
              <button type="button" className="btn sm" onClick={() => show(block, 'pb-field-form')}>
                {c.problem === 'missing' ? t(m.pickForm) : t(m.pickAnotherForm)}
              </button>
            ) : null}
          </>
        )
      }
      case 'images': {
        const img = find(wrapper, (x) => tagOf(x) === 'img' && (attrOf(x, 'src') === c.ref || String(x.get('src') ?? '') === c.ref))
        return img ? (
          <button type="button" className="btn sm" onClick={() => show(img, 'pb-field-alt')}>
            {t(m.addAltText)}
          </button>
        ) : null
      }
      case 'links': {
        const a = find(wrapper, (x) => tagOf(x) === 'a' && attrOf(x, 'href') === c.ref)
        return a ? (
          <button type="button" className="btn sm" onClick={() => show(a)}>
            {t(m.showLink)}
          </button>
        ) : null
      }
    }
    return null
  }
  const stripeMissing = blocking.some((c) => c.key === 'stripe')
  return (
    <div>
      {blocking.length ? (
        <>
          <div className="pb-grpk">
            <span>{t(m.grpBlocking, { count: blocking.length })}</span>
          </div>
          {blocking.map((c, i) => (
            <Row key={`b${i}`} c={c} actions={actions(c)} />
          ))}
        </>
      ) : null}
      {warnings.length ? (
        <>
          <div className="pb-grpk">
            <span>{t(m.grpWarnings, { count: warnings.length })}</span>
          </div>
          {warnings.map((c, i) => (
            <Row key={`w${i}`} c={c} actions={actions(c)} />
          ))}
        </>
      ) : null}
      {passed.length ? (
        <>
          <div className="pb-grpk">
            <span>{t(m.grpPassed, { count: passed.length })}</span>
          </div>
          {passed.map((c, i) => (
            <Row key={`p${i}`} c={c} />
          ))}
        </>
      ) : null}
      <div className="pb-pbody">
        {stripeMissing ? (
          <div className="note n" role="note">
            <UiIcon name="info" size={16} color="#4B5563" />
            <div>{t(m.checksStripeNote)}</div>
          </div>
        ) : (
          <span className="k">{t(m.checksFoot)}</span>
        )}
      </div>
    </div>
  )
}
