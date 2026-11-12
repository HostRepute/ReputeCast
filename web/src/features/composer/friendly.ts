import type { Component } from 'grapesjs'
import { t as tt } from '@/i18n'
import { mh } from './chrome.messages'
import { mjChildren, sectionLabel, sectionWord, tagOf } from './editor/labels'
import { countdownOf, isCountdown } from '@/features/countdown/block'
import { m as mcd } from '@/features/countdown/messages'
import { inZone, zonedMoment } from '@/features/countdown/time'
import { couponOf, couponPools, isCouponBlock } from '@/features/coupons/composer/block'
import { mc as mcp } from '@/features/coupons/composer/messages'

/**
 * How the composer names a selection for writers (Composer board): a
 * section is "Section · 2 columns" with its name ("Feature highlights"), a
 * column is "Column 2", anything else its kind ("Image", "Button").
 */
export function friendlyLabel(c: Component): { head: string; name?: string; section?: boolean } {
  const tag = tagOf(c)
  if (!tag) return { head: '' }
  const parent = c.parent()
  if (tagOf(parent) === 'mj-body' && tag !== 'mj-raw') {
    const cols = mjChildren(c).filter((x) => tagOf(x) === 'mj-column').length
    const label = sectionLabel(c)
    // "2 columns" says nothing the head doesn't: only a real name follows it.
    const name = /^\d+ columns$/.test(label) ? undefined : sectionWord(label)
    return { head: tt(mh.headSection, { tag: tag.replace(/-/g, '_'), count: cols > 1 ? cols : 0 }), name, section: true }
  }
  // The Countdown and Discount code blocks (M59): "Countdown  Fri 27 Nov, 23:59 EST", "Discount code  Welcome 15%".
  if (isCountdown(c)) {
    const s = countdownOf(c)
    const end = s ? zonedMoment(s.endsAt, s.timeZone) : null
    return { head: tt(mcd.name), name: s && end ? inZone(end, s.timeZone, document.documentElement.lang || 'en') : undefined }
  }
  if (isCouponBlock(c)) return { head: tt(mcp.name), name: couponPools()?.find((p) => p.slug === couponOf(c).pool)?.name }
  if (tag === 'mj-column' && parent) {
    const n = mjChildren(parent).filter((x) => tagOf(x) === 'mj-column').indexOf(c) + 1
    return { head: tt(mh.crumbColumn, { n: Math.max(n, 1) }) }
  }
  return { head: tt(mh.crumbTag, { tag: tag.replace(/-/g, '_') }) }
}
