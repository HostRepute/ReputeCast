import { useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { ChannelSwitch } from '@/features/channels/ChannelSwitch'
import { copyText } from '@/features/push/common'
import { useT } from '@/i18n'
import { ago, andList, clock, count, dayMonthDate, dayMonthYear, orList } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useCan, useMe, usePermissionReason, useWorkspace } from '@/lib/session'
import { DEVELOPERS_URL, QualityBadge, qualityLabel, testStatusWord, throwIfForbidden, useInvalidateWhatsApp, WhatsAppIcon, WhatsAppTabs, type Account, type Overview } from './common'
import { m } from './messages'

/** "8 Oct, 09:14". */
function when(d: string) {
  return `${dayMonthDate(d)}, ${clock(d)}`
}

/** The WhatsApp overview, shared by the three tabs. */
export function useWhatsApp() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/whatsapp', { params: { path: { workspace: slug } } }, {
    refetchInterval: (q) => (q.state.data?.account?.status === 'waiting' ? 5000 : false),
  })
}

/** The page frame of Sending providers › WhatsApp. */
export function WhatsAppFrame({ doc, data, children }: { doc: string; data?: Overview; children: ReactNode }) {
  const t = useT()
  return (
    <Page
      title={t(m.screenTitle)}
      documentTitle={doc}
      filters={
        <>
          <ChannelSwitch current="whatsapp" />
          <WhatsAppTabs />
        </>
      }
      actions={data?.account ? <HeaderActions data={data} /> : null}
    >
      {children}
    </Page>
  )
}

function HeaderActions({ data }: { data: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/whatsapp-campaigns')
  const reason = usePermissionReason('campaigns:edit')
  const a = data.account!
  return (
    <>
      {a.checkedAt ? <span className="k max-md:hidden">{t(m.metaChecked, { ago: ago(a.checkedAt) })}</span> : null}
      <Button
        variant="primary"
        disabledReason={reason}
        loading={create.isPending}
        busyLabel={t(m.creating)}
        onClick={() =>
          create.mutate(
            { params: { path: { workspace: slug } }, body: { name: t(m.newCampaignName) } },
            {
              onSuccess: (c) => void navigate({ to: '/w/$workspace/campaigns/$campaignId/whatsapp', params: { workspace: slug, campaignId: c.id } }),
              onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
            },
          )
        }
      >
        {t(m.newCampaign)}
      </Button>
    </>
  )
}

/** Sending providers › WhatsApp › Account (board WhatsAppChannel). */
export function WhatsAppAccountPage() {
  const t = useT()
  const q = useWhatsApp()
  throwIfForbidden(q.error)
  const data = q.data
  return (
    <WhatsAppFrame doc={t(m.docTitle)} data={data}>
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.loadFailedText)}
        </ErrorPanel>
      ) : !data ? (
        <Loading label={t(m.loading)} className="flex flex-col gap-4">
          <div className="px-stats">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="card px-stat">
                <Skeleton w={90} h={11} />
                <Skeleton w={70} h={22} className="mt-1" />
              </div>
            ))}
          </div>
          <Skeleton w="100%" h={220} />
        </Loading>
      ) : !data.account ? (
        <NotConnected data={data} />
      ) : (
        <>
          {data.canManage ? null : <ReadOnlyNote helpers={data.helpers} />}
          {data.account.status === 'token_rejected' ? <TokenRejected data={data} account={data.account} /> : null}
          {data.account.status === 'waiting' ? <Waiting data={data} account={data.account} /> : null}
          <Tiles data={data} account={data.account} />
          <div className="px-grid">
            <AccountCard data={data} account={data.account} />
            <ConnectionCard data={data} account={data.account} />
          </div>
        </>
      )}
    </WhatsAppFrame>
  )
}

function InfoIcon({ color = '#4B5563' }: { color?: string }) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke={color} strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 7.2v3.8M8 5v.1" />
    </svg>
  )
}

function AlertIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#B91C1C" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 4.8v3.6M8 10.9v.1" />
    </svg>
  )
}

function EyeOffIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#5B6472" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 2l12 12M6.6 6.6a2 2 0 0 0 2.8 2.8M4.2 4.3C2.9 5.2 2 6.5 1.5 8c1.1 2.9 3.6 4.5 6.5 4.5 1.3 0 2.5-.3 3.6-.9M8 3.5c2.9 0 5.4 1.6 6.5 4.5-.3.9-.8 1.7-1.4 2.4" />
    </svg>
  )
}

export function ReadOnlyNote({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { me } = useWorkspace()
  const user = useMe().data?.user
  return (
    <div className="note n" role="note">
      <InfoIcon />
      <div className="flex min-w-0 flex-col gap-1">
        <span className="font-bold">{t(m.readOnlyTitle, { name: user?.name ?? '', role: me.workspace.roleName })}</span>
        <div>
          {t(m.readOnlyText)}
          {helpers.length ? ` ${t(m.askHelpers, { helpers: orList(helpers) })}` : null}
        </div>
      </div>
    </div>
  )
}

function Tiles({ data, account }: { data: Overview; account: Account }) {
  const t = useT()
  const c = data.templateCounts
  return (
    <div className="px-stats">
      <div className="card px-stat">
        <span className="k">{t(m.tileQuality)}</span>
        <span className="v">
          <QualityBadge quality={account.quality} />
        </span>
        <span className="s">{account.checkedAt ? t(m.tileQualitySub, { ago: ago(account.checkedAt) }) : t(m.tileQualitySubNever)}</span>
      </div>
      <div className="card px-stat">
        <span className="k">{t(m.tileLimit)}</span>
        <span className="v">{count(account.messagingLimit)}</span>
        <span className="s">{t(m.tileLimitSub)}</span>
      </div>
      <div className="card px-stat">
        <span className="k">{t(m.tileConsent)}</span>
        <span className="v">{count(data.consent)}</span>
        <span className="s">{data.consentLists.length ? t(m.tileConsentSub, { lists: andList(data.consentLists.map((l) => l.name)) }) : t(m.tileConsentNone)}</span>
      </div>
      <div className="card px-stat">
        <span className="k">{t(m.tileTemplates)}</span>
        <span className="v">{count(c.approved + c.pending + c.rejected + c.other)}</span>
        <span className="s">{t(m.tileTemplatesSub, { approved: c.approved, pending: c.pending, rejected: c.rejected })}</span>
      </div>
    </div>
  )
}

function AccountCard({ data, account: a }: { data: Overview; account: Account }) {
  const t = useT()
  const [testing, setTesting] = useState(false)
  const [disconnecting, setDisconnecting] = useState(false)
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWhatsApp()
  const disconnect = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/whatsapp/account')
  return (
    <section className="card">
      <div className="px-head">
        <div>
          <h2 className="h2">{t(m.accountTitle)}</h2>
          <span className="k">{t(m.accountIntro)}</span>
        </div>
      </div>
      <div className="pad">
        <dl className="kv items-center">
          <dt>{t(m.displayName)}</dt>
          <dd className="flex flex-wrap items-center gap-2">
            <span className="font-semibold">{a.displayName}</span>
            {a.nameStatus === 'approved' ? (
              <Badge tone="ok">{t(m.nameApproved)}</Badge>
            ) : a.nameStatus === 'declined' ? (
              <Badge tone="bad">{t(m.nameDeclined)}</Badge>
            ) : a.nameStatus ? (
              <Badge tone="warn">{t(m.namePending)}</Badge>
            ) : null}
          </dd>
          <dt>{t(m.number)}</dt>
          <dd className="flex flex-wrap items-center gap-2">
            <span className="mono">{a.number}</span>
            {a.phoneStatus === 'connected' ? (
              <Badge tone="ok" dot>
                {t(m.phoneConnected)}
              </Badge>
            ) : a.phoneStatus ? (
              <Badge tone="neutral">{t(m.phoneOther, { status: a.phoneStatus })}</Badge>
            ) : null}
          </dd>
          <dt>{t(m.quality)}</dt>
          <dd className="flex flex-wrap items-center gap-2">
            <QualityBadge quality={a.quality} />
            {a.previousQuality && a.qualityChangedAt ? <span className="k">{t(m.qualityWas, { was: qualityLabel(t, a.previousQuality), date: when(a.qualityChangedAt) })}</span> : null}
          </dd>
          <dt>{t(m.limit)}</dt>
          <dd>
            {t(m.limitLine, { n: a.messagingLimit })}{' '}
            <span className="k">
              {'· '}
              {a.previousLimit && a.limitChangedAt && a.previousLimit > a.messagingLimit
                ? `${t(m.limitLowered, { was: count(a.previousLimit), date: dayMonthDate(a.limitChangedAt) })} · `
                : null}
              {t(m.limitAsMeta)}
            </span>
          </dd>
          <dt>{t(m.businessAccount)}</dt>
          <dd>{t(m.businessLine, { name: a.businessName, hint: a.businessHint })}</dd>
          <dt>{t(m.connected)}</dt>
          <dd>{t(m.connectedBy, { date: dayMonthYear(a.connectedAt), name: a.connectedBy })}</dd>
        </dl>
      </div>
      {data.canManage ? (
        <div className="px-foot flex-col items-start">
          <span className="k">{t(m.testNote)}</span>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" icon="send" onClick={() => setTesting(true)} disabled={a.status !== 'connected'}>
              {t(m.sendTest)}
            </Button>
            <Button size="sm" variant="danger" onClick={() => setDisconnecting(true)}>
              {t(m.disconnect)}
            </Button>
          </div>
          {a.demo ? <span className="k">{t(m.demoNote)}</span> : null}
        </div>
      ) : null}
      {testing ? <TestDialog data={data} onClose={() => setTesting(false)} /> : null}
      <ConfirmDialog
        open={disconnecting}
        onOpenChange={setDisconnecting}
        badge={t(m.disconnect)}
        title={t(m.disconnectTitle)}
        description={t(m.disconnectText)}
        confirmLabel={t(m.disconnect)}
        onConfirm={async () => {
          await disconnect.mutateAsync({ params: { path: { workspace: slug } } })
          toast({ kind: 'deleted', title: t(m.disconnected) })
          void invalidate()
        }}
      />
    </section>
  )
}

function Masked({ label }: { label: ReactNode }) {
  return (
    <span className="px-mask">
      <EyeOffIcon />
      ••••••
      <Badge tone="neutral" size="sm" className="mono">
        {label}
      </Badge>
    </span>
  )
}

function ConnectionCard({ data, account: a }: { data: Overview; account: Account }) {
  const t = useT()
  const [replacing, setReplacing] = useState(false)
  return (
    <section className="card">
      <div className="px-head">
        <div>
          <h2 className="h2">{t(m.connectionTitle)}</h2>
          <span className="k">{t(m.connectionIntro)}</span>
        </div>
        {data.canManage ? (
          <div>
            <Button size="sm" onClick={() => setReplacing(true)}>
              {t(m.replaceCredentials)}
            </Button>
          </div>
        ) : null}
      </div>
      <div className="pad flex flex-col gap-3.5">
        <dl className="kv">
          <dt>{t(m.phoneNumberId)}</dt>
          <dd>
            <Masked label={t(m.writeOnly)} />
          </dd>
          <dt>{t(m.accessToken)}</dt>
          <dd className="flex flex-wrap items-center gap-2">
            <Masked label={t(m.sealed)} />
            <span className="k">{t(m.permanent)}</span>
          </dd>
          <dt>{t(m.appSecret)}</dt>
          <dd>
            <Masked label={t(m.sealed)} />
          </dd>
          <dt>{t(m.webhook)}</dt>
          <dd className="flex flex-col gap-0.5">
            <span className="mono break-all">{a.webhookUrl}</span>
            <span className="k">
              {a.verifiedAt
                ? a.lastEventAt
                  ? t(m.webhookVerified, { date: when(a.verifiedAt), ago: ago(a.lastEventAt) })
                  : t(m.webhookVerifiedNoEvent, { date: when(a.verifiedAt) })
                : t(m.webhookWaiting)}
            </span>
          </dd>
        </dl>
        <span className="k text-ink-nav">{t(m.webhookNote)}</span>
      </div>
      {replacing ? <ReplaceDialog onClose={() => setReplacing(false)} /> : null}
    </section>
  )
}

/** The callback URL and verify token to paste in Meta, until Meta's verification request arrives. */
function Waiting({ data, account: a }: { data: Overview; account: Account }) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow(1000)
  const invalidate = useInvalidateWhatsApp()
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/whatsapp/account/check')
  const since = Math.max(0, now.getTime() - new Date(a.statusSince ?? a.connectedAt).getTime())
  const mins = Math.floor(since / 60000)
  const secs = Math.floor((since % 60000) / 1000)
  const copy = (v: string) => void copyText(v).then((ok) => ok && toast({ kind: 'success', title: t(m.copied) }))
  return (
    <section className="card flex flex-col gap-3 p-[18px]" role="status">
      <div className="flex flex-wrap items-center gap-2.5">
        <span className="wa-wait" aria-hidden="true" />
        <b>{t(m.waitingTitle)}</b>
        <span className="k">{t(m.waitingSince, { time: mins ? t(m.waitedMinutes, { min: mins, s: secs }) : t(m.waitedSeconds, { s: secs }) })}</span>
      </div>
      <span className="text-ink-nav">{t(m.waitingSteps)}</span>
      <dl className="kv items-center">
        <dt>{t(m.callbackUrl)}</dt>
        <dd className="flex flex-wrap items-center gap-2">
          <span className="mono break-all">{a.webhookUrl}</span>
          <Button size="sm" iconOnly icon="copy" aria-label={t(m.copyUrl)} onClick={() => copy(a.webhookUrl)} />
        </dd>
        {a.verifyToken ? (
          <>
            <dt>{t(m.verifyToken)}</dt>
            <dd className="flex flex-wrap items-center gap-2">
              <span className="mono">{a.verifyToken}</span>
              <Button size="sm" iconOnly icon="copy" aria-label={t(m.copyToken)} onClick={() => copy(a.verifyToken!)} />
            </dd>
          </>
        ) : null}
      </dl>
      {data.canManage ? (
        <div className="flex flex-wrap gap-2">
          <Button
            size="sm"
            loading={check.isPending}
            busyLabel={t(m.checking)}
            onClick={() => check.mutate({ params: { path: { workspace: slug } } }, { onSuccess: () => void invalidate(), onError: (e) => toast({ kind: 'error', title: errorMessage(e) }) })}
          >
            {t(m.checkAgain)}
          </Button>
          <ButtonAnchor size="sm" icon="external" href={DEVELOPERS_URL}>
            {t(m.openMetaDevelopers)}
          </ButtonAnchor>
        </div>
      ) : null}
    </section>
  )
}

