import { $api, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Card, CardHeader } from '@/components/ui/Card'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Select } from '@/components/ui/Field'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ag } from './agent.messages'

/** The human sees pool names and inventory; no code export is available. */
export function AgentPoolSelection({ value, onChange }: { value: string; onChange: (id: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/coupons/pools', { params: { path: { workspace: slug } } })
  const selected = q.data?.items.find((pool) => pool.id === value)
  return <div className="flex min-w-0 flex-col gap-3">
    <p className="k m-0">{t(ag.poolHint)}</p>
    {q.error ? <ErrorPanel title={t(ag.poolFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>} /> : q.isPending ? <Skeleton h={60} /> : <Field label={t(ag.pool)}><Select value={value} onChange={(e) => onChange(e.target.value)}>
      <option value="">{t(ag.poolNone)}</option>{q.data.items.filter((pool) => pool.status === 'active' || pool.id === value).map((pool) => <option key={pool.id} value={pool.id} disabled={pool.status !== 'active'}>{pool.name}</option>)}
    </Select></Field>}
    {selected ? <p className="note i m-0">{t(ag.poolInventory, { count: count(selected.free) })}</p> : null}
    {value ? <div><Button onClick={() => onChange('')}>{t(ag.poolNone)}</Button></div> : null}
  </div>
}

export function AgentPoolReview({ pool }: { pool: Schemas['AIAgentPoolSnapshot'] }) {
  const t = useT()
  return <Card className="flex min-w-0 flex-col gap-3 p-5 max-md:p-4"><CardHeader title={t(ag.pool)} />
    <p className="m-0 text-sm font-semibold [overflow-wrap:anywhere]">{pool.name}</p>
    <p className="k m-0">{t(ag.poolSnapshot, { count: count(pool.free), date: shortDate(pool.at) })}</p>
    <code className="break-all text-xs">{`{{ coupons.${pool.slug}.code }} · {{ coupons.${pool.slug}.url }}`}</code>
    <p className="k m-0">{t(ag.poolHint)}</p>
  </Card>
}
