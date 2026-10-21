import { $api } from '@/api/client'
import { useChannelSetUp } from '@/features/channels/hooks'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { countryName } from './common'
import { m } from './contactfilters.messages'

/** The contacts table's search parts this component reads and writes. */
export interface SmsContactSearch {
  phone?: 'yes' | 'no'
  phoneCountry?: string
  smsConsent?: string
}

/**
 * Contacts: "has a phone" with the number's country, and SMS consent (any
 * list, none, or one list), once texting is set up (brief §6).
 */
export function SmsContactFilters({ className, search, onChange }: { className: string; search: SmsContactSearch; onChange: (patch: SmsContactSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const sms = useChannelSetUp('sms')
  const overview = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/overview', { params: { path: { workspace: slug } } }, { enabled: sms, staleTime: 60_000 })
  const lists = $api.useQuery('get', '/api/v1/workspaces/{workspace}/audience', { params: { path: { workspace: slug } } }, { enabled: sms, staleTime: 60_000 })
  if (!sms && !search.phone && !search.smsConsent) return null
  const countries = [...new Set((overview.data?.routes ?? []).flatMap((r) => r.countries))].sort()
  const phoneValue = search.phoneCountry ? `country:${search.phoneCountry}` : (search.phone ?? '')
  return (
    <>
      <select
        className={className}
        aria-label={t(m.phone)}
        value={phoneValue}
        onChange={(e) => {
          const v = e.target.value
          if (v.startsWith('country:')) onChange({ phone: 'yes', phoneCountry: v.slice(8) })
          else onChange({ phone: (v || undefined) as SmsContactSearch['phone'], phoneCountry: undefined })
        }}
      >
        <option value="">{t(m.anyPhone)}</option>
        <option value="yes">{t(m.hasPhone)}</option>
        <option value="no">{t(m.noPhone)}</option>
        {countries.map((c) => (
          <option key={c} value={`country:${c}`}>
            {t(m.phoneIn, { country: countryName(c) })}
          </option>
        ))}
      </select>
      <select className={className} aria-label={t(m.smsConsent)} value={search.smsConsent ?? ''} onChange={(e) => onChange({ smsConsent: e.target.value || undefined })}>
        <option value="">{t(m.anySmsConsent)}</option>
        <option value="any">{t(m.smsConsentAny)}</option>
        <option value="none">{t(m.smsConsentNone)}</option>
        {(lists.data?.lists ?? []).map((l) => (
          <option key={l.id} value={l.id}>
            {t(m.smsConsentOn, { list: l.name })}
          </option>
        ))}
      </select>
    </>
  )
}

/** The Phone column's header and cell, once texting is set up. */
export function usePhoneColumn(): boolean {
  return useChannelSetUp('sms')
}
