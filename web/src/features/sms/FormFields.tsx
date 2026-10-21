import type { ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { useChannelSetUp } from '@/features/channels/hooks'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { countryName } from './common'
import { m } from './formfields.messages'

type Content = Schemas['SignupFormContent']

/** The words of a sentence that link somewhere, found in any case. */
function findWords(text: string, words: string): { at: number; length: number } | null {
  if (!words) return null
  const hit = text.match(new RegExp(words.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i'))
  return hit && hit.index !== undefined ? { at: hit.index, length: hit[0].length } : null
}

/**
 * The form builder's Fields gain "Mobile number" once texting is set up
 * (brief §6): the phone field with its country picker.
 */
export function SmsPhoneToggle({ on, required, readOnly, onToggle }: { on: boolean; required: boolean; readOnly: boolean; onToggle: (on: boolean) => void }) {
  const t = useT()
  const sms = useChannelSetUp('sms')
  if (!sms && !on) return null
  return (
    <label className="fb-check">
      <input type="checkbox" checked={on} disabled={readOnly} onChange={(e) => onToggle(e.target.checked)} />
      {t(m.phone)}
      {on ? t(required ? m.required : m.optional) : ''}
    </label>
  )
}

/**
 * The form builder's SMS consent box: its own text, unchecked by default,
 * stored with its version as evidence (brief §3.3). It needs the phone field.
 */
export function SmsConsentSettings({
  content,
  saved,
  hasPhone,
  readOnly,
  error,
  onContent,
}: {
  content: Content
  saved: Content
  hasPhone: boolean
  readOnly: boolean
  error?: string
  onContent: (patch: Partial<Content>) => void
}) {
  const t = useT()
  const sms = useChannelSetUp('sms')
  if (!sms && !content.smsConsent) return null
  const changed = (content.smsConsentText ?? '').trim() !== (saved.smsConsentText ?? '') && Boolean(saved.smsConsentText) && (saved.smsConsentVersion ?? 0) > 0
  return (
    <div className="fb-sec">
      <span className="k">{t(m.textsTitle)}</span>
      <label className="fb-check start">
        <input
          type="checkbox"
          checked={Boolean(content.smsConsent)}
          disabled={readOnly || (!hasPhone && !content.smsConsent)}
          aria-describedby={!hasPhone ? 'sms-needs-phone' : undefined}
          onChange={(e) => onContent({ smsConsent: e.target.checked })}
        />
        <span>
          {t(m.smsConsent)}
          {!hasPhone ? (
            <>
              {' '}
              <span className="k" id="sms-needs-phone">
                · {t(m.needsPhone)}
              </span>
            </>
          ) : null}
        </span>
      </label>
      {content.smsConsent ? (
        <>
          <label className="lbl">
            {t(m.consentText)}
            <textarea
              className="fb-in"
              value={content.smsConsentText ?? ''}
              maxLength={600}
              disabled={readOnly}
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? 'sms-consent-err' : undefined}
              onChange={(e) => onContent({ smsConsentText: e.target.value })}
            />
          </label>
          {error ? (
            <span className="err" id="sms-consent-err" role="alert">
              {error}
            </span>
          ) : null}
          <span className="k">
            {t(m.evidence)}
            {changed ? ` ${t(m.newVersion, { version: (saved.smsConsentVersion ?? 1) + 1 })}` : ''}
          </span>
          <span className="k">
            {t(m.separate)} {t(m.confirmation)}
          </span>
        </>
      ) : null}
    </div>
  )
}

/** The preview's phone field: the country picker, then the number. */
export function SmsPhonePreview({ label, optional }: { label?: string; optional?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const sms = useChannelSetUp('sms')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/overview', { params: { path: { workspace: slug } } }, { enabled: sms, retry: false })
  const countries = (q.data?.routes ?? []).flatMap((r) => r.countries)
  const shown = countries.length ? [...new Set(countries)] : ['US']
  return (
    <div className="pf-field">
      <span>
        {label || t(m.phone)}
        {optional ? <span className="pf-opt"> {optional}</span> : null}
      </span>
      <span className="flex gap-1.5">
        <select aria-label={t(m.countryAria)} className="w-auto! max-w-[45%] flex-none" defaultValue={shown[0]}>
          {shown.map((c) => (
            <option key={c} value={c}>
              {countryName(c)}
            </option>
          ))}
        </select>
        <input type="tel" aria-label={label || t(m.phone)} placeholder={t(m.placeholder)} />
      </span>
    </div>
  )
}

/** The preview's SMS consent box, unchecked, with the terms and the privacy policy linked. */
export function SmsConsentPreview({ text, privacy, privacyWords, color }: { text: string; privacy: string; privacyWords?: string; color: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const terms = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/terms', { params: { path: { workspace: slug } } }, { retry: false })
  const links = [
    { hit: findWords(text, t(m.termsWords)), href: terms.data?.url ?? '' },
    { hit: privacy ? findWords(text, privacyWords || 'privacy policy') : null, href: privacy },
  ]
    .filter((l): l is { hit: { at: number; length: number }; href: string } => Boolean(l.hit && l.href))
    .sort((a, b) => a.hit.at - b.hit.at)
  const parts: ReactNode[] = []
  let pos = 0
  for (const l of links) {
    if (l.hit.at < pos) continue
    parts.push(text.slice(pos, l.hit.at))
    parts.push(
      <a key={l.hit.at} href={l.href} style={{ color }} target="_blank" rel="noreferrer">
        {text.slice(l.hit.at, l.hit.at + l.hit.length)}
      </a>,
    )
    pos = l.hit.at + l.hit.length
  }
  parts.push(text.slice(pos))
  return (
    <label className="pf-consent">
      <input type="checkbox" defaultChecked={false} />
      <span>{parts}</span>
    </label>
  )
}