/** Meta rejected the token: what's held, and the field for a new one. */
function TokenRejected({ data, account: a }: { data: Overview; account: Account }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWhatsApp()
  const [token, setToken] = useState('')
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/whatsapp/account/credentials')
  const at = a.statusSince ? `${dayMonthDate(a.statusSince)} ${clock(a.statusSince)}` : ''
  return (
    <section className="card flex flex-col gap-3.5 p-[18px]" style={{ borderColor: 'var(--bad-line)' }}>
      <div className="note b" role="alert">
        <AlertIcon />
        <div className="px-note-body">
          <b>{a.statusDetail ? t(m.tokenTitle, { date: at, detail: a.statusDetail }) : t(m.tokenTitleNoDetail, { date: at })}</b>
          <div>{a.held.length ? t(m.tokenHeld, { held: andList(a.held.map((h) => `“${h}”`)) }) : t(m.tokenHeldNone)}</div>
        </div>
      </div>
      {data.canManage ? (
        <form
          className="flex flex-wrap items-end gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            save.mutate(
              { params: { path: { workspace: slug } }, body: { accessToken: token.trim() } },
              {
                onSuccess: () => {
                  setToken('')
                  toast({ kind: 'success', title: t(m.reconnected) })
                  void invalidate()
                },
              },
            )
          }}
        >
          <div className="lbl min-w-0 flex-[0_1_360px]">
            <label htmlFor="wa-new-token">{t(m.newToken)}</label>
            <input
              id="wa-new-token"
              className="in w-full"
              type="password"
              autoComplete="off"
              value={token}
              placeholder={t(m.newTokenPh)}
              onChange={(e) => setToken(e.target.value)}
              aria-invalid={save.isError || undefined}
            />
          </div>
          <Button variant="primary" type="submit" loading={save.isPending} busyLabel={t(m.reconnecting)} disabled={!token.trim()}>
            {t(m.reconnect)}
          </Button>
          {save.isError ? (
            <span className="err w-full" role="alert">
              {errorMessage(save.error)}
            </span>
          ) : null}
        </form>
      ) : null}
    </section>
  )
}

