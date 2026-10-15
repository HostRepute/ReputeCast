import { useT } from '@/i18n'
import { DeviceSeg, ViewSeg } from '../CanvasArea'
import { m } from './messages'
import { useTx } from './state'

/** "Preview with": the test data set the canvas, the inbox line and the API tab use. */
export function PreviewWith({ id = 'tx-preview-with' }: { id?: string }) {
  const t = useT()
  const tx = useTx()
  return (
    <label className="txe-with" htmlFor={id}>
      {t(m.previewWith)}
      <select id={id} className="in" value={tx.previewWith} onChange={(e) => tx.setPreviewWith(e.target.value)}>
        {tx.testData.map((s) => (
          <option key={s.name} value={s.name}>
            {s.name}
          </option>
        ))}
        <option value="">{t(m.samples)}</option>
      </select>
    </label>
  )
}

/** The canvas toolbar (TransactionalTemplate): device, view, Preview with, zoom and shortcuts. */
export function TxToolbar({ zoom }: { zoom: number }) {
  const t = useT()
  return (
    <div className="cmp-tools">
      <DeviceSeg />
      <ViewSeg />
      <PreviewWith />
      <span className="k" style={{ marginLeft: 'auto' }}>
        {t(m.zoomHint, { zoom: Math.round(zoom * 100) })}
      </span>
    </div>
  )
}
