import { Link } from '@tanstack/react-router'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { clock, orList } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { useComposer, type SaveStatus } from './context'
import { statusWord } from './format'
import { ToolIcon } from './icons'
import { AudiencePicker, SenderPicker } from './Pickers'
import { PaidOnlyBadge } from '@/features/earn/composer/EarnBlocks'
import { SendTestButton } from './SendTest'
import { ComposerLanguageSwitcher, versionLanguage, withoutVersion } from '@/features/languages/composerParts'
import { DigestBadge, DigestHeaderActions, isDigestTemplate } from '@/features/digests/composer/DigestMode'
import { IssueHeaderLabel, IssueHeaderStatus, isDigestIssue } from '@/features/digests/composer/IssueMode'
import { dc } from '@/features/digests/composer/messages'
import { T, t as tt, useT } from '@/i18n'
import { m } from './messages'
import { useAssistBanner } from '@/features/ai/composer/panel'
import { AssistMenu } from '@/features/ai/composer/assistMenu'
import { WebChip } from '@/features/website/composer/campaign'

const routeColors: Record<string, string> = { ses: 'var(--chart-ses)', postal: 'var(--chart-postal)', postmark: 'var(--chart-postmark)' }

/** The header badge: "Draft · autosaved 09:41 · v12". Every form ends with " · v{version}" (withoutVersion drops it). */
export function statusText(status: SaveStatus, campaignStatus: string): string {
  const word = statusWord(campaignStatus)
  switch (status.kind) {
    case 'new':
      return tt(m.statusNew)
    case 'saving':
      return tt(m.statusSaving, { status: word, version: status.version })
    case 'offline':
      return tt(m.statusNotSaved, { status: word, version: status.version })
    case 'conflict':
      return tt(m.statusConflict, { status: word, version: status.version })
    case 'error':
      return tt(m.statusNotSaved, { status: word, version: status.version })
    case 'dirty':
      return status.at ? tt(m.statusAutosaved, { status: word, time: clock(status.at), version: status.version }) : tt(m.statusEditing, { status: word, version: status.version })
    default:
      return tt(m.statusAutosaved, { status: word, time: clock(status.at), version: status.version })
  }
}

function RouteButton() {
  const { campaign, context, slug } = useComposer()
  const t = useT()
  // A campaign without its own split follows the Broadcasts routing rule.
  const fromRule = campaign.route.source !== 'campaign'
  const route = campaign.route.shares.length ? campaign.route : context.broadcastRoute
  // The words the button shows, so its accessible name starts with them (WCAG 2.5.3).
  const visible = fromRule
    ? route.shares.length
      ? t(m.routeBroadcasts, { shares: route.shares.map((s) => `${s.short} ${Math.round(s.percent)}`).join(' / ') })
      : t(m.routeNotSetUp)
    : route.shares.map((s) => `${s.short} ${Math.round(s.percent)}%`).join(' ')
  const label = (
    <>
      {fromRule ? (
        <>
          <span className="dot" style={{ color: routeColors[route.shares[0]?.adapter ?? ''] ?? 'var(--accent)' }} aria-hidden="true" />
          {visible}
        </>
      ) : (
        route.shares.map((s) => (
          <span key={s.providerId} className="inline-flex items-center gap-1.5">
            <span className="dot" style={{ color: routeColors[s.adapter] ?? 'var(--chart-other)' }} aria-hidden="true" />
            {s.short} {Math.round(s.percent)}%
          </span>
        ))
      )}
    </>
  )
  if (!campaign.access.canManageRouting) {
    return (
      <span className="btn" aria-disabled="true" title={t(m.routeNeedsPermission)} style={{ opacity: 0.7, cursor: 'default' }}>
        {fromRule ? label : <T msg={m.routeReadOnly} values={{ route: label }} />}
      </span>
    )
  }
  return (
    <Link to="/w/$workspace/providers/routing" params={{ workspace: slug }} className="btn" aria-label={t(m.routeOpen, { route: visible })}>
      {label}
    </Link>
  )
}

