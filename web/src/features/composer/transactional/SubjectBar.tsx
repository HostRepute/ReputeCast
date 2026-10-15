import { useT } from '@/i18n'
import { useNow } from '@/lib/hooks'
import { useComposer } from '../context'
import { inboxTime } from '../format'
import { m } from './messages'
import { useTx } from './state'

/**
 * Subject and preheader, both Liquid, with the inbox line rendered with
 * the chosen test data (TransactionalTemplate board).
 */
export function TxSubjectBar() {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const now = useNow(60_000)
  const p = tx.preview
  const sender = p?.fromName || tx.template.sender?.name || tx.settings?.defaultSender?.name || tx.settings?.productName || t(m.senderFallback)
  const subject = p ? p.subject : c.subject
  const pre = p ? p.preheader : c.preheader
  const empty = !c.subject.trim() && !c.preheader.trim()
  const label = tx.previewWith ? t(m.inboxWith, { name: tx.previewWith }) : t(m.inboxSamples)
  return (
    <div className="cmp-subj">
      <label style={{ flex: '2 1 300px' }}>
        <span className="k">{t(m.subject)}</span>
        <input
          className="in"
          value={c.subject}
          placeholder={t(m.subjectPlaceholder)}
          onChange={(e) => c.setSubject(e.target.value.replace(/[\r\n]+/g, ' '))}
          readOnly={!tx.canEdit}
          maxLength={998}
        />
      </label>
      <label style={{ flex: '2 1 300px' }}>
        <span className="k">{t(m.preheader)}</span>
        <input
          className="in mono"
          style={{ fontSize: 12 }}
          value={c.preheader}
          placeholder={t(m.preheaderPlaceholder)}
          onChange={(e) => c.setPreheader(e.target.value.replace(/[\r\n]+/g, ' '))}
          readOnly={!tx.canEdit}
          maxLength={500}
        />
      </label>
      <div style={{ flex: '3 1 360px' }}>
        <span className="k" id="tx-inbox-label">
          {label}
        </span>
        {empty ? (
          <div className="cmp-inbox empty" aria-labelledby="tx-inbox-label">
            <span style={{ fontWeight: 700, width: 96 }}>{sender}</span>
            <span>{t(m.inboxEmpty)}</span>
            <span className="k" style={{ marginLeft: 'auto' }}>
              {inboxTime(now)}
            </span>
          </div>
        ) : (
          <div className="cmp-inbox" aria-labelledby="tx-inbox-label">
            <span style={{ fontWeight: 700, width: 96, flex: 'none', overflow: 'hidden', textOverflow: 'ellipsis' }}>{sender}</span>
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>
              <b style={{ fontWeight: 600 }}>{subject}</b>
              {pre ? <span style={{ color: 'var(--ink-3)' }}> - {pre}</span> : null}
            </span>
            <span className="k" style={{ marginLeft: 'auto' }}>
              {inboxTime(now)}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}
