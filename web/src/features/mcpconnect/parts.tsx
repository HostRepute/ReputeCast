import { Badge } from '@/components/ui/Badge'
import { Card, CardHeader } from '@/components/ui/Card'
import { md } from '@/features/ai/draft.messages'
import { useT } from '@/i18n'
import { mc } from './messages'
import './mcpconnect.css'

export const capabilities = [
  { key: 'editorial:read', title: mc.editorial, hint: mc.editorialHint },
  { key: 'reports:read', title: mc.reports, hint: mc.reportsHint },
  { key: 'drafts:propose', title: mc.drafts, hint: mc.draftsHint },
] as const

export function ConnectionIdentity({ name, verified }: { name: string; verified: boolean }) {
  const t = useT()
  return <div className="flex flex-wrap items-start justify-between gap-3">
    <h2 className="h2 min-w-0 [overflow-wrap:anywhere]">{name}</h2>
    <Badge tone={verified ? 'ok' : 'warn'} className="whitespace-normal">{t(verified ? mc.verified : mc.unverified)}</Badge>
  </div>
}

export function ConnectionDisclosure() {
  const t = useT()
  return <aside className="mcp-connect-side">
    <Card className="flex flex-col gap-3 p-5 max-md:p-4"><CardHeader title={t(mc.protected)} />
      <p className="m-0 text-ink-2">{t(mc.protectedData)}</p><p className="m-0 text-ink-2">{t(mc.protectedActions)}</p>
      <p className="k m-0">{t(md.sharingRisk)}</p>
    </Card>
    <Card className="flex flex-col gap-3 p-5 max-md:p-4"><CardHeader title={t(mc.separate)} />
      <p className="m-0 text-ink-2">{t(mc.budget)}</p><p className="m-0 text-ink-2">{t(md.externalPolicy)}</p>
    </Card>
  </aside>
}