/** The composer's top bar: back, name, status, audience, sender, route and actions. */
export function Header({ onReview }: { onReview: () => void }) {
  const c = useComposer()
  const { campaign, view, slug, status, canEdit } = c
  const t = useT()
  // A recurring digest's template: back to the digest, and its own actions.
  // One of its issues: back to the digest too, with the issue's state and Check this issue.
  const digest = isDigestTemplate(campaign)
  const issue = isDigestIssue(campaign)
  const word = digest ? t(dc.template) : campaign.status
  const saveTrouble = status.kind === 'offline' || status.kind === 'error' || status.kind === 'conflict'
  const blocks = c.checks?.issues.filter((i) => i.severity === 'block' && i.group !== 'Subject').length ?? 0
  const empty = c.isEmpty
  const preview = view === 'preview'
  const reviewLabel = campaign.access.canSend ? t(m.reviewSend) : t(m.requestApproval)
  return (
    <header className="cmp-head">
      {preview ? (
        <button type="button" className="btn ic" aria-label={t(m.backToEditor)} onClick={() => c.setView('visual')}>
          <ToolIcon name="back" size={16} />
        </button>
      ) : digest || issue ? (
        <Link to="/w/$workspace/campaigns/digests/$digestId" params={{ workspace: slug, digestId: campaign.digestId! }} className="btn ic" aria-label={t(dc.backToDigest, { name: c.name })}>
          <ToolIcon name="back" size={16} />
        </Link>
      ) : (
        <Link to="/w/$workspace/campaigns" params={{ workspace: slug }} className="btn ic" aria-label={t(m.backToCampaigns)}>
          <ToolIcon name="back" size={16} />
        </Link>
      )}
      <div className="flex min-w-0 flex-col">
        {preview || view === 'mjml' || view === 'html' || !canEdit ? (
          <>
            <span className="k" style={{ fontSize: 11 }}>
              {preview ? t(m.previewLabel) : digest ? t(dc.template) : issue ? <IssueHeaderLabel fallback={t(m.campaignLabel)} /> : t(m.campaignLabel)}
            </span>
            <span className="text-[15px] font-bold">{c.name || t(m.untitled)}</span>
          </>
        ) : (
          <>
            <label className="k" htmlFor="cname" style={{ fontSize: 11 }}>
              {digest ? t(dc.template) : issue ? <IssueHeaderLabel fallback={t(m.campaignLabel)} /> : t(m.campaignLabel)}
            </label>
            <input
              id="cname"
              className={empty ? 'cmp-name wide' : 'cmp-name'}
              value={c.name}
              placeholder={t(m.untitled)}
              onChange={(e) => c.setName(e.target.value)}
              onBlur={() => c.commitName()}
              onKeyDown={(e) => {
                if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
                if (e.key === 'Escape') {
                  c.setName(campaign.name)
                  ;(e.target as HTMLInputElement).blur()
                }
              }}
              maxLength={200}
              aria-describedby="cstatus"
            />
          </>
        )}
      </div>
      <DigestBadge />
      {campaign.audience.paid === 'paid' ? <PaidOnlyBadge /> : null}
      {digest && !canEdit && !preview ? (
        <Badge id="cstatus" tone="neutral" role="status">
          {t(dc.readOnly)}
        </Badge>
      ) : issue && !preview && !saveTrouble ? (
        <span id="cstatus" role="status" className="contents">
          <IssueHeaderStatus />
        </span>
      ) : (
        <Badge id="cstatus" tone={status.kind === 'offline' || status.kind === 'error' ? 'bad' : status.kind === 'conflict' ? 'warn' : 'neutral'} role="status">
          {withoutVersion(
            preview ? t(m.statusVersion, { status: statusWord(word), version: status.kind === 'new' ? 1 : status.version }) : statusText(status, word),
            Boolean(versionLanguage(campaign) || c.missingLanguage),
          )}
        </Badge>
      )}
      {preview || digest || issue ? null : <ComposerLanguageSwitcher />}
      {preview ? null : (
        <div className="flex flex-wrap gap-2" style={{ marginLeft: 12 }}>
          <AudiencePicker />
          <SenderPicker />
          <RouteButton />
          <WebChip />
        </div>
      )}
      <div className="flex flex-wrap items-center gap-2" style={{ marginLeft: 'auto' }}>
        {preview ? (
          // The preview fills this with "Preview as", the test button and Review & send.
          <PreviewAsSlot />
        ) : digest ? (
          <DigestHeaderActions />
        ) : issue ? (
          <>
            <SendTestButton disabled={empty} />
            <button type="button" className={`btn pri${blocks ? ' bad-pri' : ''}`} disabled={empty} onClick={onReview}>
              {t(dc.checkThis)}
            </button>
          </>
        ) : (
          <>
            <AssistMenu />
            <button type="button" className="btn" disabled={empty} style={empty ? { opacity: 0.5 } : undefined} onClick={() => c.setView('preview')}>
              {t(m.preview)}
            </button>
            <SendTestButton disabled={empty} />
            <button
              type="button"
              className={`btn pri${blocks ? ' bad-pri' : ''}`}
              disabled={empty}
              style={empty ? { opacity: 0.5 } : undefined}
              aria-describedby={campaign.access.canSend ? undefined : 'perm-hint'}
              onClick={onReview}
            >
              {blocks ? t(m.reviewBlocks, { action: campaign.access.canSend ? 'send' : 'approval', count: blocks }) : reviewLabel}
            </button>
          </>
        )}
      </div>
    </header>
  )
}

/** Filled by the preview view (the "Preview as" select lives in the header). */
export function PreviewAsSlot() {
  const { setPreviewSlot } = useComposer()
  return <div ref={setPreviewSlot} className="contents" />
}

/** What a member without campaigns:send can do here (A5 §6, Content Writer). */
export function PermissionBanner() {
  const { campaign, slug } = useComposer()
  const { me } = useWorkspace()
  const { data } = useMe()
  const t = useT()
  // With the assistant (AIComposer): "You can edit this campaign and use the assistant."
  const assisted = useAssistBanner()
  const user = data?.user
  if (!user) return null
  const helpers = campaign.access.sendHelpers
  return (
    <div role="status" className="cmp-perm">
      <Avatar name={user.name || user.email} color={user.avatarColor} size={24} />
      <span id="perm-hint" style={{ flex: '1 1 480px', minWidth: 0 }}>
        {assisted ? (
          assisted({ name: user.name || user.email, role: roleName({ key: me.workspace.roleKey, name: me.workspace.roleName }), helpers })
        ) : (
          <T
            msg={m.permissionBanner}
            values={{
              name: user.name || user.email,
              role: roleName({ key: me.workspace.roleKey, name: me.workspace.roleName }),
              hasHelpers: helpers.length ? 'yes' : 'no',
              helpers: orList(helpers),
              routing: campaign.access.canManageRouting ? 'no' : 'yes',
              b: (chunks) => <b>{chunks}</b>,
              mono: (chunks) => <span className="mono">{chunks}</span>,
            }}
          />
        )}
      </span>
      <Link
        to="/w/$workspace/campaigns/$campaignId/review"
        params={{ workspace: slug, campaignId: campaign.id }}
        style={{ marginLeft: 'auto', fontWeight: 600, minHeight: 24, display: 'inline-flex', alignItems: 'center' }}
      >
        {t(m.whatApprovalCovers)}
      </Link>
    </div>
  )
}
