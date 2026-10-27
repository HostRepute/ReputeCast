import { Link } from '@tanstack/react-router'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { clock, orList } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { useComposer, type SaveStatus } from './context'
import { statusWord } from './format'
import { ToolIcon } from './icons'
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
import { PresenceChip } from './Collab'
import { mh } from './chrome.messages'

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

export function RouteButton() {
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

/** The save state as the top bar's chip: "✓ Saved 09:38 · v14" (Composer board). */
export function StatusChip({ word, draft }: { word: string; draft: boolean }) {
  const c = useComposer()
  const t = useT()
  const { status, campaign } = c
  const versioned = Boolean(versionLanguage(campaign) || c.missingLanguage)
  const args = { draft: draft ? 'yes' : 'no', status: statusWord(word) }
  let text: string
  let tone: 'ok' | 'warn' | 'bad' | 'neutral' = 'neutral'
  switch (status.kind) {
    case 'saved':
      text = t(mh.saved, { ...args, time: clock(status.at), version: status.version })
      tone = 'ok'
      break
    case 'dirty':
      text = status.at ? t(mh.saved, { ...args, time: clock(status.at), version: status.version }) : t(mh.editing, { ...args, version: status.version })
      break
    case 'saving':
      text = t(mh.saving, { ...args, version: status.version })
      break
    default:
      text = statusText(status, word)
      tone = status.kind === 'offline' || status.kind === 'error' ? 'bad' : status.kind === 'conflict' ? 'warn' : 'neutral'
  }
  return (
    <span id="cstatus" role="status" className={`stchip${tone === 'bad' ? ' bad' : tone === 'warn' ? ' warn' : ''}`}>
      {tone === 'ok' ? <ToolIcon name="check" size={12} strokeWidth={2.2} /> : null}
      {withoutVersion(text, versioned)}
    </span>
  )
}

const MODES = ['write', 'visual', 'code'] as const

/** Write · Design · Code (Composer and ComposerWrite boards): arrow keys move between them. */
export function ModeTabs() {
  const c = useComposer()
  const t = useT()
  const current = c.view === 'write' ? 'write' : c.view === 'mjml' || c.view === 'html' ? 'code' : 'visual'
  const label = (m: (typeof MODES)[number]) => (m === 'write' ? t(mh.modeWrite) : m === 'visual' ? t(mh.modeDesign) : t(mh.modeCode))
  const go = (m: (typeof MODES)[number]) => c.setView(m === 'code' ? 'mjml' : m)
  return (
    <div className="modes" role="tablist" aria-label={t(mh.modeGroup)}>
      {MODES.map((m) => (
        <button
          key={m}
          type="button"
          role="tab"
          id={`mode-${m}`}
          aria-selected={current === m}
          tabIndex={current === m ? 0 : -1}
          onClick={() => go(m)}
          onKeyDown={(e) => {
            const i = MODES.indexOf(m)
            const next = e.key === 'ArrowRight' ? MODES[(i + 1) % MODES.length] : e.key === 'ArrowLeft' ? MODES[(i + MODES.length - 1) % MODES.length] : e.key === 'Home' ? MODES[0] : e.key === 'End' ? MODES[MODES.length - 1] : null
            if (!next) return
            e.preventDefault()
            go(next)
            window.requestAnimationFrame(() => document.getElementById(`mode-${next}`)?.focus())
          }}
        >
          {label(m)}
        </button>
      ))}
    </div>
  )
}

/** Desktop · Mobile: the canvas's width (it only applies in Design mode). */
export function DeviceToggle() {
  const c = useComposer()
  const t = useT()
  const design = c.view === 'visual'
  return (
    <div className="devt" role="group" aria-label={design ? t(mh.widthGroup) : t(mh.widthGroupWrite)}>
      {(['desktop', 'mobile'] as const).map((d) => {
        const name = d === 'desktop' ? t(mh.desktop) : t(mh.mobile)
        return (
          <button
            key={d}
            type="button"
            aria-pressed={c.device === d}
            aria-disabled={design ? undefined : true}
            aria-label={design ? name : t(mh.widthInDesign, { device: name })}
            title={design ? (d === 'desktop' ? t(mh.desktopTitle) : t(mh.mobileTitle)) : t(mh.widthInDesignTitle)}
            onClick={() => design && c.setDevice(d)}
          >
            <ToolIcon name={d} size={16} strokeWidth={1.6} />
          </button>
        )
      })}
    </div>
  )
}

/** Blocks and warnings the checks found (dismissed warnings aside), A/B variants included. */
export function useReadiness(): { blocks: number; warnings: number } {
  const c = useComposer()
  const issues = c.checks?.issues ?? []
  const dismissed = new Set((c.checks?.dismissed ?? []).map((d) => d.index))
  let blocks = 0
  let warnings = 0
  issues.forEach((i, n) => {
    if (i.severity === 'block') blocks++
    else if (i.severity === 'warn' && !dismissed.has(n)) warnings++
  })
  for (const v of c.checks?.variants ?? []) {
    blocks += v.blocks
    warnings += v.warnings
  }
  return { blocks, warnings }
}

/** The composer's top bar (Composer board): the campaign, Write · Design · Code, the canvas width, and the actions. */
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
  const ready = useReadiness()
  const empty = c.isEmpty
  const preview = view === 'preview'
  const action = campaign.access.canSend ? 'send' : 'approval'
  const nameReadOnly = preview || view === 'mjml' || view === 'html' || !canEdit
  return (
    <header className={`cbar${preview ? ' is-preview' : ''}`}>
      <div className="cb-left">
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
        <div className="cb-id">
          {nameReadOnly ? (
            <>
              <span className={preview || digest || issue ? 'k cb-kind' : 'sr-only'}>
                {preview ? t(m.previewLabel) : digest ? t(dc.template) : issue ? <IssueHeaderLabel fallback={t(m.campaignLabel)} /> : t(m.campaignLabel)}
              </span>
              <span className="cb-name">{c.name || t(m.untitled)}</span>
            </>
          ) : (
            <>
              <label className={digest || issue ? 'k cb-kind' : 'sr-only'} htmlFor="cname">
                {digest ? t(dc.template) : issue ? <IssueHeaderLabel fallback={t(m.campaignLabel)} /> : t(m.campaignName)}
              </label>
              <input
                id="cname"
                className={empty ? 'cmp-name cb-name wide' : 'cmp-name cb-name'}
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
          {digest && !canEdit && !preview ? (
            <Badge id="cstatus" tone="neutral" role="status">
              {t(dc.readOnly)}
            </Badge>
          ) : issue && !preview && !saveTrouble ? (
            <span id="cstatus" role="status" className="contents">
              <IssueHeaderStatus />
            </span>
          ) : preview ? (
            <span id="cstatus" role="status" className="stchip">
              {withoutVersion(t(m.statusVersion, { status: statusWord(word), version: status.kind === 'new' ? 1 : status.version }), Boolean(versionLanguage(campaign) || c.missingLanguage))}
            </span>
          ) : (
            <StatusChip word={word} draft={campaign.status === 'draft' && !digest} />
          )}
        </div>
        <DigestBadge />
        {campaign.audience.paid === 'paid' ? <PaidOnlyBadge /> : null}
        {/* Who else has it open (CO-10). */}
        {preview ? null : <PresenceChip />}
      </div>
      {preview ? null : (
        <div className="cb-mid">
          <ModeTabs />
          <DeviceToggle />
        </div>
      )}
      <div className="cb-end">
        {preview || digest || issue ? null : <ComposerLanguageSwitcher />}
        <WebChip />
        {preview ? (
          // The preview fills this with "Preview as", the test button and Review & send.
          <PreviewAsSlot />
        ) : digest ? (
          <DigestHeaderActions />
        ) : issue ? (
          <>
            <SendTestButton disabled={empty} />
            <button type="button" className={`btn pri${ready.blocks ? ' bad-pri' : ''}`} disabled={empty} onClick={onReview}>
              {t(dc.checkThis)}
            </button>
          </>
        ) : (
          <>
            <AssistMenu />
            <button type="button" className="btn" disabled={empty} style={empty ? { opacity: 0.5 } : undefined} aria-label={t(m.preview)} onClick={() => c.setView('preview')}>
              <span className="bi" aria-hidden="true">
                <ToolIcon name="eye" size={15} strokeWidth={1.6} />
              </span>
              <span className="bt">{t(m.preview)}</span>
            </button>
            <span className="hide-ds">
              <SendTestButton disabled={empty} />
            </span>
          </>
        )}
      </div>
      {preview || digest || issue ? null : (
        <div className="cb-go">
          <button
            type="button"
            className={`btn pri rsend${ready.blocks ? ' bad-pri' : ''}`}
            disabled={empty}
            style={empty ? { opacity: 0.5 } : undefined}
            aria-label={t(mh.readinessLabel, { action, blocks: ready.blocks, warnings: ready.warnings })}
            aria-describedby={campaign.access.canSend ? undefined : 'perm-hint'}
            onClick={onReview}
          >
            <span className="rs-a">{t(mh.readinessAction, { action })}</span>
            {ready.blocks || ready.warnings ? (
              <>
                <span className="rs-dot" aria-hidden="true">
                  ·
                </span>
                <span className="rs-b">{t(mh.readinessCount, { blocks: ready.blocks, warnings: ready.warnings })}</span>
              </>
            ) : null}
          </button>
        </div>
      )}
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
