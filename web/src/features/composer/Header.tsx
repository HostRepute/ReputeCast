import { Link } from '@tanstack/react-router'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { clock } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { useComposer, type SaveStatus } from './context'
import { statusWord } from './format'
import { ToolIcon } from './icons'
import { AudiencePicker, SenderPicker } from './Pickers'
import { SendTestButton } from './SendTest'
import { ComposerLanguageSwitcher, versionLanguage, withoutVersion } from '@/features/languages/composerParts'
import { DigestBadge, DigestHeaderActions, isDigestTemplate } from '@/features/digests/composer/DigestMode'
import { IssueHeaderLabel, IssueHeaderStatus, isDigestIssue } from '@/features/digests/composer/IssueMode'
import { dc } from '@/features/digests/composer/messages'
import { useT } from '@/i18n'
import { useAssistBanner } from '@/features/ai/composer/panel'
import { AssistMenu } from '@/features/ai/composer/assistMenu'
import { WebChip } from '@/features/website/composer/campaign'

const routeColors: Record<string, string> = { ses: 'var(--chart-ses)', postal: 'var(--chart-postal)', postmark: 'var(--chart-postmark)' }

export function statusText(status: SaveStatus, campaignStatus: string): string {
  const word = statusWord(campaignStatus)
  switch (status.kind) {
    case 'new':
      return 'New draft · not saved yet'
    case 'saving':
      return `${word} · saving… · v${status.version}`
    case 'offline':
      return `${word} · not saved · v${status.version}`
    case 'conflict':
      return `${word} · conflict · v${status.version}`
    case 'error':
      return `${word} · not saved · v${status.version}`
    case 'dirty':
      return status.at ? `${word} · autosaved ${clock(status.at)} · v${status.version}` : `${word} · editing · v${status.version}`
    default:
      return `${word} · autosaved ${clock(status.at)} · v${status.version}`
  }
}

function RouteButton() {
  const { campaign, context, slug } = useComposer()
  // A campaign without its own split follows the Broadcasts routing rule.
  const fromRule = campaign.route.source !== 'campaign'
  const route = campaign.route.shares.length ? campaign.route : context.broadcastRoute
  // The words the button shows, so its accessible name starts with them (WCAG 2.5.3).
  const visible = fromRule
    ? route.shares.length
      ? `Broadcasts route · ${route.shares.map((s) => `${s.short} ${Math.round(s.percent)}`).join(' / ')}`
      : 'Broadcasts route · not set up'
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
      <span className="btn" aria-disabled="true" title="Routing needs providers:manage" style={{ opacity: 0.7, cursor: 'default' }}>
        {label}
        {fromRule ? null : ' · read-only'}
      </span>
    )
  }
  return (
    <Link to="/w/$workspace/providers/routing" params={{ workspace: slug }} className="btn" aria-label={`${visible}: open routing`}>
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
  const reviewLabel = campaign.access.canSend ? 'Review & send' : 'Request approval'
  return (
    <header className="cmp-head">
      {preview ? (
        <button type="button" className="btn ic" aria-label="Back to editor" onClick={() => c.setView('visual')}>
          <ToolIcon name="back" size={16} />
        </button>
      ) : digest || issue ? (
        <Link to="/w/$workspace/campaigns/digests/$digestId" params={{ workspace: slug, digestId: campaign.digestId! }} className="btn ic" aria-label={t(dc.backToDigest, { name: c.name })}>
          <ToolIcon name="back" size={16} />
        </Link>
      ) : (
        <Link to="/w/$workspace/campaigns" params={{ workspace: slug }} className="btn ic" aria-label="Back to campaigns">
          <ToolIcon name="back" size={16} />
        </Link>
      )}
      <div className="flex min-w-0 flex-col">
        {preview || view === 'mjml' || view === 'html' || !canEdit ? (
          <>
            <span className="k" style={{ fontSize: 11 }}>
              {preview ? 'Preview' : digest ? t(dc.template) : issue ? <IssueHeaderLabel fallback="Campaign" /> : 'Campaign'}
            </span>
            <span className="text-[15px] font-bold">{c.name || 'Untitled campaign'}</span>
          </>
        ) : (
          <>
            <label className="k" htmlFor="cname" style={{ fontSize: 11 }}>
              {digest ? t(dc.template) : issue ? <IssueHeaderLabel fallback="Campaign" /> : 'Campaign'}
            </label>
            <input
              id="cname"
              className={empty ? 'cmp-name wide' : 'cmp-name'}
              value={c.name}
              placeholder="Untitled campaign"
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
            preview ? `${statusWord(word)} · v${status.kind === 'new' ? 1 : status.version}` : statusText(status, word),
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
              Preview
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
              {reviewLabel}
              {blocks ? ` · ${blocks} ${blocks === 1 ? 'block' : 'blocks'}` : ''}
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
  // With the assistant (AIComposer): "You can edit this campaign and use the assistant."
  const assisted = useAssistBanner()
  const user = data?.user
  if (!user) return null
  const helpers = campaign.access.sendHelpers
  const hidden: string[] = ['Scheduling', 'test sends to the whole team']
  if (!campaign.access.canManageRouting) hidden.push('route changes')
  const ask = helpers.length ? `: ask ${helpers.length === 1 ? helpers[0] : `${helpers.slice(0, -1).join(', ')} or ${helpers[helpers.length - 1]}`}` : ''
  const list = hidden.length > 1 ? `${hidden.slice(0, -1).join(', ')} and ${hidden[hidden.length - 1]}` : hidden[0]
  return (
    <div role="status" className="cmp-perm">
      <Avatar name={user.name || user.email} color={user.avatarColor} size={24} />
      <span id="perm-hint" style={{ flex: '1 1 480px', minWidth: 0 }}>
        {assisted ? (
          assisted({ name: user.name || user.email, role: me.workspace.roleName, helpers })
        ) : (
          <>
            <b>
              {user.name || user.email} · {me.workspace.roleName}.
            </b>{' '}
            You can edit this campaign and request approval. Sending needs the <span className="mono">campaigns:send</span> permission{ask}. {list} are hidden for your role.
          </>
        )}
      </span>
      <Link
        to="/w/$workspace/campaigns/$campaignId/review"
        params={{ workspace: slug, campaignId: campaign.id }}
        style={{ marginLeft: 'auto', fontWeight: 600, minHeight: 24, display: 'inline-flex', alignItems: 'center' }}
      >
        What approval covers
      </Link>
    </div>
  )
}
