import { lazy, Suspense } from 'react'
import { useT } from '@/i18n'
import { ViewSeg } from '../CanvasArea'
import { m } from './messages'
import { useTx } from './state'
import { ScrollText } from './ScrollText'
import { PreviewWith } from './Toolbar'

const CodeEditor = lazy(() => import('./CodeEditor'))

/**
 * The plain-text part. Empty means it's generated from the HTML at send
 * time; "Write my own" starts from that generated text. Beside it, the text
 * rendered with the chosen test data.
 */
export function TextView() {
  const t = useT()
  const tx = useTx()
  const own = tx.plainText.trim() !== ''
  const generated = tx.checks?.plainText ?? ''
  return (
    <section id="main" className="cmp-main" style={{ flex: '999 1 620px' }} aria-label={t(m.textViewLabel)}>
      <div className="cmp-tools">
        <ViewSeg />
        <PreviewWith id="tx-text-preview-with" />
        <span className="k" style={{ marginLeft: 'auto' }}>
          {own ? t(m.textOwn) : t(m.textGenerated)}
        </span>
      </div>
      <div className="txe-split">
        <div className="card txe-pane">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="font-semibold">{t(m.textSource)}</span>
            {tx.canEdit ? (
              own ? (
                <button type="button" className="btn sm" onClick={() => tx.setPlainText('')}>
                  {t(m.useGenerated)}
                </button>
              ) : (
                <button type="button" className="btn sm" onClick={() => tx.setPlainText(generated || ' ')}>
                  {t(m.writeOwn)}
                </button>
              )
            ) : null}
          </div>
          {own ? (
            <div className="txe-code light">
              <Suspense fallback={<pre className="txe-pre">{tx.plainText}</pre>}>
                <CodeEditor lang="text" theme="light" label={t(m.textSource)} value={tx.plainText} onChange={tx.canEdit ? tx.setPlainText : undefined} readOnly={!tx.canEdit} />
              </Suspense>
            </div>
          ) : (
            <ScrollText className="txe-pre" label={t(m.textGenerated)}>
              {generated || t(m.checking)}
            </ScrollText>
          )}
          <span className="k">{t(m.textHelp)}</span>
        </div>
        <div className="card txe-pane">
          <span className="font-semibold">{tx.previewWith ? t(m.textRenderedWith, { name: tx.previewWith }) : t(m.textRenderedSamples)}</span>
          <ScrollText className="txe-pre" label={t(m.textRendered)}>
            {tx.preview?.text ?? t(m.checking)}
          </ScrollText>
          {tx.preview?.problem ? <span className="k txe-bad">{tx.preview.problem}</span> : null}
        </div>
      </div>
    </section>
  )
}
