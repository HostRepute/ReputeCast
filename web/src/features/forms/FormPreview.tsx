import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { fieldLabel, PublisherMark, type FormInput, type Overview } from './common'
import { SmsConsentPreview, SmsPhonePreview } from '@/features/sms/FormFields'
import { T, useT } from '@/i18n'
import { m } from './builder.messages'
import { severalSteps, type FormField } from './steps'

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
  /** A multi-step form's buttons and "Step {n} of {total}". */
  next?: string
  back?: string
  stepOf?: string
}

/** The copy a variant's preview edits in place. */
export interface PreviewCopy {
  heading?: string
  description?: string
  button?: string
  afterSubmitText?: string
}

/**
 * Text edited where it shows (contentEditable, plain text): it commits on
 * blur or Enter; Escape puts the text back.
 */
function EditableText({ value, label, placeholder, className, onCommit }: { value: string; label: string; placeholder?: string; className?: string; onCommit: (v: string) => void }) {
  const ref = useRef<HTMLSpanElement>(null)
  useLayoutEffect(() => {
    if (ref.current && ref.current.textContent !== value) ref.current.textContent = value
  }, [value])
  return (
    <span
      ref={ref}
      className={`pf-edit ${className ?? ''}`}
      contentEditable="plaintext-only"
      suppressContentEditableWarning
      role="textbox"
      tabIndex={0}
      aria-label={label}
      data-placeholder={placeholder}
      onBlur={(e) => {
        const v = (e.currentTarget.textContent ?? '').replace(/\s+/g, ' ').trim()
        // The saved text until the draft takes the new one (it may refuse it, like an empty button).
        e.currentTarget.textContent = value
        if (v !== value) onCommit(v)
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          e.preventDefault()
          e.currentTarget.blur()
        } else if (e.key === 'Escape') {
          e.currentTarget.textContent = value
          e.currentTarget.blur()
        }
      }}
    />
  )
}

/**
 * The live preview of a form as visitors see it (FormBuilder's center
 * column). It follows the unsaved settings. The list checkboxes are the
 * lists that start checked: ticking one here makes it a default. With a
 * language, it shows the form in that language. A form with several steps
 * shows one step (`step`, or its own Back and Next). With `onCopy` (an
 * A/B test's variant), its heading, description, button and after-submit
 * message are edited in place.
 */
