import type { components } from '@/api/schema'
import { andList, count, orList } from '@/lib/format'

export type ListImpact = components['schemas']['ListImpact']
export type ListRestore = components['schemas']['ListRestore']

const plural = (n: number, one: string, many: string) => (n === 1 ? one : many)

/** "Weekly Digest", "Weekly Digest or Beta Program", "A, B, C or 2 other lists". */
export function otherListsText(im: Pick<ListImpact, 'otherLists' | 'otherListCount'>): string {
  const more = im.otherListCount - im.otherLists.length
  return orList(more > 0 ? [...im.otherLists, `${count(more)} other ${plural(more, 'list', 'lists')}`] : im.otherLists)
}

/**
 * The consequences sentence of the Delete list dialog: "14,110 contacts are
 * on this list. 9,802 are also on Weekly Digest and stay subscribed there;
 * 4,308 are on no other list and will become contacts without a
 * subscription. Suppressions are never removed."
 */
export function membersText(im: ListImpact): string {
  const kept = 'Suppressions are never removed.'
  if (im.members === 0) return `Nobody is subscribed to this list. ${kept}`
  const on = `${count(im.members)} ${plural(im.members, 'contact is', 'contacts are')} on this list.`
  const only = im.members - im.elsewhere
  const elsewhere = `${plural(im.elsewhere, 'is', 'are')} also on ${otherListsText(im)} and ${plural(im.elsewhere, 'stays', 'stay')} subscribed there`
  const alone = `${plural(only, 'is', 'are')} on no other list and will become ${plural(only, 'a contact', 'contacts')} without a subscription`
  if (im.elsewhere === 0) return `${on} ${im.members === 1 ? 'It' : 'All of them'} ${alone}. ${kept}`
  if (only === 0) return `${on} ${im.members === 1 ? 'It' : 'All of them'} ${elsewhere}. ${kept}`
  return `${on} ${count(im.elsewhere)} ${elsewhere}; ${count(only)} ${alone}. ${kept}`
}

/**
 * The Affects row: "2 signup forms, 1 automation trigger ("Joined Product
 * Updates") and 3 scheduled campaigns use this list. They will pause."
 * Empty when nothing uses the list.
 */
export function affectsText(im: ListImpact): string {
  const parts: string[] = []
  if (im.forms.length) parts.push(`${count(im.forms.length)} signup ${plural(im.forms.length, 'form', 'forms')}`)
  if (im.automations.length) {
    const names = im.automations.length <= 2 ? ` (${im.automations.map((a) => `"${a.name}"`).join(', ')})` : ''
    parts.push(`${count(im.automations.length)} automation ${plural(im.automations.length, 'trigger', 'triggers')}${names}`)
  }
  if (im.campaigns.length) parts.push(`${count(im.campaigns.length)} scheduled ${plural(im.campaigns.length, 'campaign', 'campaigns')}`)
  if (!parts.length) return ''
  const one = im.forms.length + im.automations.length + im.campaigns.length === 1
  return `${andList(parts)} ${one ? 'uses this list. It will pause.' : 'use this list. They will pause.'}`
}

/** What a restore turned back on, for its toast: "1 signup form and 1 automation are live again; Weekly Digest #142 is back on schedule." */
export function restoredText(r: ListRestore): string {
  const live = [
    r.forms.length ? `${count(r.forms.length)} signup ${plural(r.forms.length, 'form', 'forms')}` : '',
    r.automations.length ? `${count(r.automations.length)} ${plural(r.automations.length, 'automation', 'automations')}` : '',
  ].filter(Boolean)
  const parts: string[] = []
  if (live.length) parts.push(`${andList(live)} ${r.forms.length + r.automations.length === 1 ? 'is' : 'are'} live again`)
  if (r.campaigns.length) parts.push(`${andList(r.campaigns)} ${plural(r.campaigns.length, 'is', 'are')} back on schedule`)
  let out = parts.length ? `${parts.join('; ')}.` : 'Its members are back.'
  if (r.left.length) out += ` ${andList(r.left)} ${plural(r.left.length, 'stays', 'stay')} as ${plural(r.left.length, 'it is', 'they are')}.`
  return out
}
