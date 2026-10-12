import { useEffect, useRef, useState } from 'react'
import type { Editor } from 'grapesjs'
import { createComposerEditor } from './editor/grapes'
import { canvasHead, joinMjml, splitMjml } from './editor/mjmlDoc'
import './composer.css'

const SAMPLE = `<mjml>
  <mj-head>
    <mj-title>Three launches, one deprecation, and the roadmap for Q4</mj-title>
    <mj-preview>Feature flags are GA, audit log exports, and what we're retiring in November.</mj-preview>
    <mj-attributes>
      <mj-all font-family="Georgia, 'Times New Roman', serif"></mj-all>
      <mj-text color="#1F2937" padding="0px"></mj-text>
    </mj-attributes>
  </mj-head>
  <mj-body width="600px" background-color="#F3F4F6">
    <mj-section background-color="#ffffff" padding="20px 24px" border-bottom="1px solid #EEF0F3">
      <mj-column width="40%" vertical-align="middle">
        <mj-image src="" alt="Acme" width="96px" height="28px" align="left" padding="0px"></mj-image>
      </mj-column>
      <mj-column width="60%" vertical-align="middle">
        <mj-text align="right" font-family="'IBM Plex Sans', Helvetica, Arial, sans-serif" font-size="12px" color="#5B6472">Blog · Docs · Changelog</mj-text>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="20px 24px 8px">
      <mj-column>
        <mj-image src="" alt="Q4 roadmap illustration" width="552px" height="180px" padding="0px 0px 16px"></mj-image>
        <mj-text font-size="26px" line-height="1.2" font-weight="700" color="#0B1220" padding="0px 0px 10px">Three launches, one deprecation, and the roadmap for Q4</mj-text>
        <mj-text font-size="15px" line-height="1.55">Hi {{ first_name | default: "there" }}, here's what shipped this week at Acme, what we're retiring in November, and where the next quarter is headed.</mj-text>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="16px 24px">
      <mj-column width="50%" padding="0px 8px 0px 0px">
        <mj-image src="" alt="Feature flags dashboard" width="268px" height="120px" padding="0px 0px 10px"></mj-image>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Feature flags are generally available</mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">Target rollouts by plan, region or percentage, with an audit trail for every change.</mj-text>
        <mj-button href="https://docs.acme.io/flags" align="left" background-color="#0F766E" color="#ffffff" font-family="'IBM Plex Sans', Helvetica, Arial, sans-serif" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px">Read the docs</mj-button>
      </mj-column>
      <mj-column width="50%" padding="0px 0px 0px 8px">
        <mj-image src="" alt="Audit log export settings" width="268px" height="120px" padding="0px 0px 10px"></mj-image>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Audit log exports</mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">Stream every workspace event to S3 or your SIEM, with 13 months of history.</mj-text>
        <mj-button href="https://acme.io/changelog" align="left" background-color="#0F766E" color="#ffffff" font-family="'IBM Plex Sans', Helvetica, Arial, sans-serif" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px">See the changelog</mj-button>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="8px 24px">
      <mj-column><mj-divider border-width="1px" border-color="#E3E6EB" padding="0px"></mj-divider></mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="8px 24px 16px">
      <mj-column>
        <mj-text font-size="17px" font-weight="700" color="#0B1220" padding="0px 0px 6px">Retiring the v1 API on 30 November</mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151">If you still call api.acme.io/v1, the <a href="https://docs.acme.io/migrate-v2" style="color:#0F766E">migration guide</a> covers every endpoint. Most teams finish in an afternoon.</mj-text>
      </mj-column>
    </mj-section>
    <!-- rc:footer -->
    <mj-section background-color="#F3F4F6" padding="18px 24px">
      <mj-column>
        <mj-text align="center" font-family="'IBM Plex Sans', Helvetica, Arial, sans-serif" font-size="12px" line-height="1.6" color="#5B6472">Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105<br><a href="{{ unsubscribe_url }}" style="color:#0F766E;text-decoration:none">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:#0F766E;text-decoration:none">Email preferences</a> · <a href="{{ view_in_browser_url }}" style="color:#0F766E;text-decoration:none">View in browser</a></mj-text>
      </mj-column>
    </mj-section>
  </mj-body>
</mjml>`

export function Prototype() {
  const ref = useRef<HTMLDivElement>(null)
  const [ed, setEd] = useState<Editor>()
  const [html, setHtml] = useState('')
  useEffect(() => {
    let editor: Editor | undefined
    let cancelled = false
    const parts = splitMjml(SAMPLE)
    void createComposerEditor(ref.current!, { body: parts.body, head: () => canvasHead(parts.head), device: 'desktop' }).then((r) => {
      if (cancelled) {
        r.editor.destroy()
        return
      }
      editor = r.editor
      setEd(r.editor)
      const out = r.compile(joinMjml(parts.head, r.editor.getHtml()))
      setHtml(`${out.html.length} bytes, ${out.errors.length} errors`)
      ;(window as unknown as { ed: Editor }).ed = r.editor
    })
    return () => {
      cancelled = true
      editor?.destroy()
    }
  }, [])
  return (
    <div className="cmp">
      <div style={{ padding: 8, background: '#fff' }}>
        prototype · {ed ? 'ready' : 'loading'} · {html}
      </div>
      <div className="cmp-canvas">
        <div ref={ref} />
      </div>
    </div>
  )
}
