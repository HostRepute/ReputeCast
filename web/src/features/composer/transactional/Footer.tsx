import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { useComposer } from '../context'
import { Breadcrumb } from '../Footer'
import { m } from './messages'
import { useTx } from './state'

function ChecksBadge() {
  const t = useT()
  const { checks } = useTx()
  if (!checks) return null
  if (!checks.blocks && !checks.warnings)
    return (
      <Badge tone="ok" dot>
        {t(m.noWarnings)}
      </Badge>
    )
  return (
    <Badge tone={checks.blocks ? 'bad' : 'warn'} dot>
      {checks.blocks && checks.warnings
        ? t(m.blocksAndWarnings, { blocks: checks.blocks, warnings: checks.warnings })
        : checks.blocks
          ? t(m.blocking, { count: checks.blocks })
          : t(m.warnings, { count: checks.warnings })}
    </Badge>
  )
}

/** Breadcrumb, size against Gmail's clipping, links and images, the checks and the version history. */
export function TxFooter() {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const r = tx.checks
  const left =
    tx.view === 'mjml' ? (
      <span className="k">{t(m.footerMjml)}</span>
    ) : tx.view === 'html' && !tx.htmlMode ? (
      <span className="k">{t(m.footerHtml)}</span>
    ) : tx.view === 'text' ? (
      <span className="k">{t(m.footerText)}</span>
    ) : tx.htmlMode ? (
      <span className="k">{t(m.footerHtmlOnly)}</span>
    ) : c.isEmpty ? (
      <span className="k">{t(m.footerEmpty)}</span>
    ) : (
      <Breadcrumb />
    )
  return (
    <footer className="cmp-foot">
      {left}
      <span className="k" style={{ marginLeft: 'auto' }}>
        {r ? t(m.sizeLine, { size: Math.round(r.sizeBytes / 1024), links: r.links, images: r.images }) : t(m.checking)}
      </span>
      <ChecksBadge />
      <button type="button" className="txe-link k" onClick={tx.openVersions}>
        {t(m.versionHistory, { count: tx.template.versionsCount })}
      </button>
    </footer>
  )
}
