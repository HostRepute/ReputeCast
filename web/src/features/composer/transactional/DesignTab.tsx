import { useId, useState } from 'react'
import { Segmented } from '@/components/ui/Tabs'
import { useT } from '@/i18n'
import { useComposer } from '../context'
import { tagOf } from '../editor/labels'
import { describe, SettingsFields, StyleFields } from '../Inspector'
import { m } from './messages'
import { useTx } from './state'

const KEY_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/

/** The template's own settings: ID, description, category, sender, reply-to and tracking. */
function TemplateSettings() {
  const t = useT()
  const tx = useTx()
  const tpl = tx.template
  const ids = useId()
  const [key, setKey] = useState(tpl.key)
  const [description, setDescription] = useState(tpl.description)
  const [replyTo, setReplyTo] = useState(tx.meta.replyTo)
  // A saved value (or a teammate's draft, loaded) replaces what the field shows.
  const [saved, setSaved] = useState({ key: tpl.key, description: tpl.description, replyTo: tx.meta.replyTo })
  if (saved.key !== tpl.key || saved.description !== tpl.description || saved.replyTo !== tx.meta.replyTo) {
    setSaved({ key: tpl.key, description: tpl.description, replyTo: tx.meta.replyTo })
    if (saved.key !== tpl.key) setKey(tpl.key)
    if (saved.description !== tpl.description) setDescription(tpl.description)
    if (saved.replyTo !== tx.meta.replyTo) setReplyTo(tx.meta.replyTo)
  }
  const keyError = key && !KEY_RE.test(key) ? t(m.idInvalid) : undefined
  const read = !tx.canEdit
  const senders = tx.settings?.senders ?? []
  const def = tx.settings?.defaultSender
  return (
    <div className="pbody">
      <span className="font-semibold">{t(m.templateSettings)}</span>
      <label className="lbl" htmlFor={`${ids}-key`}>
        {t(m.idLabel)}
        <input
          id={`${ids}-key`}
          className="in mono"
          style={{ fontSize: 12 }}
          value={key}
          readOnly={read || tpl.keyLocked}
          aria-invalid={keyError ? true : undefined}
          aria-describedby={`${ids}-keyh`}
          maxLength={64}
          onChange={(e) => setKey(e.target.value.toLowerCase())}
          onBlur={() => {
            if (!keyError && key && key !== tpl.key) void tx.updateTemplate({ key }).then((ok) => !ok && setKey(tpl.key))
            else if (keyError || !key) setKey(tpl.key)
          }}
        />
      </label>
      <span id={`${ids}-keyh`} className={keyError ? 'k txe-bad' : 'k'}>
        {keyError ?? (tpl.keyLocked ? t(m.idLockedHelp) : t(m.idHelp))}
      </span>
      <label className="lbl">
        {t(m.description)}
        <textarea
          className="in"
          rows={2}
          value={description}
          readOnly={read}
          maxLength={300}
          placeholder={t(m.descriptionPlaceholder)}
          onChange={(e) => setDescription(e.target.value)}
          onBlur={() => description.trim() !== tpl.description && void tx.updateTemplate({ description: description.trim() })}
        />
      </label>
      <label className="lbl">
        {t(m.category)}
        <select className="in" value={tpl.category?.id ?? ''} disabled={read} onChange={(e) => void tx.updateTemplate({ categoryId: e.target.value })}>
          {tpl.category ? null : <option value="">{t(m.noCategory)}</option>}
          {tx.categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.kind === 'optional' ? t(m.categoryOptional, { name: c.name }) : t(m.categoryEssential, { name: c.name })}
            </option>
          ))}
        </select>
      </label>
      <span className="k">{t(m.categoryHelp)}</span>
      <label className="lbl">
        {t(m.sender)}
        <select className="in" value={tx.meta.senderId} disabled={read} onChange={(e) => tx.setMeta({ senderId: e.target.value })}>
          <option value="">{def ? t(m.senderDefaultNamed, { name: def.name, email: def.email }) : t(m.senderDefault)}</option>
          {senders.map((s) => (
            <option key={s.id} value={s.id}>
              {s.verified ? t(m.senderOption, { name: s.name, email: s.email }) : t(m.senderUnverified, { name: s.name, email: s.email })}
            </option>
          ))}
        </select>
      </label>
      <span className="k">{senders.length ? t(m.senderHelp) : t(m.noSenders)}</span>
      <label className="lbl">
        {t(m.replyTo)}
        <input
          className="in"
          type="email"
          value={replyTo}
          readOnly={read}
          placeholder={t(m.replyToPlaceholder)}
          maxLength={254}
          onChange={(e) => setReplyTo(e.target.value)}
          onBlur={() => replyTo.trim() !== tx.meta.replyTo && tx.setMeta({ replyTo: replyTo.trim() })}
        />
      </label>
      <fieldset className="txe-fs" disabled={read}>
        <legend className="font-semibold">{t(m.tracking)}</legend>
        <div className="txe-sw">
          <span className="font-semibold">{t(m.trackOpens)}</span>
          <input type="checkbox" role="switch" className="tg" aria-label={t(m.trackOpens)} checked={tx.meta.trackOpens} onChange={(e) => tx.setMeta({ trackOpens: e.target.checked })} />
        </div>
        <div className="txe-sw">
          <span className="font-semibold">{t(m.trackClicks)}</span>
          <input type="checkbox" role="switch" className="tg" aria-label={t(m.trackClicks)} checked={tx.meta.trackClicks} onChange={(e) => tx.setMeta({ trackClicks: e.target.checked })} />
        </div>
        <span className="k">{tx.kind === 'essential' ? t(m.trackingEssential) : t(m.trackingHelp)}</span>
      </fieldset>
    </div>
  )
}

/**
 * Design: the selected component's Style and Settings, as in the composer;
 * with nothing selected, the template's settings.
 */
export function DesignTab() {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const sel = c.selected && tagOf(c.selected) ? c.selected : null
  if (!sel || tx.view !== 'visual')
    return (
      <>
        {tx.view === 'visual' && !c.isEmpty ? <p className="k txe-hint">{t(m.selectHint)}</p> : null}
        <TemplateSettings />
      </>
    )
  return (
    <>
      <div className="flex flex-col gap-1.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
        <div className="flex items-center justify-between gap-2">
          <span className="font-semibold">{tagOf(sel)}</span>
          <Segmented
            value={tx.designTab}
            onChange={tx.setDesignTab}
            label={t(m.designFields)}
            items={[
              { value: 'style', label: t(m.styleTab) },
              { value: 'settings', label: t(m.settingsTab) },
            ]}
          />
        </div>
        <span className="k">{describe(sel, tx.designTab)}</span>
      </div>
      <fieldset disabled={!tx.canEdit} className="contents">
        {tx.designTab === 'style' ? <StyleFields key={`s${sel.cid}`} c={sel} /> : <SettingsFields key={`t${sel.cid}`} c={sel} />}
      </fieldset>
      <div className="txe-hint" style={{ marginTop: 'auto' }}>
        <button type="button" className="btn sm" onClick={() => c.editor?.select(undefined)}>
          {t(m.templateSettings)}
        </button>
      </div>
    </>
  )
}