/** Not connected: the four write-only values, then the webhook to paste in Meta. */
function NotConnected({ data }: { data: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWhatsApp()
  const canManage = useCan('providers:manage')
  const [v, setV] = useState({ businessAccountId: '', phoneNumberId: '', accessToken: '', appSecret: '' })
  const connect = $api.useMutation('post', '/api/v1/workspaces/{workspace}/whatsapp/account')
  const field = (k: keyof typeof v, label: string, ph: string, secret?: boolean) => (
    <div className="lbl">
      <label htmlFor={`wa-${k}`}>{label}</label>
      <input
        id={`wa-${k}`}
        className="in w-full"
        type={secret ? 'password' : 'text'}
        autoComplete="off"
        spellCheck={false}
        value={v[k]}
        placeholder={ph}
        disabled={!canManage}
        onChange={(e) => setV((x) => ({ ...x, [k]: e.target.value }))}
      />
    </div>
  )
  return (
    <section className="card px-empty" aria-labelledby="wa-empty-title">
      <span className="inline-flex size-[52px] items-center justify-center rounded-full bg-accent-soft text-accent" aria-hidden="true">
        <WhatsAppIcon size={24} />
      </span>
      <h2 className="tt" id="wa-empty-title">
        {t(m.notConnectedTitle)}
      </h2>
      <p className="ds">{t(m.notConnectedText)}</p>
      <form
        className="flex w-full max-w-[560px] flex-col gap-3 rounded-[10px] border border-line p-[18px] text-left"
        onSubmit={(e) => {
          e.preventDefault()
          connect.mutate(
            { params: { path: { workspace: slug } }, body: v },
            {
              onSuccess: () => {
                toast({ kind: 'success', title: t(m.connectedToast) })
                void invalidate()
              },
            },
          )
        }}
      >
        {field('businessAccountId', t(m.wabaId), t(m.wabaIdPh))}
        {field('phoneNumberId', t(m.phoneNumberId), t(m.phoneIdPh))}
        {field('accessToken', t(m.tokenLabel), t(m.tokenPh), true)}
        {field('appSecret', t(m.appSecret), t(m.secretPh), true)}
        <span className="k">{t(m.writeOnlyAll)}</span>
        {connect.isError ? (
          <span className="err" role="alert">
            {errorMessage(connect.error)}
          </span>
        ) : null}
        <div className="flex justify-end">
          <Button
            variant="primary"
            type="submit"
            loading={connect.isPending}
            busyLabel={t(m.connecting)}
            disabledReason={canManage ? undefined : t(m.connectNeeds, { helpers: orList(data.helpers) })}
            disabled={canManage && Object.values(v).some((x) => !x.trim())}
          >
            {t(m.connect)}
          </Button>
        </div>
      </form>
    </section>
  )
}

function ReplaceDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWhatsApp()
  const [v, setV] = useState({ businessAccountId: '', phoneNumberId: '', accessToken: '', appSecret: '' })
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/whatsapp/account/credentials')
  const field = (k: keyof typeof v, label: string, secret?: boolean) => (
    <div className="lbl">
      <label htmlFor={`wa-r-${k}`}>{label}</label>
      <input
        id={`wa-r-${k}`}
        className="in w-full"
        type={secret ? 'password' : 'text'}
        autoComplete="off"
        spellCheck={false}
        value={v[k]}
        onChange={(e) => setV((x) => ({ ...x, [k]: e.target.value }))}
      />
    </div>
  )
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.replaceTitle)}
      description={t(m.replaceText)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button
            variant="primary"
            loading={save.isPending}
            disabled={Object.values(v).every((x) => !x.trim())}
            onClick={() =>
              save.mutate(
                {
                  params: { path: { workspace: slug } },
                  body: Object.fromEntries(Object.entries(v).filter(([, x]) => x.trim())) as Schemas['WhatsAppCredentialsInput'],
                },
                {
                  onSuccess: () => {
                    toast({ kind: 'success', title: t(m.replaced) })
                    void invalidate()
                    onClose()
                  },
                },
              )
            }
          >
            {t(m.replaceSave)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {field('businessAccountId', t(m.wabaId))}
        {field('phoneNumberId', t(m.phoneNumberId))}
        {field('accessToken', t(m.tokenLabel), true)}
        {field('appSecret', t(m.appSecret), true)}
        {save.isError ? (
          <span className="err" role="alert">
            {errorMessage(save.error)}
          </span>
        ) : null}
      </div>
    </Dialog>
  )
}

