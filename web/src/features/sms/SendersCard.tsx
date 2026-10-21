import { useQueryClient } from '@tanstack/react-query'
import { $api, errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Table, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useChannelSetUp } from '@/features/channels/hooks'
import { useT } from '@/i18n'
import { ago, dayMonthYear } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './senders.messages'

const rulesPath = '/api/v1/workspaces/{workspace}/sms/rules' as const

/**
 * Settings › Senders adds the short-link domain (go.acme.io), with its
 * record and status (brief §6). Shown once texting is set up; the domain
 * is set in Sending providers › SMS › Routing & rules.
 */
export function SmsShortLinkCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const sms = useChannelSetUp('sms')
  const opts = { params: { path: { workspace: slug } } }
  const q = $api.useQuery('get', rulesPath, opts, { enabled: sms })
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/short-domain/check', {
    onSuccess: (next) => {
      qc.setQueryData($api.queryOptions('get', rulesPath, opts).queryKey, next)
      toast({ kind: next.shortDomain.status === 'verified' ? 'success' : 'error', title: t(m.checked) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const data = q.data
  const reason = usePermissionReason('providers:manage', data?.helpers)
  if (!sms || !data) return null
  const d = data.shortDomain
  const params = { workspace: slug }
  const sub = d.domain.split('.')[0] ?? ''
  return (
    <section className="card" aria-labelledby="short-h">
      <div className="sec">
        <div className="flex flex-col">
          <h2 className="h2" id="short-h">
            {t(m.title)}
          </h2>
          <span className="k">{t(m.sub)}</span>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/providers/sms/rules" params={params} aria-label={t(m.manageLabel)}>
          {t(m.manage)}
        </ButtonLink>
      </div>
      {d.domain ? (
        <>
          <Table className="sd-tbl" fit caption={t(m.caption)}>
            <thead>
              <tr>
                <Th>{t(m.domain)}</Th>
                <Th>{t(m.usedBy)}</Th>
                <Th>{t(m.record)}</Th>
                <Th>{t(m.status)}</Th>
                <Th>{t(m.lastCheck)}</Th>
                <Th>
                  <span className="vh">{t(m.checkDns)}</span>
                </Th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td className="nw">
                  <span className="mono font-semibold">{d.domain}</span>
                </td>
                <td className="nw">{t(m.smsLinks)}</td>
                <td className="mono nw">{t(m.cname, { name: sub, target: d.target })}</td>
                <td className="nw">
                  {d.status === 'verified' ? (
                    <Badge tone="ok" dot>
                      {d.verifiedAt ? t(m.verifiedOn, { date: dayMonthYear(d.verifiedAt) }) : t(m.notVerified)}
                    </Badge>
                  ) : d.status === 'failed' ? (
                    <Badge tone="bad">{t(m.checkFailed)}</Badge>
                  ) : (
                    <Badge tone="warn">{t(m.notVerified)}</Badge>
                  )}
                </td>
                <td className="nw">
                  {d.checkedAt ? ago(d.checkedAt) : <span className="k">{t(m.never)}</span>}
                  {d.status !== 'verified' && d.detail ? <div className="k">{d.detail}</div> : null}
                </td>
                <td>
                  <Button size="sm" loading={check.isPending} disabledReason={reason} onClick={() => check.mutate({ params: { path: { workspace: slug } } })}>
                    {t(m.checkDns)}
                  </Button>
                </td>
              </tr>
            </tbody>
          </Table>
          {d.status !== 'verified' ? <div className="pad k border-t border-line-2">{t(m.untilVerified)}</div> : null}
        </>
      ) : (
        <div className="pad flex flex-wrap items-center justify-between gap-2">
          <span className="k">{t(m.none)}</span>
          <ButtonLink size="sm" variant="primary" to="/w/$workspace/providers/sms/rules" params={params}>
            {t(m.setUp)}
          </ButtonLink>
        </div>
      )}
    </section>
  )
}
