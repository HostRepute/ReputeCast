import type { components } from '@/api/schema'
import { t } from '@/i18n'
import { andList, count, orList } from '@/lib/format'
import { m } from './lists.messages'

export type ListImpact = components['schemas']['ListImpact']
export type ListRestore = components['schemas']['ListRestore']

/** "Weekly Digest", "Weekly Digest or Beta Program", "A, B, C or 2 other lists". */
export function otherListsText(im: Pick<ListImpact, 'otherLists' | 'otherListCount'>): string {
  const more = im.otherListCount - im.otherLists.length
  return orList(more > 0 ? [...im.otherLists, t(m.otherLists, { n: more, count: count(more) })] : im.otherLists)
}

/**
 * The consequences sentence of the Delete list dialog: "14,110 contacts are
 * on this list. 9,802 are also on Weekly Digest and stay subscribed there;
 * 4,308 are on no other list and will become contacts without a
 * subscription. Suppressions are never removed."
 */
export function membersText(im: ListImpact): string {
  if (im.members === 0) return t(m.nobody)
  const on = t(m.onList, { n: im.members, count: count(im.members) })
  const only = im.members - im.elsewhere
  const lists = otherListsText(im)
  const middle =
    im.elsewhere === 0
      ? t(m.allAlone, { n: im.members })
      : only === 0
        ? t(m.allElsewhere, { n: im.members, lists })
        : t(m.split, { elsewhere: im.elsewhere, elsewhereCount: count(im.elsewhere), lists, alone: only, aloneCount: count(only) })
  return `${on} ${middle} ${t(m.kept)}`
}

/**
 * The Affects row: "2 signup forms, 1 automation trigger ("Joined Product
 * Updates") and 3 scheduled campaigns use this list. They will pause."
 * Empty when nothing uses the list.
 */
export function affectsText(im: ListImpact): string {
  const parts: string[] = []
  if (im.forms.length) parts.push(t(m.forms, { n: im.forms.length, count: count(im.forms.length) }))
  if (im.automations.length) {
    const n = { n: im.automations.length, count: count(im.automations.length) }
    parts.push(im.automations.length <= 2 ? t(m.automationsNamed, { ...n, names: im.automations.map((a) => `"${a.name}"`).join(', ') }) : t(m.automations, n))
  }
  if (im.campaigns.length) parts.push(t(m.campaigns, { n: im.campaigns.length, count: count(im.campaigns.length) }))
  if (!parts.length) return ''
  const one = im.forms.length + im.automations.length + im.campaigns.length === 1
  return t(m.usesIt, { one: one ? 'yes' : 'no', things: andList(parts) })
}

/** What a restore turned back on, for its toast: "1 signup form and 1 automation are live again; Weekly Digest #142 is back on schedule." */
export function restoredText(r: ListRestore): string {
  const live = [
    r.forms.length ? t(m.restoredForms, { n: r.forms.length, count: count(r.forms.length) }) : '',
    r.automations.length ? t(m.restoredAutomations, { n: r.automations.length, count: count(r.automations.length) }) : '',
  ].filter(Boolean)
  const parts: string[] = []
  if (live.length) parts.push(t(m.liveAgain, { one: r.forms.length + r.automations.length === 1 ? 'yes' : 'no', things: andList(live) }))
  if (r.campaigns.length) parts.push(t(m.backOnSchedule, { n: r.campaigns.length, campaigns: andList(r.campaigns) }))
  let out = parts.length ? `${parts.join('; ')}.` : t(m.membersBack)
  if (r.left.length) out += ` ${t(m.stayAsIs, { n: r.left.length, names: andList(r.left) })}`
  return out
}
