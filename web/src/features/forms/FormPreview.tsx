import { useState } from 'react'
import { fieldLabel, PublisherMark, type FormInput, type Overview } from './common'

/** The consent sentence with "privacy policy" linked when there is a policy. */
export function ConsentText({ text, privacy, color }: { text: string; privacy: string; color: string }) {
  // A case-insensitive search that returns an index into text itself.
  const at = privacy ? text.search(/privacy policy/i) : -1
  if (at < 0) return <>{text}</>
  return (
    <>
      {text.slice(0, at)}
      <a href={privacy} style={{ color }} target="_blank" rel="noreferrer">
        {text.slice(at, at + 14)}
      </a>
      {text.slice(at + 14)}
    </>
  )
}

/**
 * The live preview of a form as visitors see it (FormBuilder's center
 * column). It follows the unsaved settings. The list checkboxes are the
 * lists that start checked: ticking one here makes it a default.
 */
export function FormPreview({
  form,
  overview,
  mobile,
  onToggleDefault,
  readOnly,
}: {
  form: FormInput
  overview: Overview
  mobile: boolean
  onToggleDefault: (listId: string) => void
  readOnly: boolean
}) {
  const b = overview.branding
  const [agreed, setAgreed] = useState(false)
  const [done, setDone] = useState(false)
  const offered = overview.lists.filter((l) => form.listIds.includes(l.id))
  const c = form.content
  const style = { maxWidth: mobile ? 360 : 520 }
  if (done) {
    const confirm = form.doubleOptIn
    return (
      <div className="pf" style={style} role="status">
        <div className="pf-brand">
          <PublisherMark branding={b} className="pf-mark" logoClassName="logo" />
          <span className="text-[16px] font-bold">{c.heading || b.publisherName}</span>
        </div>
        {form.afterSubmit === 'redirect' && form.redirectUrl ? (
          <p className="pf-desc">
            Visitors go to <span className="mono text-[13px]">{form.redirectUrl}</span> after they submit.
          </p>
        ) : (
          <div className="pf-done">
            <strong className="text-[16px]">{confirm ? 'Check your inbox' : "You're subscribed"}</strong>
            <p className="pf-desc">
              {confirm
                ? 'We sent you a link to confirm your subscription. It only takes a click.'
                : `Thanks for joining ${b.publisherName}. The next issue is on its way to your inbox.`}
            </p>
          </div>
        )}
        <button type="button" className="btn sm self-start" onClick={() => setDone(false)}>
          Back to the form
        </button>
      </div>
    )
  }
  return (
    <div className="pf" style={style}>
      <div className="pf-brand">
        <PublisherMark branding={b} className="pf-mark" logoClassName="logo" />
        <span className="text-[16px] font-bold">{c.heading || b.publisherName}</span>
      </div>
      {c.description ? <p className="pf-desc">{c.description}</p> : null}
      {form.fields.map((f) => {
        const label = fieldLabel(f, overview.customFields)
        const custom = overview.customFields.find((x) => x.key === f.key)
        const optional = !f.required && f.key !== 'email'
        return (
          <label key={f.key} className="pf-field">
            {optional ? (
              <span>
                {label} <span className="pf-opt">optional</span>
              </span>
            ) : (
              label
            )}
            {custom?.type === 'select' ? (
              <select defaultValue="">
                <option value="">Choose…</option>
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
            <label key={l.id} className="pf-check" title="Checked when the form loads">
              <input type="checkbox" checked={form.defaultListIds.includes(l.id)} disabled={readOnly} onChange={() => onToggleDefault(l.id)} />
              {l.name}
            </label>
          ))}
        </div>
      ) : null}
      {form.consentText ? (
        <label className="pf-consent">
          <input type="checkbox" checked={agreed} onChange={(e) => setAgreed(e.target.checked)} />
          <span>
            <ConsentText text={form.consentText} privacy={b.privacyUrl} color={b.accentText} />
          </span>
        </label>
      ) : null}
      <button type="button" className="pf-submit" style={{ background: b.accent, color: b.onAccent }} onClick={() => setDone(true)}>
        {c.button || 'Subscribe'}
      </button>
      {c.footnote ? <span className="pf-foot">{c.footnote}</span> : null}
    </div>
  )
}
