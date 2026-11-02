import { keepPreviousData } from '@tanstack/react-query'
import { $api, type Schemas } from '@/api/client'
import type { FirstQuery } from '@/lib/prefetch'
import type { ContactsSearch } from './ContactsPage'

/*
 * The queries Contacts asks for first. The page and its route's loader build
 * them here, so a prefetch and the page's own query are the same query; this
 * module stays small, as the route definitions load with the app.
 */

type Filter = Schemas['ContactFilter']

export const filterKeys = [
  'q',
  'list',
  'tier',
  'mailbox',
  'consent',
  'source',
  'tag',
  'joinedAfter',
  'joinedBefore',
  'import',
  'segment',
  'language',
  // Scoring and Growth (X13).
  'band',
  'scoreFrom',
  'scoreTo',
  'growthSource',
  'utmSource',
  'utmMedium',
  'utmCampaign',
  'referringDomain',
  'sort',
  // The latest address check (X14).
  'check',
  // Paid status (Earn, X16).
  'paid',
  'paidTier',
  // Members of the website (X18).
  'signedIn',
  // Buyers of digital products (X19).
  'bought',
  // Store customers and what they spent (X20).
  'storeCustomer',
  'spentFrom',
  'spentTo',
  // The phone and SMS consent (X21).
  'phone',
  'phoneCountry',
  'smsConsent',
  // Push subscribed and WhatsApp consent (X22).
  'pushSubscribed',
  'whatsAppConsent',
] as const

/** The filter part of the search params, without empty values. */
export function filterOf(s: ContactsSearch): Filter {
  const f: Record<string, string | boolean> = {}
  for (const k of filterKeys) {
    const v = s[k]
    if (v) f[k] = typeof v === 'boolean' ? v : String(v)
  }
  return f as Filter
}

/** The contacts list for a search, the fields of its custom columns and "Count all". */
export function contactListQuery(slug: string, search: ContactsSearch, fields: string[], exact: boolean) {
  return $api.queryOptions(
    'get',
    '/api/v1/workspaces/{workspace}/contacts',
    {
      params: {
        path: { workspace: slug },
        query: {
          ...filterOf(search),
          cursor: search.cursor,
          page: search.cursor ? undefined : (search.page ?? 1),
          pageSize: search.size ?? 10,
          exact: exact || undefined,
          fields: fields.length ? fields.join(',') : undefined,
        },
      },
    },
    // ?view= is replaced by the view's filters before the list is read.
    { placeholderData: keepPreviousData, enabled: !search.view },
  )
}

export const contactViewsQuery = (slug: string) => $api.queryOptions('get', '/api/v1/workspaces/{workspace}/contacts/views', { params: { path: { workspace: slug } } })

/** The Audience header's counts, under every Audience tab. */
export const audienceSummaryQuery = (slug: string) => $api.queryOptions('get', '/api/v1/workspaces/{workspace}/audience', { params: { path: { workspace: slug } } })

/**
 * What Contacts asks for first, for its route's loader. The list only for
 * Contacts as the menu opens it: with filters, a page or chosen columns in
 * the address, the screen asks for its list itself.
 */
export function contactsFirst(slug: string, search: object): FirstQuery[] {
  const perm = 'subscribers:view'
  const out: FirstQuery[] = [
    { options: audienceSummaryQuery(slug), perm },
    { options: contactViewsQuery(slug), perm },
  ]
  if (Object.keys(search).length === 0) out.push({ options: contactListQuery(slug, {}, [], false), perm })
  return out
}
