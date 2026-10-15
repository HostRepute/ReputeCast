import { lazy, Suspense } from 'react'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { ViewSeg } from '../CanvasArea'
import { m } from './messages'
import { useTx } from './state'
import { PreviewWith } from './Toolbar'

const CodeEditor = lazy(() => import('./CodeEditor'))

/**
 * An HTML-only template (imported as HTML, without MJML): its source and
 * the email rendered with the chosen test data. "Edit on the canvas" puts
 * the HTML in one Raw HTML block of an MJML document.
 */
export function HtmlView() {
  const t = useT()
  const tx = useTx()
  return (
    <section id="main" className="cmp-main" style={{ flex: '999 1 620px' }} aria-label={t(m.htmlViewLabel)}>
      <div className="cmp-tools">
        <ViewSeg />
        <PreviewWith id="tx-html-preview-with" />
        {tx.canEdit ? (
          <button type="button" className="btn" style={{ marginLeft: 'auto' }} onClick={tx.convertToMjml}>
            {t(m.editOnCanvas)}
          </button>
        ) : null}
      </div>
      <div className="flex flex-col gap-3" style={{ padding: '14px 16px 0' }}>
        <Note tone="info">{t(m.htmlOnlyBody)}</Note>
      </div>
      <div className="txe-split">
        <div className="txe-cbx txe-pane-dark">
          <div className="txe-cbh">
            <span>{t(m.htmlSource)}</span>
          </div>
          <Suspense fallback={<pre className="txe-pre-dark">{tx.html}</pre>}>
            <CodeEditor lang="html" label={t(m.htmlSource)} value={tx.html} onChange={tx.canEdit ? tx.setHtml : undefined} readOnly={!tx.canEdit} />
          </Suspense>
        </div>
        <div className="card txe-pane">
          <span className="font-semibold">{tx.previewWith ? t(m.htmlRenderedWith, { name: tx.previewWith }) : t(m.htmlRenderedSamples)}</span>
          <iframe className="txe-frame" title={t(m.htmlRendered)} sandbox="" srcDoc={tx.preview?.html ?? ''} />
        </div>
      </div>
    </section>
  )
}
