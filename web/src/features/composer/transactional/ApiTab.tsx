import { useMemo, useState } from 'react'
import { Icon } from '@/components/icons/Icon'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { useComposer } from '../context'
import { m } from './messages'
import { acceptedAnswer, apiSample, LANGS, missingAnswer, unpublishedAnswer, type Lang } from './samples'
import { ScrollText } from './ScrollText'
import { useTx } from './state'
import { PREVIEW_TO } from './types'

function copy(text: string, title: string, failed: string, detail: string) {
  void navigator.clipboard.writeText(text).then(
    () => toast({ kind: 'success', title }),
    () => toast({ kind: 'error', title: failed, detail }),
  )
}

/**
 * API (TransactionalTemplateStates · API tab): the exact call that sends
 * this template, in five languages, with the preview's data; and what it
 * answers.
 */
export function ApiTab() {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const [lang, setLang] = useState<Lang>('curl')
  const origin = c.context.publicUrl || window.location.origin
  const input = useMemo(() => ({ origin, workspace: c.slug, template: tx.template.key, to: PREVIEW_TO, data: tx.previewData }), [origin, c.slug, tx.template.key, tx.previewData])
  const code = apiSample(lang, input)
  const required = tx.variables.filter((v) => v.required).map((v) => v.name)
  const version = tx.template.liveVersion
  const label = LANGS.find((l) => l.key === lang)?.label ?? lang
  return (
    <div className="pbody">
      <button type="button" className="btn sm self-start" onClick={() => copy(apiSample('curl', input), t(m.copiedSample, { lang: 'cURL' }), t(m.copyFailed), t(m.copyFailedDetail))}>
        <Icon name="copy" size={13} />
        {t(m.copyCurl)}
      </button>
      <div className="txe-cbx">
        <div className="txe-cbh" role="group" aria-label={t(m.codeSample)}>
          {LANGS.map((l) => (
            <button key={l.key} type="button" className={lang === l.key ? 'on' : undefined} aria-pressed={lang === l.key} onClick={() => setLang(l.key)}>
              {l.label}
            </button>
          ))}
          <button type="button" className="cp" aria-label={t(m.copyCode)} onClick={() => copy(code, t(m.copiedSample, { lang: label }), t(m.copyFailed), t(m.copyFailedDetail))}>
            <Icon name="copy" size={13} />
            {t(m.copy)}
          </button>
        </div>
        <ScrollText className="txe-pre-dark" label={t(m.codeOf, { lang: label })}>
          {code}
        </ScrollText>
      </div>
      <span className="font-semibold">{t(m.answers)}</span>
      <ScrollText className="txe-answer">
        <span className="txe-ok">202 Accepted</span>
        {'\n'}
        {acceptedAnswer()}
      </ScrollText>
      {version && required.length ? (
        <ScrollText className="txe-answer">
          <span className="txe-err">422 Unprocessable Content</span>
          {'\n'}
          {missingAnswer(tx.template.key, version, required)}
        </ScrollText>
      ) : !version ? (
        <ScrollText className="txe-answer">
          <span className="txe-err">422 Unprocessable Content</span>
          {'\n'}
          {unpublishedAnswer(tx.template.key)}
        </ScrollText>
      ) : null}
      <span className="k">
        <T msg={tx.previewWith ? m.apiHelpWith : m.apiHelpSamples} values={{ name: tx.previewWith, mono: (chunks) => <span className="mono">{chunks}</span> }} />
      </span>
    </div>
  )
}
