import { useState } from 'react'
import { fieldLabel, PublisherMark, type FormInput, type Overview } from './common'
import { SmsConsentPreview, SmsPhonePreview } from '@/features/sms/FormFields'
import { T, useT } from '@/i18n'
import { m } from './builder.messages'

/**
 * The words of a consent text that link to the privacy policy, ignoring
 * case; the index and length are the text's own.
 */
function findPhrase(text: string, phrase: string): { at: number; length: number } | null {
  if (!phrase) return null
  const m = text.match(new RegExp(phrase.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i'))
  return m && m.index !== undefined ? { at: m.index, length: m[0].length } : null
}

/** The consent sentence with its words for the privacy policy linked when there is a policy. */
export function ConsentText({ text, privacy, color, phrase = 'privacy policy' }: { text: string; privacy: string; color: string; phrase?: string }) {
  const hit = privacy ? findPhrase(text, phrase) : null
  if (!hit) return <>{text}</>
  return (
    <>
      {text.slice(0, hit.at)}
      <a href={privacy} style={{ color }} target="_blank" rel="noreferrer">
        {text.slice(hit.at, hit.at + hit.length)}
      </a>
      {text.slice(hit.at + hit.length)}
    </>
  )
}

/**
 * The form's words in the language it shows (the builder's language view):
 * the built-in labels, list names and privacy link of that language.
 */
export interface PreviewLanguage {
  lang: string
  /** Labels of the built-in fields, by key (email, first_name…). */
  fields: Record<string, string>
  optional: string
  choose: string
  /** The consent text's words that link to the privacy policy. */
  privacy: string
  privacyUrl: string
  listNames: Record<string, string>
  confirmHeading: string
  confirmText: string
  subscribedHeading: string
  subscribedText: string
}

/**
 * The live preview of a form as visitors see it (FormBuilder's center
 * column). It follows the unsaved settings. The list checkboxes are the
 * lists that start checked: ticking one here makes it a default. With a
 * language, it shows the form in that language.
 */
export function FormPreview({
  form,
  overview,
  mobile,
  onToggleDefault,
  readOnly,
  language,
  width = 520,
}: {
  form: FormInput
  overview: Overview
  mobile: boolean
  onToggleDefault: (listId: string) => void
  readOnly: boolean
  language?: PreviewLanguage
  /** The widest the form gets on a desktop preview. */
  width?: number
}) {
  const t = useT()
  const b = overview.branding
  const [agreed, setAgreed] = useState(false)
  const [done, setDone] = useState(false)
  const offered = overview.lists.filter((l) => form.listIds.includes(l.id))
  const c = form.content
  const style = { maxWidth: mobile ? 360 : width }
  const L = language
  const publisher = (s: string) => s.replaceAll('{publisher}', b.publisherName)
  if (done) {
    const confirm = form.doubleOptIn
    const after = c.afterSubmitText
    return (
      <div className="pf" style={style} role="status" lang={L?.lang}>
        <div className="pf-brand">
          <PublisherMark branding={b} className="pf-mark" logoClassName="logo" />
          <span className="text-[16px] font-bold">{c.heading || b.publisherName}</span>
        </div>
        {form.afterSubmit === 'redirect' && form.redirectUrl ? (
          <p className="pf-desc">
            <T msg={m.visitorsGoTo} values={{ url: form.redirectUrl, mono: (c) => <span className="mono text-[13px]">{c}</span> }} />
          </p>
        ) : (
          <div className="pf-done">
            <strong className="text-[16px]">{confirm ? (L?.confirmHeading ?? 'Check your inbox') : (L?.subscribedHeading ?? "You're subscribed")}</strong>
            <p className="pf-desc">
              {after ||
                (confirm
                  ? (L?.confirmText ?? 'We sent you a link to confirm your subscription. It only takes a click.')
                  : publisher(L?.subscribedText ?? `Thanks for joining ${b.publisherName}. The next issue is on its way to your inbox.`))}
            </p>
          </div>
        )}
        <button type="button" className="btn sm self-start" onClick={() => setDone(false)}>
          {t(m.backToForm)}
        </button>
      </div>
    )
  }
  return (
    <div className="pf" style={style} lang={L?.lang}>
      <div className="pf-brand">
        <PublisherMark branding={b} className="pf-mark" logoClassName="logo" />
        <span className="text-[16px] font-bold">{c.heading || b.publisherName}</span>
      </div>
      {c.description ? <p className="pf-desc">{c.description}</p> : null}
      {form.fields.map((f) => {
        // Phase 8: the phone field with its country picker.
        if (f.key === 'phone') return <SmsPhonePreview key={f.key} label={f.label || undefined} optional={f.required ? undefined : (L?.optional ?? 'optional')} />
        const label = f.label || L?.fields[f.key] || fieldLabel(f, overview.customFields)
        const custom = overview.customFields.find((x) => x.key === f.key)
        const optional = !f.required && f.key !== 'email'
        return (
          <label key={f.key} className="pf-field">
            {optional ? (
              <span>
                {label} <span className="pf-opt">{L?.optional ?? 'optional'}</span>
              </span>
            ) : (
              label
            )}
            {custom?.type === 'select' ? (
              <select defaultValue="">
                <option value="">{L?.choose ?? 'Choose…'}</option>
                {custom.options.map((o) => (
                  <option key={o}>{o}</option>
                ))}
              </select>
            ) : (
              <input
                type={f.key === 'email' ? 'email' : custom?.type === 'number' ? 'number' : custom?.type === 'date' ? 'date' : 'text'}
                placeholder={f.key === 'email' ? c.emailPlaceholder : f.key === 'first_name' ? c.namePlaceholder : undefined}
              />
            )}
          </label>
        )
      })}
      {form.listChoice && offered.length > 1 ? (
        <div className="pf-lists">
          <span className="text-[13px] font-semibold">{c.listsLabel}</span>
          {offered.map((l) => (
            <label key={l.id} className="pf-check" title={t(m.checkedOnLoad)}>
              <input type="checkbox" checked={form.defaultListIds.includes(l.id)} disabled={readOnly} onChange={() => onToggleDefault(l.id)} />
              {L?.listNames[l.id] || l.name}
            </label>
          ))}
        </div>
      ) : null}
      {form.consentText ? (
        <label className="pf-consent">
          <input type="checkbox" checked={agreed} onChange={(e) => setAgreed(e.target.checked)} />
          <span>
            <ConsentText text={form.consentText} privacy={L?.privacyUrl || b.privacyUrl} color={b.accentText} phrase={L?.privacy} />
          </span>
        </label>
      ) : null}
      {c.smsConsent && form.fields.some((f) => f.key === 'phone') ? (
        <SmsConsentPreview text={c.smsConsentText ?? ''} privacy={L?.privacyUrl || b.privacyUrl} privacyWords={L?.privacy} color={b.accentText} />
      ) : null}
      <button type="button" className="pf-submit" style={{ background: b.accent, color: b.onAccent }} onClick={() => setDone(true)}>
        {c.button || 'Subscribe'}
      </button>
      {c.footnote ? <span className="pf-foot">{c.footnote}</span> : null}
    </div>
  )
}