export function FormPreview({
  form,
  overview,
  mobile,
  onToggleDefault,
  readOnly,
  language,
  width = 520,
  step,
  onStep,
  onCopy,
  listsLocked = false,
}: {
  form: FormInput
  overview: Overview
  mobile: boolean
  onToggleDefault: (listId: string) => void
  readOnly: boolean
  /** An A/B test's B or C is shown: the lists are the same in every variant. */
  listsLocked?: boolean
  language?: PreviewLanguage
  /** The widest the form gets on a desktop preview. */
  width?: number
  /** The step shown, from 0 (several steps). */
  step?: number
  onStep?: (n: number) => void
  onCopy?: (patch: PreviewCopy) => void
}) {
  const t = useT()
  const b = overview.branding
  const [agreed, setAgreed] = useState(false)
  const [done, setDone] = useState(false)
  const [ownStep, setOwnStep] = useState(0)
  const offered = overview.lists.filter((l) => form.listIds.includes(l.id))
  const c = form.content
  const style = { maxWidth: mobile ? 360 : width }
  const L = language
  const edit = readOnly ? undefined : onCopy
  const publisher = (s: string) => s.replaceAll('{publisher}', b.publisherName)
  const heading = edit ? (
    <EditableText className="text-[16px] font-bold" value={c.heading} placeholder={b.publisherName} label={t(m.editHeading)} onCommit={(heading) => edit({ heading })} />
  ) : (
    <span className="text-[16px] font-bold">{c.heading || b.publisherName}</span>
  )
  if (done) {
    const confirm = form.doubleOptIn
    const fallback = confirm
      ? (L?.confirmText ?? 'We sent you a link to confirm your subscription. It only takes a click.')
      : publisher(L?.subscribedText ?? `Thanks for joining ${b.publisherName}. The next issue is on its way to your inbox.`)
    return (
      <div className="pf keep-light" style={style} role="status" lang={L?.lang}>
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
            {edit ? (
              <EditableText
                className="pf-desc block"
                value={c.afterSubmitText || fallback}
                label={t(m.editAfterSubmit)}
                onCommit={(v) => edit({ afterSubmitText: v === fallback ? '' : v })}
              />
            ) : (
              <p className="pf-desc">{c.afterSubmitText || fallback}</p>
            )}
          </div>
        )}
        <button type="button" className="btn sm self-start" onClick={() => setDone(false)}>
          {t(m.backToForm)}
        </button>
      </div>
    )
  }

  const field = (f: FormField) => {
    // Phase 8: the phone field with its country picker.
    if (f.key === 'phone') return <SmsPhonePreview key={f.key} label={f.label || undefined} optional={f.required ? undefined : (L?.optional ?? 'optional')} />
    const label = f.label || L?.fields[f.key] || fieldLabel(f, overview.customFields)
    const optional = !f.required && f.key !== 'email'
    const title = optional ? (
      <span>
        {label} <span className="pf-opt">{L?.optional ?? 'optional'}</span>
      </span>
    ) : (
      label
    )
    if (f.type === 'choice') {
      return (
        <fieldset key={f.key} className="pf-choice">
          <legend>{title}</legend>
          {(f.options ?? []).map((o, i) => (
            <label key={i} className="pf-check">
              <input type={f.multiple ? 'checkbox' : 'radio'} name={`pf-${f.key}`} style={{ accentColor: b.accent }} />
              {o.label}
            </label>
          ))}
        </fieldset>
      )
    }
    const custom = overview.customFields.find((x) => x.key === f.key)
    return (
      <label key={f.key} className="pf-field">
        {title}
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
  }

  // The last step (or the form) ends with the lists and the consent boxes.
  const tail = (
    <>
      {form.listChoice && offered.length > 1 ? (
        <div className="pf-lists">
          <span className="text-[13px] font-semibold">{c.listsLabel}</span>
          {offered.map((l) => (
            <label key={l.id} className="pf-check" title={t(m.checkedOnLoad)}>
              <input type="checkbox" checked={form.defaultListIds.includes(l.id)} disabled={readOnly || listsLocked} onChange={() => onToggleDefault(l.id)} />
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
    </>
  )
  const accent = { background: b.accent, color: b.onAccent }
  // The submit button; a variant's text is edited in place, and the
  // after-submit message opens from the button below it.
  const submit = (className: string, text: string, own: boolean): ReactNode =>
    edit && !own ? (
      <div className={`${className} pf-submit-edit`} style={accent}>
        <EditableText value={c.button} label={t(m.editButton)} onCommit={(button) => button && edit({ button })} />
      </div>
    ) : (
      <button type="button" className={className} style={accent} onClick={() => setDone(true)}>
        {text}
      </button>
    )
  const afterSubmit = edit && form.afterSubmit === 'message' ? (
    <button type="button" className="btn sm self-start" onClick={() => setDone(true)}>
      {t(m.showAfterSubmit)}
    </button>
  ) : null

  const steps = severalSteps(c.steps) ? c.steps : undefined
  const current = steps ? Math.min(Math.max(step ?? ownStep, 0), steps.length - 1) : 0
  const go = (n: number) => (onStep ?? setOwnStep)(n)
  const st = steps?.[current]
  const last = steps ? current === steps.length - 1 : true
  const fields = st ? st.fields.map((k) => form.fields.find((f) => f.key === k)).filter((f): f is FormField => Boolean(f)) : form.fields

  return (
    <div className="pf keep-light" style={style} lang={L?.lang}>
      <div className="pf-brand">
        <PublisherMark branding={b} className="pf-mark" logoClassName="logo" />
        {heading}
      </div>
      {edit ? (
        <EditableText className="pf-desc block" value={c.description} placeholder={t(m.addDescription)} label={t(m.editDescription)} onCommit={(description) => edit({ description })} />
      ) : c.description ? (
        <p className="pf-desc">{c.description}</p>
      ) : null}
      {steps && st ? (
        <>
          <div className="pf-dots" aria-hidden="true">
            {steps.map((s, i) => (
              <span key={s.id} className={i === current ? 'on' : undefined} style={i <= current ? { background: b.accent } : undefined} />
            ))}
          </div>
          <span className="pf-stepof" role="status">
            {(L?.stepOf ?? 'Step {n} of {total}').replace('{n}', String(current + 1)).replace('{total}', String(steps.length))}
          </span>
          {st.heading ? <strong className="pf-steph">{st.heading}</strong> : null}
          {st.description ? <span className="pf-stepdesc">{st.description}</span> : null}
          {fields.map(field)}
          {last ? tail : null}
          <div className="pf-nav">
            {current > 0 ? (
              <button type="button" className="pf-btn ghost" onClick={() => go(current - 1)}>
                {L?.back ?? 'Back'}
              </button>
            ) : null}
            {last ? (
              submit('pf-btn', st.button || c.button || 'Subscribe', Boolean(st.button))
            ) : (
              <button type="button" className="pf-btn" style={accent} onClick={() => go(current + 1)}>
                {st.button || (L?.next ?? 'Next')}
              </button>
            )}
          </div>
          {last ? afterSubmit : null}
        </>
      ) : (
        <>
          {fields.map(field)}
          {tail}
          {submit('pf-submit', c.button || 'Subscribe', false)}
          {afterSubmit}
        </>
      )}
      {c.footnote ? <span className="pf-foot">{c.footnote}</span> : null}
    </div>
  )
}