function TestDialog({ data, onClose }: { data: Overview; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const approved = data.templates.filter((x) => x.status === 'approved')
  const [phone, setPhone] = useState(data.testNumbers[0] ?? '')
  const [templateId, setTemplateId] = useState(approved[0]?.id ?? '')
  const [testId, setTestId] = useState<string>()
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/whatsapp/test')
  const status = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/whatsapp/tests/{messageId}',
    { params: { path: { workspace: slug, messageId: testId ?? '' } } },
    { enabled: Boolean(testId), refetchInterval: (q) => (q.state.data && ['delivered', 'failed', 'undelivered', 'cancelled', 'skipped'].includes(q.state.data.status) ? false : 2000) },
  )
  const r = status.data
  const word = (s: Schemas['WhatsAppTestResult']['status']) => t(testStatusWord[s])
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.testTitle)}
      description={t(m.testText)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button
            variant="primary"
            icon="send"
            loading={send.isPending}
            busyLabel={t(m.testSending)}
            disabled={!phone.trim() || !approved.length}
            onClick={() =>
              send.mutate(
                { params: { path: { workspace: slug } }, body: { phone: phone.trim(), templateId: templateId || undefined } },
                {
                  onSuccess: (res) => {
                    setTestId(res.id)
                    toast({ kind: 'success', title: t(m.testQueued, { to: res.to }) })
                  },
                },
              )
            }
          >
            {t(m.testSend)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {approved.length ? (
          <div className="lbl">
            <label htmlFor="wa-test-template">{t(m.testTemplate)}</label>
            <select id="wa-test-template" className="in w-full" value={templateId} onChange={(e) => setTemplateId(e.target.value)}>
              {approved.map((x) => (
                <option key={x.id} value={x.id}>
                  {x.name} · {x.languageLabel}
                </option>
              ))}
            </select>
          </div>
        ) : (
          <div className="note w" role="note">
            <span>{t(m.testNoTemplates)}</span>
          </div>
        )}
        <div className="lbl">
          <label htmlFor="wa-test-phone">{t(m.testPhone)}</label>
          <input id="wa-test-phone" className="in w-full" type="tel" list="wa-test-numbers" value={phone} placeholder={t(m.testPhonePh)} onChange={(e) => setPhone(e.target.value)} />
          <datalist id="wa-test-numbers">
            {data.testNumbers.map((n) => (
              <option key={n} value={n} />
            ))}
          </datalist>
        </div>
        {send.isError ? (
          <span className="err" role="alert">
            {errorMessage(send.error)}
          </span>
        ) : null}
        {r ? (
          <div className={r.status === 'failed' || r.status === 'undelivered' ? 'note b' : r.status === 'delivered' ? 'note g' : 'note i'} role="status">
            <span>
              {r.status === 'failed' || r.status === 'undelivered'
                ? t(m.testFailed, { template: r.template, to: r.to, reason: r.reason ?? word(r.status) })
                : r.status === 'delivered' && r.read
                  ? t(m.testStatusRead, { template: r.template, to: r.to })
                  : t(m.testStatus, { template: r.template, to: r.to, status: word(r.status) })}
            </span>
          </div>
        ) : null}
      </div>
    </Dialog>
  )
}
